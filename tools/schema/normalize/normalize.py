#!/usr/bin/env python3
# SPDX-License-Identifier: MPL-2.0
"""Consumer-owned OpenAPI adapter. Inventory is not resource authorization."""
import argparse
import copy
import hashlib
import json
from pathlib import Path
import re
import sys

ROOT = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(ROOT / 'tools/schema/discover'))
import discover

CONTROLS = {'.query', '.proplist', 'numbers', 'number', 'copy-from', 'as-string', 'as-string-value', 'value-name'}
HARDWARE = {'/interface/ethernet', '/interface/wireless', '/interface/wifi', '/interface/lte'}
METHODS = {'get', 'put', 'post', 'patch', 'delete', 'head', 'options', 'trace'}
FORMAT = 'routeros-adaptation@1'
APPROVED_PATHS = {'ip_address': '/ip/address', 'interface_bridge': '/interface/bridge', 'interface_bridge_port': '/interface/bridge/port', 'interface_vlan': '/interface/vlan', 'interface_list': '/interface/list', 'interface_list_member': '/interface/list/member', 'ip_pool': '/ip/pool', 'ip_dhcp_server_network': '/ip/dhcp-server/network', 'ip_dhcp_server': '/ip/dhcp-server', 'ip_dhcp_server_lease': '/ip/dhcp-server/lease', 'ip_route': '/ip/route', 'ip_firewall_addr_list': '/ip/firewall/address-list', 'ip_firewall_filter': '/ip/firewall/filter', 'ip_firewall_nat': '/ip/firewall/nat', 'ip_firewall_mangle': '/ip/firewall/mangle', 'ip_firewall_raw': '/ip/firewall/raw', 'ip_dhcp_client_option': '/ip/dhcp-client/option', 'ip_dhcp_server_option': '/ip/dhcp-server/option', 'ip_dhcp_relay': '/ip/dhcp-relay', 'ip_dns_record': '/ip/dns/static'}
ORDERED_RESOURCES = {'ip_firewall_filter', 'ip_firewall_nat', 'ip_firewall_mangle', 'ip_firewall_raw'}

class AdaptationError(Exception):
    pass

def check(ok, message):
    if not ok:
        raise AdaptationError(message)

def sha(data):
    return hashlib.sha256(data).hexdigest()

def snake(name):
    check(isinstance(name, str) and re.fullmatch(r'[A-Za-z0-9_.-]+', name), 'unsafe property name')
    if name == '.id':
        return 'id'
    result = re.sub(r'[^A-Za-z0-9]+', '_', name).strip('_').lower()
    check(re.fullmatch(r'[a-z][a-z0-9_]*', result), 'invalid Terraform identifier')
    return result

def symbols(names):
    tf, go = {}, {}
    for wire in sorted(names):
        name = snake(wire)
        symbol = ''.join(part.capitalize() for part in name.split('_'))
        check(name not in tf, 'Terraform name collision: ' + name)
        check(symbol not in go, 'Go symbol collision: ' + symbol)
        tf[name], go[symbol] = wire, wire
    return tf

class Resolver:
    def __init__(self, spec):
        self.spec = spec
        self.removed = set()

    def pointer(self, ref):
        check(isinstance(ref, str) and ref.startswith('#/'), 'external refs are forbidden')
        node = self.spec
        for part in ref[2:].split('/'):
            part = part.replace('~1', '/').replace('~0', '~')
            check(isinstance(node, dict) and part in node, 'unresolved internal ref: ' + ref)
            node = node[part]
        check(isinstance(node, dict), 'ref target must be an object')
        return node

    def merge(self, left, right):
        result = copy.deepcopy(left)
        for key, value in right.items():
            if key == 'properties':
                target = result.setdefault(key, {})
                check(isinstance(value, dict) and isinstance(target, dict), 'invalid object properties')
                for name, definition in value.items():
                    check(name not in target or target[name] == definition, 'incompatible allOf property: ' + name)
                    target[name] = definition
            elif key == 'required':
                check(isinstance(value, list) and all(isinstance(v, str) for v in value), 'invalid required set')
                result[key] = sorted(set(result.get(key, [])) | set(value))
            else:
                check(key not in result or result[key] == value, 'incompatible composition keyword: ' + key)
                result[key] = copy.deepcopy(value)
        return result

    def resolve(self, node, stack=(), depth=0):
        check(depth <= 64 and isinstance(node, dict), 'invalid/deep schema object')
        if '$ref' in node:
            ref = node['$ref']
            if ref == '#/components/schemas/QueryOptions':
                helper = self.wrapper(self.pointer(ref))
                check(helper.get('type', 'object') == 'object' and isinstance(helper.get('properties'), dict), 'invalid shared QueryOptions object')
                self.removed.add('shared QueryOptions')
                return {'type': 'object', 'properties': {}}
            check(isinstance(ref, str) and ref not in stack, 'recursive ref is unsupported')
            target = self.resolve(self.pointer(ref), stack + (ref,), depth + 1)
            siblings = {k: v for k, v in node.items() if k != '$ref'}
            return self.merge(target, self.resolve(siblings, stack, depth + 1))
        check(not any(key in node for key in ('oneOf', 'anyOf', 'not')), 'unsupported schema union/composition')
        check('type' not in node or node['type'] in ('object', 'array', 'string', 'boolean', 'integer', 'number'), 'unsupported/invalid schema type')
        check('enum' not in node or isinstance(node['enum'], list) and node['enum'], 'invalid enum')
        out = {}
        for key, value in node.items():
            if key == 'allOf':
                continue
            if key == 'properties':
                check(isinstance(value, dict), 'invalid properties')
                out[key] = {}
                for name, definition in sorted(value.items()):
                    if name in CONTROLS:
                        self.removed.add(name)
                    else:
                        out[key][name] = self.resolve(definition, stack, depth + 1)
            elif key == 'items' or key == 'additionalProperties' and isinstance(value, dict):
                out[key] = self.resolve(value, stack, depth + 1)
            elif key == 'required':
                check(isinstance(value, list) and all(isinstance(v, str) for v in value), 'invalid required')
                out[key] = sorted(v for v in value if v not in CONTROLS)
            else:
                out[key] = copy.deepcopy(value)
        if 'allOf' in node:
            members = node['allOf']
            check(isinstance(members, list) and members, 'invalid allOf')
            for member in members:
                flattened = self.resolve(member, stack, depth + 1)
                check(flattened.get('type', 'object') == 'object' and not any(k in flattened for k in ('items', 'enum')), 'only object allOf is supported')
                out = self.merge(out, flattened)
        if 'properties' in out:
            check(out.get('type', 'object') == 'object', 'properties on non-object')
            out.setdefault('type', 'object')
            check(set(out.get('required', [])) <= set(out['properties']), 'required property is absent')
        return out

    def wrapper(self, node):
        """Resolve request/response object refs without treating them as schemas."""
        seen = set()
        check(isinstance(node, dict), 'invalid OpenAPI wrapper object')
        while '$ref' in node:
            ref = node['$ref']
            check(isinstance(ref, str) and ref not in seen and len(seen) < 64, 'recursive/deep wrapper ref')
            seen.add(ref)
            node = self.merge(self.pointer(ref), {k: v for k, v in node.items() if k != '$ref'})
        return node

    def body(self, operation):
        check(isinstance(operation, dict), 'invalid operation')
        body = self.wrapper(operation.get('requestBody'))
        try:
            schema = body['content']['application/json']['schema']
        except (TypeError, KeyError):
            raise AdaptationError('CRUD body requires application/json schema') from None
        resolved = self.resolve(schema)
        check(resolved.get('type') == 'object' and isinstance(resolved.get('properties'), dict), 'CRUD body is not an object')
        return resolved

    def response(self, operation):
        response = self.wrapper(operation.get('responses', {}).get('200'))
        try:
            schema = response['content']['application/json']['schema']
        except (TypeError, KeyError):
            raise AdaptationError('read requires JSON response schema (empty objects may be curated)') from None
        return self.resolve(schema)

def inventory(spec):
    result = []
    identifiers, go_symbols = set(), set()
    for path, operations in sorted(spec['paths'].items()):
        if path.startswith('x-'):
            continue
        check(isinstance(path, str) and path.startswith('/') and isinstance(operations, dict), 'invalid path inventory')
        methods = sorted(METHODS & operations.keys())
        if 'post' in methods:
            result.append({'path': path, 'classification': 'action-excluded', 'methods': ['post']})
        if '{' in path:
            continue
        item = spec['paths'].get(path + '/{id}', {})
        complete = 'put' in methods and isinstance(item, dict) and {'get', 'patch', 'delete'} <= item.keys()
        if complete:
            identifier = snake(path.strip('/').replace('/', '_'))
            symbol = ''.join(p.capitalize() for p in identifier.split('_'))
            check(identifier not in identifiers and symbol not in go_symbols, 'resource identifier collision: ' + identifier)
            identifiers.add(identifier)
            go_symbols.add(symbol)
            classification = 'collection-candidate-unreviewed'
        elif {'get', 'patch'} <= operations.keys():
            classification = 'singleton-or-existing-object-deferred'
        else:
            classification = 'incomplete-or-read-only-deferred'
        if path in HARDWARE:
            classification = 'existing-hardware-deferred'
        entry = {'path': path, 'classification': classification, 'methods': methods, 'structural_crud': complete}
        if complete:
            # Unsupported unapproved endpoints are inventory findings, not public
            # schemas. Approved endpoints are independently re-resolved strictly.
            try:
                resolver = Resolver(spec)
                create = resolver.body(operations['put'])['properties']
                update = resolver.body(item['patch'])['properties']
                symbols(set(create) | set(update))
                entry['create_only'] = sorted(set(create) - set(update))
                entry['update_only'] = sorted(set(update) - set(create))
                entry['wire_fields'] = sorted(set(create) | set(update))
                entry['enum_fields_pending_classification'] = sorted(n for n, s in create.items() if 'enum' in s)
                entry['adaptation_status'] = 'observed-not-authorized'
            except AdaptationError as e:
                entry['adaptation_status'] = 'unsupported-needs-review'
                entry['reason'] = str(e)
        result.append(entry)
    return result

class SchemaSnapshot:
    """Invocation-local immutable-input reuse; never an approval cache.

    Policies, fields and lifecycle observations are checked for every resource.
    Resolver only reads this parsed input. Inventory is observed once, then copied
    before any resource-specific classification so reports cannot taint siblings.
    """
    def __init__(self, raw):
        self.input_sha256 = sha(raw)
        self.spec = discover.parse(raw)
        check(isinstance(self.spec, dict) and isinstance(self.spec.get('info'), dict), 'input must have an OpenAPI info object')
        self.version = discover.version(self.spec.get('info', {}).get('version'))
        discover.validate_spec(self.spec, self.version, 'adapter input')
        self._inventory = None

    def observed_inventory(self):
        if self._inventory is None:
            self._inventory = inventory(self.spec)
        return self._inventory


def normalize(raw, policy, *, metadata=None, origin=None, snapshot=None, include_inventory=True):
    snapshot = snapshot or SchemaSnapshot(raw)
    check(snapshot.input_sha256 == sha(raw), 'cached schema input hash mismatch')
    spec, version = snapshot.spec, snapshot.version
    check(isinstance(policy, dict), 'policy must be an object')
    check(policy.get('schema_version') == 0 and policy.get('migration_compatibility') == 'not claimed', 'unreviewed schema-version/migration policy')
    name = policy.get('resource_name')
    check(name in APPROVED_PATHS and policy.get('wire_path') == APPROVED_PATHS[name], 'unsupported lifecycle registration')
    resource_name = name
    fields = policy.get('attributes')
    check(isinstance(fields, list) and fields, 'missing curated attributes')
    field_map = {}
    for field in fields:
        check(isinstance(field, dict), 'invalid field policy')
        wire, name = field.get('wire'), field.get('name')
        check(name == snake(wire) and name not in field_map, 'invalid/duplicate catalog alias')
        check(field.get('type') in ('string', 'boolean', 'integer', 'array') and field.get('mode') in ('required', 'computed_optional', 'computed'), 'unimplemented field type/mode')
        check(field.get('sensitive') is False and (field.get('force_new') is False or resource_name != 'ip_address' and field.get('force_new') is True and (name == 'name' or resource_name == 'ip_dns_record' and name == 'type') and field.get('type') == 'string'), 'sensitivity/replacement policy requires a maintained implementation')
        check(resource_name != 'ip_address' or field.get('type') in ('string', 'boolean'), 'unimplemented IP address field type')
        check(field.get('codec') != 'placement' or resource_name in ORDERED_RESOURCES and name == 'place_before' and field['type'] == 'string' and field['mode'] == 'computed_optional', 'unimplemented ordered lifecycle')
        check(resource_name not in ORDERED_RESOURCES or name != 'place_before' or field.get('codec') == 'placement', 'missing explicit ordering codec')
        check(field.get('type') != 'array' or field.get('codec') == 'csv' and (resource_name, name) in (('ip_pool', 'ranges'), ('ip_dhcp_server_network', 'dns_server')), 'unimplemented array codec')
        if 'read_default' in field:
            expected_default = 'true' if (resource_name, name) == ('ip_dhcp_server', 'conflict_detection') else 'false' if (resource_name, name) in (('ip_dhcp_server_network', 'dns_none'), ('ip_dhcp_server', 'add_arp'), ('ip_dhcp_server_lease', 'block_access'), ('ip_dhcp_server_option', 'force'), ('ip_dns_record', 'match_subdomain'), ('ip_route', 'dynamic'), ('ip_route', 'active')) else 'none' if resource_name == 'ip_pool' and name == 'next_pool' else '' if (resource_name in ORDERED_RESOURCES and name in ('src_address_list', 'dst_address_list', 'to_addresses', 'new_connection_mark', 'new_packet_mark') or name in ('comment', 'include', 'exclude') and resource_name != 'ip_address' or resource_name == 'ip_dhcp_server_network' and name in ('dns_server', 'domain')) else None
            check(field['read_default'] == expected_default and expected_default is not None and field['type'] in ('string', 'array', 'boolean') and (field['mode'] == 'computed_optional' or resource_name == 'ip_route' and name in ('dynamic', 'active') and field['mode'] == 'computed'), 'unreviewed omission default')
        check('conditional_read' not in field or resource_name == 'interface_bridge' and name == 'pvid' and field['conditional_read'] == 'vlan_filtering', 'unimplemented conditional read')
        check(field.get('enum_kind') in ('none', 'device-instance-removed'), 'static enum requires reviewed runtime validator')
        field_map[name] = field
    check(field_map.get('id', {}).get('mode') == 'computed' and field_map['id']['wire'] == '.id', 'exactly one computed ID required')
    symbols(f['wire'] for f in fields)
    for mode, key in [('required', 'required'), ('computed_optional', 'optional_computed'), ('computed', 'computed')]:
        check(set(policy.get(key, [])) == {f['name'] for f in fields if f['mode'] == mode}, 'conflicting mode policy: ' + key)
    check(set(policy.get('booleans', [])) == {f['name'] for f in fields if f['type'] == 'boolean'}, 'conflicting boolean policy')
    path = policy['wire_path']
    paths = spec['paths']
    check(path in paths and path + '/{id}' in paths and 'put' in paths[path] and {'get', 'patch', 'delete'} <= paths[path + '/{id}'].keys(), 'approved resource lost full CRUD lifecycle')
    resolver = Resolver(spec)
    create = resolver.body(paths[path]['put'])
    update = resolver.body(paths[path + '/{id}']['patch'])
    read = resolver.response(paths[path + '/{id}']['get'])
    check(read.get('type', 'object') == 'object', 'read must describe an object')
    cp, up = create['properties'], update['properties']
    rp = read.get('properties', {})
    check(isinstance(rp, dict), 'invalid read properties')
    observed = set(cp) | set(up) | set(rp)
    # .id and an actual id together must not silently collapse into one attribute.
    symbols(observed | {f['wire'] for f in fields})
    create_only = sorted(set(cp) - set(up))
    for field in fields:
        for properties in (cp, up, rp):
            observed_schema = properties.get(field['wire'], {})
            allowed_types = (None, 'string', field['type'])
            if field['type'] == 'array' and observed_schema.get('type') == 'array':
                check(observed_schema.get('items', {}).get('type') == 'string', 'unreviewed list element type')
            check(observed_schema.get('type') in allowed_types, 'reviewed field type drift requires policy review: ' + field['wire'])
            if field['type'] == 'boolean' and observed_schema.get('type') == 'string' and 'enum' in observed_schema:
                check(all(v in ('yes', 'no', 'true', 'false') for v in observed_schema['enum']), 'unreviewed boolean wire domain: ' + field['wire'])
        if field['mode'] != 'computed':
            check(field['wire'] not in create_only or field['force_new'] or resource_name in ORDERED_RESOURCES and field['name'] == 'place_before' and field['codec'] == 'placement', 'create-only approved field requires replacement policy: ' + field['wire'])
    meta_hashes, suggestions = {}, set()
    for label, data in sorted((metadata or {}).items()):
        obj = discover.parse(data)
        check(isinstance(obj, dict), 'inspect must be an object')
        if '_meta' in obj:
            check(isinstance(obj['_meta'], dict) and obj['_meta'].get('version') == version, 'inspect version mismatch')
        meta_hashes[label] = sha(data)
        # Inspect command trees/completions are observations, not type/default truth.
        def visit(node, depth=0):
            check(depth <= 64, 'inspect metadata too deep')
            if isinstance(node, dict):
                for key, value in node.items():
                    if key in ('properties', 'completions') and isinstance(value, dict):
                        suggestions.update(k for k in value if isinstance(k, str) and re.fullmatch(r'[A-Za-z][A-Za-z0-9_-]*', k))
                    visit(value, depth + 1)
            elif isinstance(node, list):
                for value in node: visit(value, depth + 1)
        visit(obj)
    bodies, response_props = {}, {}
    for name, field in sorted(field_map.items()):
        definition = {'type': field['type']}
        if field['type'] == 'array': definition['items'] = {'type': 'string'}
        response_props[name] = definition
        if field['mode'] != 'computed':
            bodies[name] = definition
    body = {'type': 'object', 'required': sorted(f['name'] for f in fields if f['mode'] == 'required'), 'properties': bodies}
    response = {'description': 'Curated IP address state', 'content': {'application/json': {'schema': {'type': 'object', 'properties': response_props}}}}
    def write():
        return {'requestBody': {'required': True, 'content': {'application/json': {'schema': body}}}, 'responses': {'200': response}}
    normalized = {'openapi': '3.0.3', 'info': {'title': 'RouterOS curated IP address slice', 'version': version},
                  'paths': {path: {'put': write()}, path + '/{id}': {'get': {'responses': {'200': response}}, 'patch': write(), 'delete': {'responses': {'204': {'description': 'Deleted'}}}}}}
    descriptor = {'format': 'routeros-wire-descriptors@1', 'policy_revision': policy['revision'], 'resources': [{
        'name': resource_name, 'terraform_type': 'routeros_' + resource_name, 'schema_version': 0, 'lifecycle': 'reviewed-collection',
        'path': path, 'id_wire_key': '.id', 'schema_read': {'method': 'GET', 'path': path + '/{id}', 'omit_path_parameter_from_schema': True},
        'runtime_read': {'method': 'GET', 'path': path, 'id_filter': '.id'},
        'operations': {'create': {'method': 'PUT', 'path': path}, 'update': {'method': 'PATCH', 'path': path + '/{id}'}, 'delete': {'method': 'DELETE', 'path': path + '/{id}'}},
        'fields': sorted(fields, key=lambda f: f['name']), 'migration_compatibility': 'not claimed'}]}
    observed_catalog = snapshot.observed_inventory()
    catalog = copy.deepcopy(observed_catalog) if include_inventory else []
    for entry in catalog:
        if entry['path'] == path and entry['classification'] == 'collection-candidate-unreviewed':
            entry['classification'] = 'approved-maintained-resource'
    pins = discover.parse((ROOT / 'schemas/provenance.json').read_bytes())
    producer = {k: pins[k] for k in ('normalizer_revision', 'runtime_revision', 'openapi_generator', 'framework_generator', 'provider_code_spec_version', 'codegen_spec_dependencies')}
    producer['adapter_source_sha256'] = sha(Path(__file__).read_bytes())
    report = {'format': FORMAT, 'producer': producer, 'input_sha256': sha(raw), 'version': version, 'origin': origin,
              'policy_revision': policy['revision'], 'policy_sha256': sha(discover.encode(policy)),
              'config_sha256': None, 'wire_descriptor_sha256': sha(discover.encode(descriptor)),
              'metadata_sha256': meta_hashes, 'normalized_sha256': sha(discover.encode(normalized)),
              'supported_resources': [resource_name], 'data_sources': [], 'catalog': catalog,
              'coverage': {'structural_crud_candidates': sum(e.get('structural_crud', False) for e in observed_catalog),
                           'approved_maintained_resources': 1, 'automatically_exposed_new_resources': 0},
              'adaptations': {'removed_controls': sorted(resolver.removed), 'synthetic_id': True,
                  'removed_schema_path_parameter': 'id', 'repaired_response_from_curated_policy': True,
                  'instance_enums_removed': sorted(f['wire'] for f in fields if f['enum_kind'] == 'device-instance-removed'),
                  'typed_codecs': {f['wire']: 'yes/no boolean' for f in fields if f['type'] == 'boolean'},
                  'create_only_unexposed': create_only,
                  'retained_curated_fields_absent_from_observation': sorted(f['wire'] for f in fields if f['wire'] not in observed and f['wire'] != '.id'),
                  'unreviewed_wire_string_candidates': sorted((observed | suggestions) - {f['wire'] for f in fields} - CONTROLS)},
              'schema_version': 0, 'state_upgrade': 'not claimed', 'acceptance': 'no additional version/flavor acceptance implied'}
    config = 'provider:\n  name: routeros\nresources:\n  ' + resource_name + ':\n'
    for action, method, route in [('create', 'PUT', path), ('read', 'GET', path + '/{id}'), ('update', 'PATCH', path + '/{id}'), ('delete', 'DELETE', path + '/{id}')]:
        config += f'    {action}:\n      path: {route}\n      method: {method}\n'
    report['config_sha256'] = sha(config.encode())
    return normalized, config.encode(), descriptor, report

def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--input', type=Path, help='immutable OpenAPI input; defaults to the pinned fixture for the selected catalog')
    parser.add_argument('--policy', type=Path, default=ROOT / 'schemas/ip-address-policy.json')
    parser.add_argument('--output', type=Path, default=ROOT / 'schemas')
    parser.add_argument('--inspect', action='append', type=Path, default=[])
    parser.add_argument('--collections', action='store_true', help='include the explicitly reviewed networking resource catalog')
    parser.add_argument('--manifest', type=Path, help='discovery manifest: verifies input, metadata and processing fingerprints')
    parser.add_argument('--key', help='selected discovery candidate key; required with --manifest')
    args = parser.parse_args()
    if args.input is None:
        args.input = ROOT / 'tools/schema/normalize/fixtures' / ('collections.upstream.json' if args.collections else 'ip-address.upstream.json')
    try:
        check(args.input.stat().st_size <= discover.LIMIT, 'input exceeds size limit')
        raw = args.input.read_bytes()
        snapshot = SchemaSnapshot(raw)
        check(not args.manifest or not args.inspect, 'manifest metadata cannot be overridden')
        check(len({p.name for p in args.inspect}) == len(args.inspect), 'duplicate inspect filenames')
        metadata = {}
        for p in args.inspect:
            check(p.stat().st_size <= discover.LIMIT, 'inspect exceeds size limit')
            metadata[p.name] = p.read_bytes()
        origin = {'kind': 'explicit-input', 'input_sha256': sha(raw)}
        if args.manifest:
            manifest = discover.parse(args.manifest.read_bytes())
            check(isinstance(manifest, dict), 'manifest must be an object')
            check(isinstance(manifest.get('candidates'), list) and all(isinstance(c, dict) for c in manifest['candidates']), 'invalid candidate list')
            check(manifest.get('format') == discover.FORMAT and manifest.get('repository') == discover.REPOSITORY, 'invalid discovery manifest')
            check(isinstance(args.key, str), '--key required for discovery manifest')
            matches = [c for c in manifest.get('candidates', []) if c.get('key') == args.key]
            check(len(matches) == 1, 'candidate key missing/duplicated')
            candidate = matches[0]
            check(candidate['repository'] == discover.REPOSITORY and candidate['architecture'] == 'x86' and candidate['flavor'] in ('base', 'extra'), 'unsupported candidate repository/architecture/flavor')
            check(candidate['channel'] == discover.channel(candidate['version']), 'candidate channel mismatch')
            expected_path = 'docs/' + ('nightly' if candidate['channel'] == 'nightly' else candidate['version'])
            expected_path += ('/extra' if candidate['flavor'] == 'extra' else '') + '/openapi.json'
            check(candidate['schema_path'] == expected_path, 'candidate path/lane mismatch')
            identity_keys = ('repository', 'channel', 'version', 'flavor', 'architecture', 'schema_path', 'schema_sha256', 'semantic_metadata', 'inputs')
            check(sha(discover.encode({k: candidate[k] for k in identity_keys})) == args.key, 'candidate identity/key mismatch')
            check(candidate['inputs'] == discover.fingerprints(ROOT), 'processing inputs changed: rediscover candidate')
            check(candidate['schema_sha256'] == sha(raw), 'input does not match discovered schema hash')
            discover.valid_path(candidate['schema_path'])
            check(discover.SHA.fullmatch(manifest['upstream_sha']) and candidate['upstream_sha'] == manifest['upstream_sha'], 'upstream SHA mismatch')
            check(snapshot.version == candidate['version'], 'candidate version mismatch')
            # Re-read only saved immutable snapshot metadata, with hash checks.
            metadata = {}
            for path, hashes in candidate['metadata'].items():
                discover.valid_path(path)
                artifact = args.manifest.parent / candidate['upstream_sha'] / path
                check(artifact.stat().st_size <= discover.LIMIT, 'snapshot metadata exceeds size limit')
                data = artifact.read_bytes()
                check(sha(data) == hashes['sha256'] and sha(discover.encode(discover.semantic(discover.parse(data)))) == candidate['semantic_metadata'][path], 'snapshot metadata hash mismatch')
                if path.endswith('nightly.json'):
                    continue  # Discovery already checked all phase provenance; hash remains bound to key.
                metadata[path] = data
            origin = {k: candidate[k] for k in ('repository', 'upstream_sha', 'channel', 'version', 'flavor', 'architecture', 'schema_path', 'key')}
        elif args.input.resolve() == (ROOT / 'tools/schema/normalize/fixtures/ip-address.upstream.json').resolve():
            origin = discover.parse((args.input.parent / 'provenance.json').read_bytes())
            check(origin['fixture_sha256'] == sha(raw), 'baseline fixture hash mismatch')
        if args.input.resolve() == (ROOT / 'tools/schema/normalize/fixtures/collections.upstream.json').resolve():
            origin = discover.parse((args.input.parent / 'collections.provenance.json').read_bytes())
            check(origin['fixture_sha256'] == sha(raw), 'collection fixture hash mismatch')
        policy = discover.parse(args.policy.read_bytes())
        check(not args.manifest or sha(discover.encode(policy)) == sha(discover.encode(discover.parse((ROOT / 'schemas/ip-address-policy.json').read_bytes()))), 'manifest policy override forbidden')
        normalized, config, descriptor, report = normalize(raw, policy, metadata=metadata, origin=origin, snapshot=snapshot)
        if args.collections:
            policies = discover.parse((ROOT / 'internal/catalog/collections.json').read_bytes())
            check({p['resource_name'] for p in policies} == set(APPROVED_PATHS) - {'ip_address'} and len(policies) == len(APPROVED_PATHS) - 1, 'incomplete/duplicate reviewed networking catalog')
            reports = {}
            for collection_policy in policies:
                extra, extra_config, extra_descriptor, extra_report = normalize(raw, collection_policy, metadata=metadata, origin=origin, snapshot=snapshot, include_inventory=False)
                check(not (set(normalized['paths']) & set(extra['paths'])), 'duplicate catalog paths')
                normalized['paths'].update(extra['paths'])
                config += extra_config.split(b'resources:\n', 1)[1]
                descriptor['resources'].extend(extra_descriptor['resources'])
                reports[collection_policy['resource_name']] = extra_report['adaptations']
            report['collection_adaptations'] = reports
            report['collection_policy_sha256'] = sha(discover.encode(policies))
            report['supported_resources'] = ['ip_address'] + [p['resource_name'] for p in policies]
            report['coverage']['approved_maintained_resources'] = len(APPROVED_PATHS)
            for entry in report['catalog']:
                if entry['path'] in APPROVED_PATHS.values(): entry['classification'] = 'approved-maintained-resource'
            report['normalized_sha256'] = sha(discover.encode(normalized))
            report['config_sha256'] = sha(config)
            report['wire_descriptor_sha256'] = sha(discover.encode(descriptor))
        # No output/checkpoint changes until every input has passed adaptation.
        outputs = {'upstream-input.json': raw, 'ip-address.openapi.json': discover.encode(normalized),
                   'generator-config.yml': config, 'wire-descriptors.json': discover.encode(descriptor),
                   'adaptation-report.json': discover.encode(report)}
        with discover.locked(args.output / 'adaptation'):
            for name in sorted(set(outputs) - {'adaptation-report.json'}):
                discover.atomic(args.output / name, outputs[name])
            # The report is the final bundle marker; hashes bind every artifact.
            discover.atomic(args.output / 'adaptation-report.json', outputs['adaptation-report.json'])
        print(json.dumps({'resources': report['supported_resources'], 'input_sha256': sha(raw), 'catalog_entries': len(report['catalog'])}, sort_keys=True))
    except (AdaptationError, discover.DiscoveryError, OSError, KeyError, TypeError, ValueError, RecursionError) as e:
        print('adaptation-failure: ' + str(e), file=sys.stderr)
        return 1
    return 0

if __name__ == '__main__':
    sys.exit(main())
