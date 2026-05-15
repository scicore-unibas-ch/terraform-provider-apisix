#!/usr/bin/env python3
"""
APISIX Resource Importer.

Discovers existing resources in a running APISIX instance via the Admin API
and emits Terraform/OpenTofu .tf files plus an import.sh script that maps the
APISIX objects into Terraform state.

The HCL emitted here matches the apisix provider's 0.2+ Plugin Framework
schema:
  - id is the explicit URL key on every resource (Required + RequiresReplace).
  - Nested objects use attribute syntax: nodes = [{...}], timeout = {...},
    upstream = {...}, keepalive_pool = {...}, tls = {...}.
  - Plugin maps emit jsonencode({...}) values for readability.

apisix_ssl is not implemented in the provider yet, so SSL objects are listed
in the summary but not emitted as Terraform resources.

The HCL is emitted in a straightforward shape that is correct but not
canonically aligned. Run `tofu fmt -recursive <output-dir>` (or
`terraform fmt -recursive`) on the output before committing — this script
will auto-run it when a `tofu` or `terraform` binary is found on PATH.
"""

import argparse
import json
import os
import re
import shutil
import subprocess
import sys
from datetime import datetime
from urllib.error import HTTPError, URLError
from urllib.request import Request, urlopen

# Resource type → APISIX endpoint and the field that carries the URL key
# in the response. The provider exposes that key under the Terraform
# attribute name `id` for every resource (the consumer's `username` is
# mapped to `id` by the provider).
RESOURCE_TYPES = {
    "upstream":       {"endpoint": "upstreams",       "id_field": "id"},
    "route":          {"endpoint": "routes",          "id_field": "id"},
    "service":        {"endpoint": "services",        "id_field": "id"},
    "consumer":       {"endpoint": "consumers",       "id_field": "username"},
    "consumer_group": {"endpoint": "consumer_groups", "id_field": "id"},
    "plugin_config":  {"endpoint": "plugin_configs",  "id_field": "id"},
    "global_rule":    {"endpoint": "global_rules",    "id_field": "id"},
}

# APISIX resources that are known but not yet supported by the Terraform
# provider. They are included in the discovery summary so users see what's
# present in their cluster, but no .tf is emitted.
UNSUPPORTED_RESOURCE_TYPES = {
    "ssl": {"endpoint": "ssl"},
}

# HCL identifier regex used to decide whether a map key needs quoting.
_IDENTIFIER_RE = re.compile(r"^[A-Za-z_][A-Za-z0-9_-]*$")


def _hcl_string(value):
    """Return value formatted as a double-quoted HCL string."""
    s = str(value)
    s = s.replace("\\", "\\\\").replace('"', '\\"').replace("\n", "\\n")
    return f'"{s}"'


def _format_json_value(value, indent_level=2):
    """Format a Python value as the body of a jsonencode({...}) call."""
    indent = "    " * indent_level
    next_indent = "    " * (indent_level + 1)

    if isinstance(value, bool):
        return str(value).lower()
    if isinstance(value, (int, float)):
        return str(value)
    if isinstance(value, str):
        return _hcl_string(value)
    if isinstance(value, list):
        if not value:
            return "[]"
        lines = ["["]
        for item in value:
            lines.append(f"{next_indent}{_format_json_value(item, indent_level + 1)},")
        lines.append(f"{indent}]")
        return "\n".join(lines)
    if isinstance(value, dict):
        if not value:
            return "{}"
        lines = ["{"]
        for k, v in value.items():
            key = k if _IDENTIFIER_RE.match(k) else _hcl_string(k)
            lines.append(f"{next_indent}{key} = {_format_json_value(v, indent_level + 1)},")
        lines.append(f"{indent}}}")
        return "\n".join(lines)
    return _hcl_string(str(value))


def get_latest_provider_version(provider_source):
    """Get latest provider version from the OpenTofu Registry API."""
    url = f"https://registry.opentofu.org/v1/providers/{provider_source}/versions"
    print(f"📦 Fetching latest version from OpenTofu Registry: {url}")
    try:
        req = Request(url)
        req.add_header("Accept", "application/json")
        with urlopen(req, timeout=10) as response:
            data = json.loads(response.read().decode())
            versions = data.get("versions", [])
            if versions:
                version = versions[0]["version"]
                print(f"✓ Found latest version: {version}")
                return version
            print("⚠ No versions found in OpenTofu Registry")
            return None
    except HTTPError as e:
        if e.code == 404:
            print(f"⚠ Provider {provider_source} not found in OpenTofu Registry")
        else:
            print(f"⚠ OpenTofu Registry API error: {e.code} {e.reason}")
        return None
    except Exception as e:
        print(f"⚠ Error fetching version: {type(e).__name__}: {e}")
        return None


class APISIXClient:
    def __init__(self, base_url, admin_key):
        self.base_url = base_url.rstrip("/")
        self.admin_key = admin_key

    def _request(self, endpoint):
        url = f"{self.base_url}/{endpoint}"
        req = Request(url)
        req.add_header("X-API-KEY", self.admin_key)
        try:
            with urlopen(req) as response:
                return json.loads(response.read().decode())
        except HTTPError as e:
            if e.code == 404:
                return {"total": 0, "list": []}
            raise
        except URLError as e:
            print(f"Error connecting to APISIX: {e}")
            sys.exit(1)

    def list_resources(self, endpoint):
        response = self._request(endpoint)
        items = response.get("list", [])
        return [item.get("value", item) for item in items]


class ResourceGenerator:
    def __init__(self, client, provider_source, provider_version):
        self.client = client
        self.provider_source = provider_source
        self.provider_version = provider_version
        self.resources = {}
        self.unsupported = {}

    # ----- discovery -----

    def discover_all(self):
        print("🔍 Discovering APISIX resources...")
        for resource_type, config in RESOURCE_TYPES.items():
            print(f"  Scanning {resource_type}...")
            items = self.client.list_resources(config["endpoint"])
            self.resources[resource_type] = items
            print(f"    Found {len(items)} {resource_type}(s)")
        for resource_type, config in UNSUPPORTED_RESOURCE_TYPES.items():
            try:
                items = self.client.list_resources(config["endpoint"])
            except HTTPError:
                items = []
            self.unsupported[resource_type] = items
            if items:
                print(f"  ⚠ Found {len(items)} {resource_type}(s) — not yet supported by the provider, skipping")
        return self.resources

    def _get_resource_id(self, resource_type, item):
        config = RESOURCE_TYPES[resource_type]
        return str(item.get(config["id_field"]) or item.get("id") or "unknown")

    def _generate_resource_name(self, resource_id, resource_type):
        name = re.sub(r"[^a-zA-Z0-9_]", "_", resource_id)
        if not name or name[0].isdigit():
            name = "r_" + name
        return f"{name}_{resource_type}"

    # ----- HCL writers -----

    @staticmethod
    def _write_str(f, key, value, indent="  "):
        if value is None or value == "":
            return
        f.write(f"{indent}{key} = {_hcl_string(value)}\n")

    @staticmethod
    def _write_int(f, key, value, indent="  "):
        if value is None:
            return
        f.write(f"{indent}{key} = {int(value)}\n")

    @staticmethod
    def _write_bool(f, key, value, indent="  "):
        if value is None:
            return
        f.write(f"{indent}{key} = {str(bool(value)).lower()}\n")

    @staticmethod
    def _write_string_list(f, key, value, indent="  "):
        if not value or not isinstance(value, list):
            return
        f.write(f"{indent}{key} = [\n")
        for v in value:
            f.write(f"{indent}  {_hcl_string(v)},\n")
        f.write(f"{indent}]\n")

    @staticmethod
    def _write_string_map(f, key, value, indent="  "):
        if not value or not isinstance(value, dict):
            return
        f.write(f"{indent}{key} = {{\n")
        for k, v in value.items():
            map_key = k if _IDENTIFIER_RE.match(k) else _hcl_string(k)
            f.write(f"{indent}  {map_key} = {_hcl_string(v)}\n")
        f.write(f"{indent}}}\n")

    @staticmethod
    def _write_plugins(f, plugins, indent="  "):
        if not plugins or not isinstance(plugins, dict):
            return
        f.write(f"{indent}plugins = {{\n")
        for pname, pconf in plugins.items():
            f.write(f'{indent}  "{pname}" = jsonencode({_format_json_value(pconf, 2)})\n')
        f.write(f"{indent}}}\n")

    @staticmethod
    def _write_json_string(f, key, value, indent="  "):
        """Write a field as jsonencode({...}) — used for vars / health_check."""
        if value is None or value == "":
            return
        f.write(f"{indent}{key} = jsonencode({_format_json_value(value, 1)})\n")

    # ----- per-resource writers -----

    def _write_upstream_attrs(self, f, item):
        self._write_str(f, "name",          item.get("name"))
        self._write_str(f, "desc",          item.get("desc"))
        self._write_str(f, "type",          item.get("type"))
        self._write_str(f, "scheme",        item.get("scheme"))
        self._write_str(f, "hash_on",       item.get("hash_on"))
        self._write_str(f, "key",           item.get("key"))
        self._write_str(f, "pass_host",     item.get("pass_host"))
        self._write_str(f, "upstream_host", item.get("upstream_host"))
        self._write_str(f, "service_name",  item.get("service_name"))
        self._write_str(f, "discovery_type", item.get("discovery_type"))
        self._write_int(f, "retries",       item.get("retries"))
        self._write_int(f, "retry_timeout", item.get("retry_timeout"))
        self._write_string_map(f, "discovery_args", item.get("discovery_args"))
        self._write_string_map(f, "labels",         item.get("labels"))
        self._write_nodes(f, item.get("nodes"))
        self._write_timeout(f, item.get("timeout"))
        self._write_keepalive(f, item.get("keepalive_pool"))
        self._write_tls(f, item.get("tls"))
        self._write_json_string(f, "health_check", item.get("checks"))

    def _write_route_attrs(self, f, item):
        self._write_str(f, "name", item.get("name"))
        self._write_str(f, "desc", item.get("desc"))
        # uri/uris and host/hosts are mutually exclusive in the schema; emit
        # whichever the API returned.
        self._write_str(f, "uri",          item.get("uri"))
        self._write_string_list(f, "uris", item.get("uris"))
        self._write_str(f, "host",         item.get("host"))
        self._write_string_list(f, "hosts", item.get("hosts"))
        self._write_str(f, "remote_addr",  item.get("remote_addr"))
        self._write_string_list(f, "remote_addrs", item.get("remote_addrs"))
        self._write_string_list(f, "methods", item.get("methods"))
        self._write_int(f, "priority", item.get("priority"))
        self._write_int(f, "status",   item.get("status"))
        self._write_bool(f, "enable_websocket", item.get("enable_websocket"))
        self._write_str(f, "filter_func", item.get("filter_func"))
        self._write_str(f, "script",      item.get("script"))
        self._write_str(f, "upstream_id", item.get("upstream_id"))
        self._write_str(f, "service_id",  item.get("service_id"))
        self._write_str(f, "plugin_config_id", item.get("plugin_config_id"))
        self._write_string_map(f, "labels", item.get("labels"))
        self._write_json_string(f, "vars",  item.get("vars"))
        self._write_plugins(f, item.get("plugins"))
        self._write_route_timeout(f, item.get("timeout"))
        self._write_inline_upstream(f, item.get("upstream"))

    def _write_service_attrs(self, f, item):
        self._write_str(f, "name", item.get("name"))
        self._write_str(f, "desc", item.get("desc"))
        self._write_string_list(f, "hosts", item.get("hosts"))
        self._write_str(f, "script",      item.get("script"))
        self._write_str(f, "upstream_id", item.get("upstream_id"))
        self._write_bool(f, "enable_websocket", item.get("enable_websocket"))
        self._write_string_map(f, "labels", item.get("labels"))
        self._write_plugins(f, item.get("plugins"))
        self._write_inline_upstream(f, item.get("upstream"))

    def _write_consumer_attrs(self, f, item):
        self._write_str(f, "group_id", item.get("group_id"))
        self._write_str(f, "desc",     item.get("desc"))
        self._write_plugins(f, item.get("plugins"))
        self._write_string_map(f, "labels", item.get("labels"))

    def _write_consumer_group_attrs(self, f, item):
        self._write_str(f, "name", item.get("name"))
        self._write_str(f, "desc", item.get("desc"))
        self._write_plugins(f, item.get("plugins"))
        self._write_string_map(f, "labels", item.get("labels"))

    def _write_plugin_config_attrs(self, f, item):
        self._write_str(f, "desc", item.get("desc"))
        self._write_plugins(f, item.get("plugins"))
        self._write_string_map(f, "labels", item.get("labels"))

    def _write_global_rule_attrs(self, f, item):
        # Schema is just id + plugins.
        self._write_plugins(f, item.get("plugins"))

    # ----- nested-attribute writers -----

    def _write_nodes(self, f, nodes):
        if not nodes or not isinstance(nodes, list):
            return
        f.write("  nodes = [\n")
        for node in nodes:
            f.write("    {\n")
            self._write_str(f, "host",     node.get("host"),     indent="      ")
            self._write_int(f, "port",     node.get("port"),     indent="      ")
            self._write_int(f, "weight",   node.get("weight"),   indent="      ")
            self._write_int(f, "priority", node.get("priority"), indent="      ")
            self._write_string_map(f, "metadata", node.get("metadata"), indent="      ")
            f.write("    },\n")
        f.write("  ]\n")

    def _write_timeout(self, f, timeout):
        if not timeout or not isinstance(timeout, dict):
            return
        f.write("  timeout = {\n")
        self._write_int(f, "connect", timeout.get("connect"), indent="    ")
        self._write_int(f, "send",    timeout.get("send"),    indent="    ")
        self._write_int(f, "read",    timeout.get("read"),    indent="    ")
        f.write("  }\n")

    # Routes use the same shape; aliased for clarity.
    _write_route_timeout = _write_timeout

    def _write_keepalive(self, f, kp):
        if not kp or not isinstance(kp, dict):
            return
        f.write("  keepalive_pool = {\n")
        self._write_int(f, "size",         kp.get("size"),         indent="    ")
        self._write_int(f, "idle_timeout", kp.get("idle_timeout"), indent="    ")
        self._write_int(f, "requests",     kp.get("requests"),     indent="    ")
        f.write("  }\n")

    def _write_tls(self, f, tls):
        if not tls or not isinstance(tls, dict):
            return
        f.write("  tls = {\n")
        # Note: cert/key may appear in API output if APISIX is configured to
        # echo them; emit so the imported config is round-trippable. Operators
        # should review and optionally move these to a secrets store.
        self._write_str(f, "client_cert",    tls.get("client_cert"),    indent="    ")
        self._write_str(f, "client_key",     tls.get("client_key"),     indent="    ")
        self._write_str(f, "client_cert_id", tls.get("client_cert_id"), indent="    ")
        self._write_bool(f, "verify",        tls.get("verify"),         indent="    ")
        f.write("  }\n")

    def _write_inline_upstream(self, f, upstream):
        """Emit a SingleNestedAttribute upstream block for route/service."""
        if not upstream or not isinstance(upstream, dict):
            return
        f.write("  upstream = {\n")
        for key in ("name", "desc", "type", "scheme", "hash_on", "key",
                    "pass_host", "upstream_host", "service_name", "discovery_type"):
            self._write_str(f, key, upstream.get(key), indent="    ")
        self._write_int(f, "retries",       upstream.get("retries"),       indent="    ")
        self._write_int(f, "retry_timeout", upstream.get("retry_timeout"), indent="    ")
        self._write_string_map(f, "discovery_args", upstream.get("discovery_args"), indent="    ")
        self._write_string_map(f, "labels",         upstream.get("labels"),         indent="    ")
        # Inner nested objects use the same shape, but with deeper indent.
        if isinstance(upstream.get("nodes"), list):
            f.write("    nodes = [\n")
            for node in upstream["nodes"]:
                f.write("      {\n")
                self._write_str(f, "host",     node.get("host"),     indent="        ")
                self._write_int(f, "port",     node.get("port"),     indent="        ")
                self._write_int(f, "weight",   node.get("weight"),   indent="        ")
                self._write_int(f, "priority", node.get("priority"), indent="        ")
                self._write_string_map(f, "metadata", node.get("metadata"), indent="        ")
                f.write("      },\n")
            f.write("    ]\n")
        if isinstance(upstream.get("timeout"), dict):
            t = upstream["timeout"]
            f.write("    timeout = {\n")
            self._write_int(f, "connect", t.get("connect"), indent="      ")
            self._write_int(f, "send",    t.get("send"),    indent="      ")
            self._write_int(f, "read",    t.get("read"),    indent="      ")
            f.write("    }\n")
        if isinstance(upstream.get("keepalive_pool"), dict):
            kp = upstream["keepalive_pool"]
            f.write("    keepalive_pool = {\n")
            self._write_int(f, "size",         kp.get("size"),         indent="      ")
            self._write_int(f, "idle_timeout", kp.get("idle_timeout"), indent="      ")
            self._write_int(f, "requests",     kp.get("requests"),     indent="      ")
            f.write("    }\n")
        if isinstance(upstream.get("tls"), dict):
            tls = upstream["tls"]
            f.write("    tls = {\n")
            self._write_str(f, "client_cert",    tls.get("client_cert"),    indent="      ")
            self._write_str(f, "client_key",     tls.get("client_key"),     indent="      ")
            self._write_str(f, "client_cert_id", tls.get("client_cert_id"), indent="      ")
            self._write_bool(f, "verify",        tls.get("verify"),         indent="      ")
            f.write("    }\n")
        self._write_json_string(f, "health_check", upstream.get("checks"), indent="    ")
        f.write("  }\n")

    # ----- dispatch -----

    _RESOURCE_WRITERS = {
        "upstream":       "_write_upstream_attrs",
        "route":          "_write_route_attrs",
        "service":        "_write_service_attrs",
        "consumer":       "_write_consumer_attrs",
        "consumer_group": "_write_consumer_group_attrs",
        "plugin_config":  "_write_plugin_config_attrs",
        "global_rule":    "_write_global_rule_attrs",
    }

    def _write_resource_attributes(self, f, resource_type, item):
        # id is the URL key for every resource and must come first.
        resource_id = self._get_resource_id(resource_type, item)
        f.write(f'  id = {_hcl_string(resource_id)}\n')
        writer = getattr(self, self._RESOURCE_WRITERS[resource_type])
        writer(f, item)

    # ----- file generation -----

    def generate_separate_files(self, output_dir):
        print(f"\n📝 Generating separate HCL files in: {output_dir}")

        if self.provider_version:
            version_line = f'      version = "~> {self.provider_version}"\n'
            version_comment = f"# Provider version: {self.provider_version}\n"
        else:
            version_line = ""
            version_comment = "# Using latest available provider version\n"

        provider_file = os.path.join(output_dir, "provider.tf")
        with open(provider_file, "w") as f:
            f.write("# APISIX Provider Configuration\n")
            f.write(f"# Generated: {datetime.now().isoformat()}\n")
            f.write(version_comment)
            f.write(f"# Provider source: {self.provider_source}\n\n")
            f.write("terraform {\n")
            f.write("  required_providers {\n")
            f.write("    apisix = {\n")
            f.write(f'      source  = "{self.provider_source}"\n')
            if version_line:
                f.write(version_line)
            f.write("    }\n")
            f.write("  }\n")
            f.write("}\n\n")
            f.write('provider "apisix" {\n')
            f.write('  base_url  = "http://localhost:9180/apisix/admin"\n')
            f.write('  admin_key = "test123456789"\n')
            f.write("}\n")
        print("  ✅ Generated provider.tf")

        for resource_type, items in self.resources.items():
            if not items:
                print(f"  ⊘ Skipping {resource_type} (no resources)")
                continue
            filename = f"{resource_type}s.tf"
            filepath = os.path.join(output_dir, filename)
            with open(filepath, "w") as f:
                f.write(f"# APISIX {resource_type.replace('_', ' ').title()} Configuration\n")
                f.write(f"# Generated: {datetime.now().isoformat()}\n")
                f.write(f"# Total resources: {len(items)}\n\n")
                for item in items:
                    resource_id = self._get_resource_id(resource_type, item)
                    resource_name = self._generate_resource_name(resource_id, resource_type)
                    f.write(f'resource "apisix_{resource_type}" "{resource_name}" {{\n')
                    self._write_resource_attributes(f, resource_type, item)
                    f.write("}\n\n")
            print(f"  ✅ Generated {filename} ({len(items)} resources)")

        for resource_type, items in self.unsupported.items():
            if items:
                print(f"  ⚠ Skipped {len(items)} {resource_type}(s) — provider does not implement apisix_{resource_type} yet")

    def generate_import_script(self, output_file):
        print(f"\n📝 Generating import script: {output_file}")
        with open(output_file, "w") as f:
            f.write("#!/bin/bash\n")
            f.write("# APISIX Resource Import Script\n")
            f.write(f"# Generated: {datetime.now().isoformat()}\n\n")
            f.write("set -e\n\n")
            f.write('echo "🚀 Starting APISIX resource import..."\n')
            f.write('echo ""\n\n')
            for resource_type, items in self.resources.items():
                if not items:
                    continue
                f.write(f'echo "=== Importing {resource_type.replace("_", " ").title()} ==="\n')
                for item in items:
                    resource_id = self._get_resource_id(resource_type, item)
                    resource_name = self._generate_resource_name(resource_id, resource_type)
                    f.write(f'tofu import apisix_{resource_type}.{resource_name} "{resource_id}"\n')
                f.write("\n")
            f.write('echo ""\n')
            f.write('echo "✅ Import complete!"\n')
            f.write('echo ""\n')
            f.write('echo "Next steps:"\n')
            f.write('echo "  1. Review generated .tf files and customize as needed"\n')
            f.write('echo "  2. Run: tofu init"\n')
            f.write('echo "  3. Run: tofu plan"\n')
            f.write('echo "  4. Run: tofu apply"\n')
        os.chmod(output_file, 0o755)
        print("  ✅ Generated import.sh")

    def generate_readme(self, output_file):
        print(f"\n📝 Generating README: {output_file}")
        with open(output_file, "w") as f:
            f.write("# APISIX Import Results\n\n")
            f.write(f"Generated: {datetime.now().isoformat()}\n\n")
            f.write("## Provider\n\n")
            f.write(f"- **Source**: `{self.provider_source}`\n")
            if self.provider_version:
                f.write(f"- **Version**: {self.provider_version}\n")
            else:
                f.write("- **Version**: Latest available from registry\n")
            f.write("\n## Summary\n\n")
            total = sum(len(items) for items in self.resources.values())
            f.write(f"Total resources discovered: **{total}**\n\n")
            f.write("| Resource Type | Count | Generated File |\n")
            f.write("|---------------|------:|----------------|\n")
            for rtype, items in self.resources.items():
                filename = f"{rtype}s.tf" if items else "—"
                f.write(f"| {rtype.replace('_', ' ').title()} | {len(items)} | {filename} |\n")
            unsupported_total = sum(len(v) for v in self.unsupported.values())
            if unsupported_total:
                f.write("\n### Skipped (not yet implemented in the provider)\n\n")
                f.write("| Resource Type | Count |\n")
                f.write("|---------------|------:|\n")
                for rtype, items in self.unsupported.items():
                    f.write(f"| {rtype.replace('_', ' ').title()} | {len(items)} |\n")
            f.write("\n## Generated files\n\n")
            f.write("- `provider.tf` — provider configuration\n")
            for rtype, items in self.resources.items():
                if items:
                    f.write(f"- `{rtype}s.tf` — {len(items)} {rtype} resource(s)\n")
            f.write("- `import.sh` — import script\n")
            f.write("- `README.md` — this file\n\n")
            f.write("## Usage\n\n")
            f.write("```bash\n")
            f.write("# 1. Review and customize generated files\n")
            f.write("vim *.tf\n\n")
            f.write("# 2. Initialize Terraform/OpenTofu\n")
            f.write("tofu init\n\n")
            f.write("# 3. Run the import script\n")
            f.write("./import.sh\n\n")
            f.write("# 4. Verify the import\n")
            f.write("tofu plan\n\n")
            f.write("# 5. Apply (if needed)\n")
            f.write("tofu apply\n")
            f.write("```\n\n")
            f.write("## Notes\n\n")
            f.write("- Review all generated HCL — some values may need manual adjustment.\n")
            f.write("- Sensitive material (TLS keys, plugin secrets) is emitted as-is when APISIX returns it; secure it appropriately.\n")
            if unsupported_total:
                f.write("- SSL objects discovered above are **not** emitted as Terraform resources because the provider does not yet support `apisix_ssl`. They remain manageable via the APISIX Admin API directly.\n")
        print("  ✅ Generated README.md")


def run_fmt_if_available(output_dir):
    """Run `tofu fmt` (or `terraform fmt`) over the output if available."""
    binary = shutil.which("tofu") or shutil.which("terraform")
    if not binary:
        print("\nℹ Skipping HCL formatting: neither `tofu` nor `terraform` found on PATH.")
        print(f"  Recommended: run `tofu fmt -recursive {output_dir}` to canonicalize alignment.")
        return
    print(f"\n🧹 Running `{os.path.basename(binary)} fmt -recursive` on {output_dir}")
    try:
        subprocess.run(
            [binary, "fmt", "-recursive", output_dir],
            check=True,
            stdout=subprocess.DEVNULL,
        )
        print("  ✅ Formatted")
    except subprocess.CalledProcessError as e:
        print(f"  ⚠ fmt exited with status {e.returncode}; output was generated but is not formatted")


def main():
    parser = argparse.ArgumentParser(description="Import APISIX resources to Terraform/OpenTofu")
    parser.add_argument("--base-url",         default="http://localhost:9180/apisix/admin")
    parser.add_argument("--admin-key",        default="test123456789")
    parser.add_argument("--output-dir",       default="./import-output")
    parser.add_argument("--provider-version", default=None,
                        help="Provider version (default: fetch latest from the OpenTofu Registry)")
    parser.add_argument("--provider-source",  default="scicore-unibas-ch/apisix")

    args = parser.parse_args()

    provider_version = args.provider_version
    if not provider_version:
        provider_version = get_latest_provider_version(args.provider_source)
        if not provider_version:
            print("  Will not specify version constraint (OpenTofu will use latest)")

    os.makedirs(args.output_dir, exist_ok=True)
    print(f"🔌 Connecting to APISIX: {args.base_url}")

    client = APISIXClient(args.base_url, args.admin_key)
    generator = ResourceGenerator(client, args.provider_source, provider_version)
    generator.discover_all()
    generator.generate_separate_files(args.output_dir)
    generator.generate_import_script(os.path.join(args.output_dir, "import.sh"))
    generator.generate_readme(os.path.join(args.output_dir, "README.md"))
    run_fmt_if_available(args.output_dir)

    print(f"\n✅ Complete! Output: {os.path.abspath(args.output_dir)}")


if __name__ == "__main__":
    main()
