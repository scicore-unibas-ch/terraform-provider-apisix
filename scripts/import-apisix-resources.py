#!/usr/bin/env python3
"""
APISIX Resource Importer - Fixed v2
"""
# ... (same imports)

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
    
    def generate_hcl(self, output_file):
        print(f"\n📝 Generating HCL: {output_file}")
        with open(output_file, 'w') as f:
            f.write(f"# Auto-generated APISIX configuration\n")
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
            f.write('}\n\n')
            
            for resource_type, items in self.resources.items():
                if not items:
                    continue
                f.write(f"# {'='*60}\n# {resource_type.upper()}\n# {'='*60}\n\n")
                for item in items:
                    resource_id = self._get_resource_id(resource_type, item)
                    resource_name = self._generate_resource_name(resource_id, resource_type)
                    f.write(f'resource "apisix_{resource_type}" "{resource_name}" {{\n')
                    f.write(f'  id = "{resource_id}"\n')
                    name_field = RESOURCE_TYPES[resource_type].get('name_field', 'name')
                    if name_field in item:
                        f.write(f'  name = "{item[name_field]}"\n')
                    if resource_type == 'upstream' and 'type' in item:
                        f.write(f'  type = "{item["type"]}"\n')
                    if resource_type == 'route' and 'uri' in item:
                        f.write(f'  uri = "{item["uri"]}"\n')
                    if 'plugins' in item and item['plugins']:
                        f.write('  plugins = {\n')
                        for pname, pconf in item['plugins'].items():
                            f.write(f'    "{pname}" = jsonencode({json.dumps(pconf)})\n')
                        f.write('  }\n')
                    f.write('}\n\n')
        print(f"  ✅ Generated")
    
    def generate_import_script(self, output_file):
        print(f"\n📝 Generating import script: {output_file}")
        with open(output_file, 'w') as f:
            f.write('#!/bin/bash\n')
            f.write(f'# Generated: {datetime.now().isoformat()}\n\n')
            f.write('set -e\n')
            f.write('echo "🚀 Starting APISIX resource import..."\n\n')
            for resource_type, items in self.resources.items():
                if not items:
                    continue
                f.write(f'echo "Importing {resource_type}..."\n')
                for item in items:
                    resource_id = self._get_resource_id(resource_type, item)
                    resource_name = self._generate_resource_name(resource_id, resource_type)
                    f.write(f'tofu import apisix_{resource_type}.{resource_name} "{resource_id}"\n')
                f.write('\n')
            f.write('echo "✅ Import complete!"\n')
        os.chmod(output_file, 0o755)
        print(f"  ✅ Generated")
    
    def generate_readme(self, output_file):
        print(f"\n📝 Generating README: {output_file}")
        with open(output_file, 'w') as f:
            f.write('# APISIX Import Results\n\n')
            f.write(f'Generated: {datetime.now().isoformat()}\n\n')
            total = sum(len(items) for items in self.resources.values())
            f.write(f'Total resources: **{total}**\n\n')
            f.write('| Resource | Count |\n|----------|-------|\n')
            for rtype, items in self.resources.items():
                f.write(f'| {rtype} | {len(items)} |\n')
        print(f"  ✅ Generated")

def main():
    parser = argparse.ArgumentParser(description='Import APISIX resources to Terraform')
    parser.add_argument('--base-url', default='http://localhost:9180/apisix/admin')
    parser.add_argument('--admin-key', default='test123456789')
    parser.add_argument('--output-dir', default='./import-output')
    args = parser.parse_args()
    os.makedirs(args.output_dir, exist_ok=True)
    print(f"🔌 Connecting to APISIX: {args.base_url}")
    client = APISIXClient(args.base_url, args.admin_key)
    generator = ResourceGenerator(client)
    generator.discover_all()
    generator.generate_hcl(os.path.join(args.output_dir, 'main.tf'))
    generator.generate_import_script(os.path.join(args.output_dir, 'import.sh'))
    generator.generate_readme(os.path.join(args.output_dir, 'README.md'))
    print(f"\n✅ Complete! Output: {os.path.abspath(args.output_dir)}")

if __name__ == '__main__':
    main()
