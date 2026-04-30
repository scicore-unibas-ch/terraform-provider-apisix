#!/usr/bin/env python3
"""
APISIX Resource Importer - Fetches version from GitHub Releases
"""

import argparse
import json
import os
import sys
import re
from urllib.request import Request, urlopen
from urllib.error import URLError, HTTPError
from datetime import datetime

RESOURCE_TYPES = {
    "upstream": {"endpoint": "upstreams", "id_field": "id", "name_field": "name"},
    "route": {"endpoint": "routes", "id_field": "id", "name_field": "name"},
    "service": {"endpoint": "services", "id_field": "id", "name_field": "name"},
    "consumer": {
        "endpoint": "consumers",
        "id_field": "username",
        "name_field": "username",
    },
    "consumer_group": {
        "endpoint": "consumer_groups",
        "id_field": "id",
        "name_field": "name",
    },
    "plugin_config": {
        "endpoint": "plugin_configs",
        "id_field": "id",
        "name_field": "name",
    },
    "global_rule": {"endpoint": "global_rules", "id_field": "id", "name_field": "name"},
    "ssl": {"endpoint": "ssls", "id_field": "id", "name_field": "sni"},
}


def _format_json_value(value, indent_level=2):
    """Format a value as multi-line JSON for jsonencode()"""
    indent = "    " * indent_level
    next_indent = "    " * (indent_level + 1)

    if isinstance(value, bool):
        return str(value).lower()
    elif isinstance(value, (int, float)):
        return str(value)
    elif isinstance(value, str):
        escaped = value.replace("\\", "\\\\").replace('"', '\\"').replace("\n", "\\n")
        return f'"{escaped}"'
    elif isinstance(value, list):
        if not value:
            return "[]"
        lines = ["["]
        for item in value:
            lines.append(f"{next_indent}{_format_json_value(item, indent_level + 1)},")
        lines.append(f"{indent}]")
        return "\n".join(lines)
    elif isinstance(value, dict):
        if not value:
            return "{}"
        lines = ["{"]
        for k, v in value.items():
            lines.append(
                f"{next_indent}{k} = {_format_json_value(v, indent_level + 1)},"
            )
        lines.append(f"{indent}}}")
        return "\n".join(lines)
    else:
        return str(value)


def get_latest_provider_version(provider_source):
    """
    Get latest provider version from OpenTofu Registry API.
    """
    try:
        url = f"https://registry.opentofu.org/v1/providers/{provider_source}/versions"
        print(f"📦 Fetching latest version from OpenTofu Registry: {url}")

        req = Request(url)
        req.add_header("Accept", "application/json")

        with urlopen(req, timeout=10) as response:
            data = json.loads(response.read().decode())
            versions = data.get("versions", [])
            if versions:
                # Versions are returned in order, first is latest
                version = versions[0]["version"]
                print(f"✓ Found latest version: {version}")
                return version
            else:
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

    def list_resources(self, resource_type):
        config = RESOURCE_TYPES[resource_type]
        response = self._request(config["endpoint"])
        items = response.get("list", [])
        return [item.get("value", item) for item in items]


class ResourceGenerator:
    def __init__(self, client, provider_source, provider_version):
        self.client = client
        self.provider_source = provider_source
        self.provider_version = provider_version
        self.resources = {}

    def discover_all(self):
        print("🔍 Discovering APISIX resources...")
        for resource_type in RESOURCE_TYPES.keys():
            print(f"  Scanning {resource_type}...")
            resources = self.client.list_resources(resource_type)
            self.resources[resource_type] = resources
            print(f"    Found {len(resources)} {resource_type}(s)")
        return self.resources

    def _get_resource_id(self, resource_type, item):
        config = RESOURCE_TYPES.get(resource_type, {})
        id_field = config.get("id_field", "id")
        resource_id = item.get(id_field) or item.get("id") or "unknown"
        return str(resource_id)

    def _generate_resource_name(self, resource_id, resource_type):
        name = re.sub(r"[^a-zA-Z0-9_]", "_", resource_id)
        if name[0].isdigit():
            name = "r_" + name
        return f"{name}_{resource_type}"

    def _write_resource_attributes(self, f, resource_type, item):
        """Write ONLY configurable attributes (exclude read-only fields)"""

        if resource_type == "upstream":
            if "name" in item:
                f.write(f'  name = "{item["name"]}"\n')
            if "type" in item:
                f.write(f'  type = "{item["type"]}"\n')
            if "scheme" in item:
                f.write(f'  scheme = "{item["scheme"]}"\n')
            if "hash_on" in item:
                f.write(f'  hash_on = "{item["hash_on"]}"\n')
            if "retries" in item:
                f.write(f"  retries = {item['retries']}\n")
            if "retry_timeout" in item:
                f.write(f"  retry_timeout = {item['retry_timeout']}\n")
            if "pass_host" in item:
                f.write(f'  pass_host = "{item["pass_host"]}"\n')
            if "upstream_host" in item:
                f.write(f'  upstream_host = "{item["upstream_host"]}"\n')
            if "nodes" in item:
                f.write("  nodes {\n")
                for node in item["nodes"]:
                    f.write(f'    host     = "{node.get("host", "127.0.0.1")}"\n')
                    f.write(f"    port     = {node.get('port', 80)}\n")
                    f.write(f"    weight   = {node.get('weight', 100)}\n")
                    if "priority" in node:
                        f.write(f"    priority = {node['priority']}\n")
                f.write("  }\n")
            if "timeout" in item:
                f.write("  timeout {\n")
                if "connect" in item["timeout"]:
                    f.write(f"    connect = {item['timeout']['connect']}\n")
                if "send" in item["timeout"]:
                    f.write(f"    send = {item['timeout']['send']}\n")
                if "read" in item["timeout"]:
                    f.write(f"    read = {item['timeout']['read']}\n")
                f.write("  }\n")
            if "keepalive_pool" in item:
                f.write("  keepalive_pool {\n")
                if "idle_timeout" in item["keepalive_pool"]:
                    f.write(
                        f"    idle_timeout = {item['keepalive_pool']['idle_timeout']}\n"
                    )
                if "requests" in item["keepalive_pool"]:
                    f.write(f"    requests = {item['keepalive_pool']['requests']}\n")
                if "size" in item["keepalive_pool"]:
                    f.write(f"    size = {item['keepalive_pool']['size']}\n")
                f.write("  }\n")
            if "tls" in item:
                f.write("  tls {\n")
                if "verify" in item["tls"]:
                    f.write(f"    verify = {str(item['tls']['verify']).lower()}\n")
                if "version" in item["tls"]:
                    f.write(f'    version = "{item["tls"]["version"]}"\n')
                f.write("  }\n")
            if "labels" in item and item["labels"]:
                f.write("  labels = {\n")
                for key, value in item["labels"].items():
                    f.write(f'    {key} = "{value}"\n')
                f.write("  }\n")

        elif resource_type == "route":
            if "name" in item:
                f.write(f'  name = "{item["name"]}"\n')
            if "uri" in item:
                f.write(f'  uri = "{item["uri"]}"\n')
            if "uris" in item and isinstance(item["uris"], list):
                f.write("  uris = [\n")
                for uri in item["uris"]:
                    f.write(f'    "{uri}",\n')
                f.write("  ]\n")
            if "status" in item:
                f.write(f"  status = {item['status']}\n")
            if "priority" in item:
                f.write(f"  priority = {item['priority']}\n")
            if "upstream_id" in item:
                f.write(f'  upstream_id = "{item["upstream_id"]}"\n')
            if "service_id" in item:
                f.write(f'  service_id = "{item["service_id"]}"\n')
            if "methods" in item and isinstance(item["methods"], list):
                f.write("  methods = [\n")
                for method in item["methods"]:
                    f.write(f'    "{method}",\n')
                f.write("  ]\n")
            if "hosts" in item and isinstance(item["hosts"], list):
                f.write("  hosts = [\n")
                for host in item["hosts"]:
                    f.write(f'    "{host}",\n')
                f.write("  ]\n")
            if "remote_addrs" in item and isinstance(item["remote_addrs"], list):
                f.write("  remote_addrs = [\n")
                for addr in item["remote_addrs"]:
                    f.write(f'    "{addr}",\n')
                f.write("  ]\n")
            if "enable_websocket" in item:
                f.write(
                    f"  enable_websocket = {str(item['enable_websocket']).lower()}\n"
                )
            if "labels" in item and item["labels"]:
                f.write("  labels = {\n")
                for key, value in item["labels"].items():
                    f.write(f'    {key} = "{value}"\n')
                f.write("  }\n")

        elif resource_type == "service":
            if "name" in item:
                f.write(f'  name = "{item["name"]}"\n')
            if "upstream_id" in item:
                f.write(f'  upstream_id = "{item["upstream_id"]}"\n')
            if "hosts" in item and isinstance(item["hosts"], list):
                f.write("  hosts = [\n")
                for host in item["hosts"]:
                    f.write(f'    "{host}",\n')
                f.write("  ]\n")
            if "labels" in item and item["labels"]:
                f.write("  labels = {\n")
                for key, value in item["labels"].items():
                    f.write(f'    {key} = "{value}"\n')
                f.write("  }\n")

        elif resource_type == "consumer":
            if "username" in item:
                f.write(f'  username = "{item["username"]}"\n')
            if "desc" in item:
                f.write(f'  desc = "{item["desc"]}"\n')
            if "labels" in item and item["labels"]:
                f.write("  labels = {\n")
                for key, value in item["labels"].items():
                    f.write(f'    {key} = "{value}"\n')
                f.write("  }\n")

        elif resource_type == "consumer_group":
            if "name" in item:
                f.write(f'  name = "{item["name"]}"\n')
            if "labels" in item and item["labels"]:
                f.write("  labels = {\n")
                for key, value in item["labels"].items():
                    f.write(f'    {key} = "{value}"\n')
                f.write("  }\n")
            if "plugins" in item and item["plugins"]:
                f.write("  plugins = {\n")
                for pname, pconf in item["plugins"].items():
                    f.write(f'    "{pname}" = jsonencode({{\n')
                    for k, v in pconf.items():
                        f.write(f"      {k} = {_format_json_value(v, 2)}\n")
                    f.write("    })\n")
                f.write("  }\n")
            if "plugins" in item and item["plugins"]:
                f.write("  plugins = {\n")
                for pname, pconf in item["plugins"].items():
                    f.write(f'    "{pname}" = {{\n')
                    for k, v in pconf.items():
                        f.write(f"      {k} = {_format_value(v)}\n")
                    f.write("    }\n")
                f.write("  }\n")

        elif resource_type == "plugin_config":
            if "name" in item:
                f.write(f'  name = "{item["name"]}"\n')
            if "desc" in item:
                f.write(f'  desc = "{item["desc"]}"\n')
            if "plugins" in item and item["plugins"]:
                f.write("  plugins = {\n")
                for pname, pconf in item["plugins"].items():
                    f.write(f'    "{pname}" = jsonencode({{\n')
                    for k, v in pconf.items():
                        f.write(f"      {k} = {_format_json_value(v, 2)}\n")
                    f.write("    })\n")
                f.write("  }\n")
            if "labels" in item and item["labels"]:
                f.write("  labels = {\n")
                for key, value in item["labels"].items():
                    f.write(f'    {key} = "{value}"\n')
                f.write("  }\n")
            if "labels" in item and item["labels"]:
                f.write("  labels = {\n")
                for key, value in item["labels"].items():
                    f.write(f'    {key} = "{value}"\n')
                f.write("  }\n")

        elif resource_type == "global_rule":
            if "rule_id" in item:
                f.write(f'  rule_id = "{item["rule_id"]}"\n')
            if "plugins" in item and item["plugins"]:
                f.write("  plugins = {\n")
                for pname, pconf in item["plugins"].items():
                    f.write(f'    "{pname}" = jsonencode({{\n')
                    for k, v in pconf.items():
                        f.write(f"      {k} = {_format_json_value(v, 2)}\n")
                    f.write("    })\n")
                f.write("  }\n")
            if "labels" in item and item["labels"]:
                f.write("  labels = {\n")
                for key, value in item["labels"].items():
                    f.write(f'    {key} = "{value}"\n')
                f.write("  }\n")
            if "labels" in item and item["labels"]:
                f.write("  labels = {\n")
                for key, value in item["labels"].items():
                    f.write(f'    {key} = "{value}"\n')
                f.write("  }\n")

        elif resource_type == "ssl":
            if "sni" in item:
                f.write(f'  sni = "{item["sni"]}"\n')
            if "snis" in item and isinstance(item["snis"], list):
                f.write("  snis = [\n")
                for sni in item["snis"]:
                    f.write(f'    "{sni}",\n')
                f.write("  ]\n")
            if "cert" in item:
                f.write(f"  cert = <<CERT\n{item['cert']}\nCERT\n")
            if "key" in item:
                f.write(f"  key = <<KEY\n{item['key']}\nKEY\n")
            if "ssl_protocols" in item and isinstance(item["ssl_protocols"], list):
                f.write("  ssl_protocols = [\n")
                for proto in item["ssl_protocols"]:
                    f.write(f'    "{proto}",\n')
                f.write("  ]\n")
            if "labels" in item and item["labels"]:
                f.write("  labels = {\n")
                for key, value in item["labels"].items():
                    f.write(f'    {key} = "{value}"\n')
                f.write("  }\n")

    def generate_separate_files(self, output_dir):
        """Generate separate .tf files for each resource type"""
        print(f"\n📝 Generating separate HCL files in: {output_dir}")

        if self.provider_version:
            version_line = f'      version = "{self.provider_version}"\n'
            version_comment = f"# Using provider version: {self.provider_version}\n"
        else:
            version_line = ""
            version_comment = "# Using latest available provider version\n"

        provider_file = os.path.join(output_dir, "provider.tf")
        with open(provider_file, "w") as f:
            f.write(f"# APISIX Provider Configuration\n")
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
        print(f"  ✅ Generated provider.tf")

        for resource_type, items in self.resources.items():
            if not items:
                print(f"  ⊘ Skipping {resource_type} (no resources)")
                continue

            filename = f"{resource_type}s.tf"
            filepath = os.path.join(output_dir, filename)

            with open(filepath, "w") as f:
                f.write(
                    f"# APISIX {resource_type.replace('_', ' ').title()} Configuration\n"
                )
                f.write(f"# Generated: {datetime.now().isoformat()}\n")
                f.write(f"# Total resources: {len(items)}\n\n")

                for item in items:
                    resource_id = self._get_resource_id(resource_type, item)
                    resource_name = self._generate_resource_name(
                        resource_id, resource_type
                    )

                    f.write(f'resource "apisix_{resource_type}" "{resource_name}" {{\n')
                    self._write_resource_attributes(f, resource_type, item)
                    f.write("}\n\n")

            print(f"  ✅ Generated {filename} ({len(items)} resources)")

    def generate_import_script(self, output_file):
        print(f"\n📝 Generating import script: {output_file}")
        with open(output_file, "w") as f:
            f.write("#!/bin/bash\n")
            f.write(f"# APISIX Resource Import Script\n")
            f.write(f"# Generated: {datetime.now().isoformat()}\n\n")
            f.write("set -e\n\n")
            f.write('echo "🚀 Starting APISIX resource import..."\n')
            f.write('echo ""\n\n')

            for resource_type, items in self.resources.items():
                if not items:
                    continue
                f.write(
                    f'echo "=== Importing {resource_type.replace("_", " ").title()} ==="\n'
                )
                for item in items:
                    resource_id = self._get_resource_id(resource_type, item)
                    resource_name = self._generate_resource_name(
                        resource_id, resource_type
                    )
                    f.write(
                        f'tofu import apisix_{resource_type}.{resource_name} "{resource_id}"\n'
                    )
                f.write("\n")

            f.write('echo ""\n')
            f.write('echo "✅ Import complete!"\n')
            f.write('echo ""\n')
            f.write('echo "Next steps:"\n')
            f.write("1. Review generated .tf files and customize as needed\n")
            f.write("2. Run: tofu init\n")
            f.write("3. Run: tofu plan\n")
            f.write("4. Run: tofu apply\n")

        os.chmod(output_file, 0o755)
        print(f"  ✅ Generated import.sh")

    def generate_readme(self, output_file):
        print(f"\n📝 Generating README: {output_file}")
        with open(output_file, "w") as f:
            f.write("# APISIX Import Results\n\n")
            f.write(f"Generated: {datetime.now().isoformat()}\n\n")
            f.write(f"## Provider Information\n\n")
            f.write(f"- **Source**: `{self.provider_source}`\n")
            if self.provider_version:
                f.write(f"- **Version**: {self.provider_version}\n")
            else:
                f.write(f"- **Version**: Latest available from registry\n")
            f.write("\n")
            total = sum(len(items) for items in self.resources.values())
            f.write(f"## Summary\n\n")
            f.write(f"Total resources discovered: **{total}**\n\n")
            f.write("| Resource Type | Count | Generated File |\n")
            f.write("|--------------|-------|----------------|\n")
            for rtype, items in self.resources.items():
                filename = f"{rtype}s.tf" if items else "-"
                f.write(
                    f"| {rtype.replace('_', ' ').title()} | {len(items)} | {filename} |\n"
                )
            f.write("\n## Generated Files\n\n")
            f.write("- `provider.tf` - Provider configuration\n")
            for rtype, items in self.resources.items():
                if items:
                    f.write(f"- `{rtype}s.tf` - {len(items)} {rtype} resource(s)\n")
            f.write("- `import.sh` - Import script\n")
            f.write("- `README.md` - This file\n\n")
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
            f.write("## Important Notes\n\n")
            f.write(
                "- **Review all generated HCL**: Some values may need manual adjustment\n"
            )
            f.write("- **SSL certificates**: Exported if available from APISIX\n")
            f.write(
                "- **Sensitive data**: Review and secure any sensitive configuration\n"
            )
            if self.provider_version:
                f.write(f"- **Provider version**: Locked to {self.provider_version}\n")
            else:
                f.write(
                    "- **Provider version**: Will use latest available from registry\n"
                )
        print(f"  ✅ Generated README.md")


def main():
    parser = argparse.ArgumentParser(
        description="Import APISIX resources to Terraform/OpenTofu"
    )
    parser.add_argument("--base-url", default="http://localhost:9180/apisix/admin")
    parser.add_argument("--admin-key", default="test123456789")
    parser.add_argument("--output-dir", default="./import-output")
    parser.add_argument(
        "--provider-version",
        default=None,
        help="Provider version (default: fetch from GitHub Releases)",
    )
    parser.add_argument("--provider-source", default="scicore-unibas-ch/apisix")

    args = parser.parse_args()

    provider_version = args.provider_version
    if not provider_version:
        print("📦 Fetching latest provider version from OpenTofu Registry...")
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

    print(f"\n✅ Complete! Output: {os.path.abspath(args.output_dir)}")


if __name__ == "__main__":
    main()
