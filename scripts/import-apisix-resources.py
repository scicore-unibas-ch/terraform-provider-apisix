#!/usr/bin/env python3
"""
APISIX Resource Importer

Scrapes APISIX Admin API to discover existing resources and generates:
1. Separate HCL files per resource type (upstreams.tf, routes.tf, etc.)
2. Import script (import.sh)
3. README with summary

Usage:
    python scripts/import-apisix-resources.py [OPTIONS]
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
    'upstream': {'endpoint': 'upstreams', 'id_field': 'id', 'name_field': 'name'},
    'route': {'endpoint': 'routes', 'id_field': 'id', 'name_field': 'name'},
    'service': {'endpoint': 'services', 'id_field': 'id', 'name_field': 'name'},
    'consumer': {'endpoint': 'consumers', 'id_field': 'username', 'name_field': 'username'},
    'consumer_group': {'endpoint': 'consumer_groups', 'id_field': 'id', 'name_field': 'name'},
    'plugin_config': {'endpoint': 'plugin_configs', 'id_field': 'id', 'name_field': 'name'},
    'global_rule': {'endpoint': 'global_rules', 'id_field': 'id', 'name_field': 'name'},
    'ssl': {'endpoint': 'ssls', 'id_field': 'id', 'name_field': 'sni'},
}

class APISIXClient:
    def __init__(self, base_url, admin_key):
        self.base_url = base_url.rstrip('/')
        self.admin_key = admin_key
    
    def _request(self, endpoint):
        url = f"{self.base_url}/{endpoint}"
        req = Request(url)
        req.add_header('X-API-KEY', self.admin_key)
        try:
            with urlopen(req) as response:
                return json.loads(response.read().decode())
        except HTTPError as e:
            if e.code == 404:
                return {'total': 0, 'list': []}
            raise
        except URLError as e:
            print(f"Error connecting to APISIX: {e}")
            sys.exit(1)
    
    def list_resources(self, resource_type):
        config = RESOURCE_TYPES[resource_type]
        response = self._request(config['endpoint'])
        items = response.get('list', [])
        # Extract the 'value' from each item (APISIX wraps resources in 'value')
        return [item.get('value', item) for item in items]

class ResourceGenerator:
    def __init__(self, client):
        self.client = client
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
        id_field = config.get('id_field', 'id')
        resource_id = item.get(id_field) or item.get('id') or 'unknown'
        return str(resource_id)
    
    def _generate_resource_name(self, resource_id, resource_type):
        name = re.sub(r'[^a-zA-Z0-9_]', '_', resource_id)
        if name[0].isdigit():
            name = 'r_' + name
        return f"{name}_{resource_type}"
    
    def _write_resource_attributes(self, f, resource_type, item):
        """Write resource-specific attributes"""
        # Write ID
        resource_id = self._get_resource_id(resource_type, item)
        f.write(f'  id = "{resource_id}"\n')
        
        # Write name if available
        name_field = RESOURCE_TYPES[resource_type].get('name_field', 'name')
        if name_field in item:
            f.write(f'  name = "{item[name_field]}"\n')
        
        # Write description if available
        if 'desc' in item:
            f.write(f'  desc = "{item["desc"]}"\n')
        
        # Upstream-specific fields
        if resource_type == 'upstream':
            if 'type' in item:
                f.write(f'  type = "{item["type"]}"\n')
            if 'scheme' in item:
                f.write(f'  scheme = "{item["scheme"]}"\n')
            if 'hash_on' in item:
                f.write(f'  hash_on = "{item["hash_on"]}"\n')
            if 'key' in item:
                f.write(f'  key = "{item["key"]}"\n')
            if 'retries' in item:
                f.write(f'  retries = {item["retries"]}\n')
            if 'retry_timeout' in item:
                f.write(f'  retry_timeout = {item["retry_timeout"]}\n')
            if 'pass_host' in item:
                f.write(f'  pass_host = "{item["pass_host"]}"\n')
            if 'upstream_host' in item:
                f.write(f'  upstream_host = "{item["upstream_host"]}"\n')
            if 'nodes' in item:
                f.write('  nodes {\n')
                for node in item['nodes']:
                    f.write(f'    host     = "{node.get("host", "127.0.0.1")}"\n')
                    f.write(f'    port     = {node.get("port", 80)}\n')
                    f.write(f'    weight   = {node.get("weight", 100)}\n')
                    if 'priority' in node:
                        f.write(f'    priority = {node["priority"]}\n')
                    if 'metadata' in node:
                        f.write(f'    metadata = {json.dumps(node["metadata"])}\n')
                f.write('  }\n')
            if 'timeout' in item:
                f.write('  timeout {\n')
                if 'connect' in item['timeout']:
                    f.write(f'    connect = {item["timeout"]["connect"]}\n')
                if 'send' in item['timeout']:
                    f.write(f'    send = {item["timeout"]["send"]}\n')
                if 'read' in item['timeout']:
                    f.write(f'    read = {item["timeout"]["read"]}\n')
                f.write('  }\n')
            if 'keepalive_pool' in item:
                f.write('  keepalive_pool {\n')
                if 'idle_timeout' in item['keepalive_pool']:
                    f.write(f'    idle_timeout = {item["keepalive_pool"]["idle_timeout"]}\n')
                if 'requests' in item['keepalive_pool']:
                    f.write(f'    requests = {item["keepalive_pool"]["requests"]}\n')
                if 'size' in item['keepalive_pool']:
                    f.write(f'    size = {item["keepalive_pool"]["size"]}\n')
                f.write('  }\n')
            if 'tls' in item:
                f.write('  tls {\n')
                if 'verify' in item['tls']:
                    f.write(f'    verify = {str(item["tls"]["verify"]).lower()}\n')
                if 'version' in item['tls']:
                    f.write(f'    version = "{item["tls"]["version"]}"\n')
                f.write('  }\n')
            if 'labels' in item and item['labels']:
                f.write('  labels = {\n')
                for key, value in item['labels'].items():
                    f.write(f'    {key} = "{value}"\n')
                f.write('  }\n')
        
        # Route-specific fields
        if resource_type == 'route':
            if 'uri' in item:
                f.write(f'  uri = "{item["uri"]}"\n')
            if 'uris' in item and isinstance(item['uris'], list):
                f.write('  uris = [\n')
                for uri in item['uris']:
                    f.write(f'    "{uri}",\n')
                f.write('  ]\n')
            if 'status' in item:
                f.write(f'  status = {item["status"]}\n')
            if 'priority' in item:
                f.write(f'  priority = {item["priority"]}\n')
            if 'upstream_id' in item:
                f.write(f'  upstream_id = "{item["upstream_id"]}"\n')
            if 'service_id' in item:
                f.write(f'  service_id = "{item["service_id"]}"\n')
            if 'methods' in item and isinstance(item['methods'], list):
                f.write('  methods = [\n')
                for method in item['methods']:
                    f.write(f'    "{method}",\n')
                f.write('  ]\n')
            if 'hosts' in item and isinstance(item['hosts'], list):
                f.write('  hosts = [\n')
                for host in item['hosts']:
                    f.write(f'    "{host}",\n')
                f.write('  ]\n')
            if 'remote_addrs' in item and isinstance(item['remote_addrs'], list):
                f.write('  remote_addrs = [\n')
                for addr in item['remote_addrs']:
                    f.write(f'    "{addr}",\n')
                f.write('  ]\n')
            if 'enable_websocket' in item:
                f.write(f'  enable_websocket = {str(item["enable_websocket"]).lower()}\n')
            if 'labels' in item and item['labels']:
                f.write('  labels = {\n')
                for key, value in item['labels'].items():
                    f.write(f'    {key} = "{value}"\n')
                f.write('  }\n')
        
        # Service-specific fields
        if resource_type == 'service':
            if 'upstream_id' in item:
                f.write(f'  upstream_id = "{item["upstream_id"]}"\n')
            if 'hosts' in item and isinstance(item['hosts'], list):
                f.write('  hosts = [\n')
                for host in item['hosts']:
                    f.write(f'    "{host}",\n')
                f.write('  ]\n')
            if 'labels' in item and item['labels']:
                f.write('  labels = {\n')
                for key, value in item['labels'].items():
                    f.write(f'    {key} = "{value}"\n')
                f.write('  }\n')
        
        # Consumer-specific fields
        if resource_type == 'consumer':
            if 'username' in item:
                f.write(f'  username = "{item["username"]}"\n')
            if 'desc' in item:
                f.write(f'  desc = "{item["desc"]}"\n')
            if 'labels' in item and item['labels']:
                f.write('  labels = {\n')
                for key, value in item['labels'].items():
                    f.write(f'    {key} = "{value}"\n')
                f.write('  }\n')
        
        # Consumer Group-specific fields
        if resource_type == 'consumer_group':
            if 'id' in item:
                f.write(f'  group_id = "{item["id"]}"\n')
            if 'labels' in item and item['labels']:
                f.write('  labels = {\n')
                for key, value in item['labels'].items():
                    f.write(f'    {key} = "{value}"\n')
                f.write('  }\n')
            if 'plugins' in item and item['plugins']:
                f.write('  plugins = {\n')
                for pname, pconf in item['plugins'].items():
                    f.write(f'    "{pname}" = jsonencode({json.dumps(pconf)})\n')
                f.write('  }\n')
        
        # Plugin Config-specific fields
        if resource_type == 'plugin_config':
            if 'desc' in item:
                f.write(f'  desc = "{item["desc"]}"\n')
            if 'plugins' in item and item['plugins']:
                f.write('  plugins = {\n')
                for pname, pconf in item['plugins'].items():
                    f.write(f'    "{pname}" = jsonencode({json.dumps(pconf)})\n')
                f.write('  }\n')
            if 'labels' in item and item['labels']:
                f.write('  labels = {\n')
                for key, value in item['labels'].items():
                    f.write(f'    {key} = "{value}"\n')
                f.write('  }\n')
        
        # Global Rule-specific fields
        if resource_type == 'global_rule':
            if 'rule_id' in item:
                f.write(f'  rule_id = "{item["rule_id"]}"\n')
            if 'plugins' in item and item['plugins']:
                f.write('  plugins = {\n')
                for pname, pconf in item['plugins'].items():
                    f.write(f'    "{pname}" = jsonencode({json.dumps(pconf)})\n')
                f.write('  }\n')
            if 'labels' in item and item['labels']:
                f.write('  labels = {\n')
                for key, value in item['labels'].items():
                    f.write(f'    {key} = "{value}"\n')
                f.write('  }\n')
        
        # SSL-specific fields
        if resource_type == 'ssl':
            if 'sni' in item:
                f.write(f'  sni = "{item["sni"]}"\n')
            if 'snis' in item and isinstance(item['snis'], list):
                f.write('  snis = [\n')
                for sni in item['snis']:
                    f.write(f'    "{sni}",\n')
                f.write('  ]\n')
            if 'cert' in item:
                f.write(f'  cert = <<CERT\n{item["cert"]}\nCERT\n')
            if 'key' in item:
                f.write(f'  key = <<KEY\n{item["key"]}\nKEY\n')
            if 'ssl_protocols' in item and isinstance(item['ssl_protocols'], list):
                f.write('  ssl_protocols = [\n')
                for proto in item['ssl_protocols']:
                    f.write(f'    "{proto}",\n')
                f.write('  ]\n')
            if 'labels' in item and item['labels']:
                f.write('  labels = {\n')
                for key, value in item['labels'].items():
                    f.write(f'    {key} = "{value}"\n')
                f.write('  }\n')
    
    def generate_separate_files(self, output_dir):
        """Generate separate .tf files for each resource type"""
        print(f"\n📝 Generating separate HCL files in: {output_dir}")
        
        # Write provider.tf
        provider_file = os.path.join(output_dir, 'provider.tf')
        with open(provider_file, 'w') as f:
            f.write(f"# APISIX Provider Configuration\n")
            f.write(f"# Generated: {datetime.now().isoformat()}\n\n")
            f.write('terraform {\n')
            f.write('  required_providers {\n')
            f.write('    apisix = {\n')
            f.write('      source  = "scicore-unibas-ch/apisix"\n')
            f.write('      version = "0.1.0"\n')
            f.write('    }\n')
            f.write('  }\n')
            f.write('}\n\n')
            f.write('provider "apisix" {\n')
            f.write('  base_url  = "http://localhost:9180/apisix/admin"\n')
            f.write('  admin_key = "test123456789"\n')
            f.write('}\n')
        print(f"  ✅ Generated provider.tf")
        
        # Write one file per resource type
        for resource_type, items in self.resources.items():
            if not items:
                print(f"  ⊘ Skipping {resource_type} (no resources)")
                continue
            
            filename = f"{resource_type}s.tf"
            filepath = os.path.join(output_dir, filename)
            
            with open(filepath, 'w') as f:
                f.write(f"# APISIX {resource_type.replace('_', ' ').title()} Configuration\n")
                f.write(f"# Generated: {datetime.now().isoformat()}\n")
                f.write(f"# Total resources: {len(items)}\n\n")
                
                for item in items:
                    resource_id = self._get_resource_id(resource_type, item)
                    resource_name = self._generate_resource_name(resource_id, resource_type)
                    
                    f.write(f'resource "apisix_{resource_type}" "{resource_name}" {{\n')
                    self._write_resource_attributes(f, resource_type, item)
                    f.write('}\n\n')
            
            print(f"  ✅ Generated {filename} ({len(items)} resources)")
    
    def generate_import_script(self, output_file):
        print(f"\n📝 Generating import script: {output_file}")
        with open(output_file, 'w') as f:
            f.write('#!/bin/bash\n')
            f.write(f'# APISIX Resource Import Script\n')
            f.write(f'# Generated: {datetime.now().isoformat()}\n\n')
            f.write('set -e\n\n')
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
                f.write('\n')
            
            f.write('echo ""\n')
            f.write('echo "✅ Import complete!"\n')
            f.write('echo ""\n')
            f.write('echo "Next steps:"\n')
            f.write('1. Review generated .tf files and customize as needed\n')
            f.write('2. Run: tofu plan\n')
            f.write('3. Run: tofu apply\n')
        
        os.chmod(output_file, 0o755)
        print(f"  ✅ Generated import.sh")
    
    def generate_readme(self, output_file):
        print(f"\n📝 Generating README: {output_file}")
        with open(output_file, 'w') as f:
            f.write('# APISIX Import Results\n\n')
            f.write(f'Generated: {datetime.now().isoformat()}\n\n')
            total = sum(len(items) for items in self.resources.values())
            f.write(f'## Summary\n\n')
            f.write(f'Total resources discovered: **{total}**\n\n')
            f.write('| Resource Type | Count | Generated File |\n')
            f.write('|--------------|-------|----------------|\n')
            for rtype, items in self.resources.items():
                filename = f"{rtype}s.tf" if items else '-'
                f.write(f'| {rtype.replace("_", " ").title()} | {len(items)} | {filename} |\n')
            f.write('\n## Generated Files\n\n')
            f.write('- `provider.tf` - Provider configuration\n')
            for rtype, items in self.resources.items():
                if items:
                    f.write(f'- `{rtype}s.tf` - {len(items)} {rtype} resource(s)\n')
            f.write('- `import.sh` - Import script\n')
            f.write('- `README.md` - This file\n\n')
            f.write('## Usage\n\n')
            f.write('```bash\n')
            f.write('# 1. Review and customize generated files\n')
            f.write('vim *.tf\n\n')
            f.write('# 2. Run the import script\n')
            f.write('./import.sh\n\n')
            f.write('# 3. Verify the import\n')
            f.write('tofu plan\n\n')
            f.write('# 4. Apply (if needed)\n')
            f.write('tofu apply\n')
            f.write('```\n\n')
            f.write('## Important Notes\n\n')
            f.write('- **Review all generated HCL**: Some values may need manual adjustment\n')
            f.write('- **SSL certificates**: Exported if available from APISIX\n')
            f.write('- **Sensitive data**: Review and secure any sensitive configuration\n')
            f.write('- **Provider version**: Update the provider version in provider.tf as needed\n')
        print(f"  ✅ Generated README.md")

def main():
    parser = argparse.ArgumentParser(description='Import APISIX resources to Terraform')
    parser.add_argument('--base-url', default='http://localhost:9180/apisix/admin',
                       help='APISIX Admin API URL')
    parser.add_argument('--admin-key', default='test123456789',
                       help='Admin API key')
    parser.add_argument('--output-dir', default='./import-output',
                       help='Output directory for generated files')
    args = parser.parse_args()
    
    os.makedirs(args.output_dir, exist_ok=True)
    print(f"🔌 Connecting to APISIX: {args.base_url}")
    
    client = APISIXClient(args.base_url, args.admin_key)
    generator = ResourceGenerator(client)
    generator.discover_all()
    generator.generate_separate_files(args.output_dir)
    generator.generate_import_script(os.path.join(args.output_dir, 'import.sh'))
    generator.generate_readme(os.path.join(args.output_dir, 'README.md'))
    
    print(f"\n✅ Complete! Output: {os.path.abspath(args.output_dir)}")
    print(f"\n📁 Generated files:")
    print(f"   - provider.tf (provider configuration)")
    for rtype, items in generator.resources.items():
        if items:
            print(f"   - {rtype}s.tf ({len(items)} resources)")
    print(f"   - import.sh (import script)")
    print(f"   - README.md (documentation)")

if __name__ == '__main__':
    main()
