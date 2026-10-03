#!/usr/bin/env python3
"""Evidence-bound lifecycle triage and fail-closed public contract delta assessment."""
import argparse
from collections import Counter
import json
from pathlib import Path
import re
import sys

import extract

ROOT = extract.ROOT
POLICY = ROOT / 'schemas/maintenance-policy.json'


def load(path):
    def unique(pairs):
        result = {}
        for key, value in pairs:
            if key in result:
                raise ValueError('duplicate JSON key: ' + key)
            result[key] = value
        return result
    return json.loads(path.read_text(), object_pairs_hook=unique,
                      parse_constant=lambda value: (_ for _ in ()).throw(ValueError('nonfinite JSON')))


def index(items, key):
    result = {}
    if not isinstance(items, list):
        raise ValueError('expected contract list')
    for item in items:
        name = item[key]
        if not isinstance(name, str) or not name or name in result:
            raise ValueError('duplicate or invalid contract identity')
        result[name] = item
    return result


def validate_descriptors(value):
    if not {'format', 'policy_revision', 'resources'} <= set(value):
        raise ValueError('incomplete descriptor envelope')
    resources = index(value['resources'], 'terraform_type')
    for resource in resources.values():
        index(resource['fields'], 'name')
        # Novel keys, empty contracts and migration claims yield review findings,
        # never implicit approval. All resource/field keys are compared recursively.
    return resources


def normalized(value):
    result = json.loads(extract.encode(value))
    result['resources'].sort(key=lambda item: item['terraform_type'])
    for resource in result['resources']:
        resource['fields'].sort(key=lambda item: item['name'])
        for field in resource['fields']:
            field.pop('provenance', None)  # descriptive evidence only, never runtime policy
    return result


def differences(before, after, path=''):
    if type(before) is not type(after):
        return [{'path': path, 'kind': 'type-or-value-change'}]
    if isinstance(before, dict):
        findings = []
        for key in sorted(set(before) | set(after)):
            pointer = path + '/' + key.replace('~', '~0').replace('/', '~1')
            if key not in before:
                findings.append({'path': pointer, 'kind': 'addition'})
            elif key not in after:
                findings.append({'path': pointer, 'kind': 'removal'})
            else:
                findings.extend(differences(before[key], after[key], pointer))
        return findings
    if isinstance(before, list):
        # Identity-keyed lists produce actionable names, not shifting numeric indexes.
        key = 'terraform_type' if path.endswith('/resources') else 'name' if path.endswith('/fields') else None
        if key:
            return differences(index(before, key), index(after, key), path)
        return [] if before == after else [{'path': path, 'kind': 'value-change'}]
    return [] if before == after else [{'path': path, 'kind': 'value-change'}]


def assess(before, after, policy):
    if extract.encode(policy) != extract.encode(load(POLICY)):
        raise ValueError('unreviewed maintenance policy')
    validate_descriptors(before)
    validate_descriptors(after)
    if extract.digest(extract.encode(before)) != policy['baseline_descriptor_sha256']:
        raise ValueError('baseline descriptor does not match reviewed policy')
    changes = differences(normalized(before), normalized(after))
    return {'format': 'routeros-contract-delta@1', 'policy_revision': policy['revision'],
            'policy_sha256': extract.digest(extract.encode(policy)),
            'baseline_sha256': extract.digest(extract.encode(before)),
            'candidate_sha256': extract.digest(extract.encode(after)),
            'decision': 'review-required' if changes else 'offline-compatible',
            'findings': changes, 'new_exposure_authorized': False,
            'live_tested': False, 'publication_authorized': False,
            'scope': 'adapted public contract only; no upstream semantic or live-lane certification'}


def lifecycle(contract):
    props = contract['resource_properties']
    methods = {name: props.get(name + 'Context', {}).get('call', {}).get('callee')
               for name in ('Create', 'Read', 'Update', 'Delete')}
    if methods == {name: 'Default' + name for name in methods}:
        return 'ordinary-collection-candidate', methods
    if methods == {name: 'DefaultSystem' + name for name in methods}:
        return 'singleton-candidate', methods
    if any(value and value.startswith('updateOnlyDevice') for value in methods.values()):
        return 'hardware-update-only', methods
    return 'custom-or-unresolved', methods


def matrix(bundle, descriptors):
    manifest = extract.verify_bundle(bundle)
    inventory = load(bundle / 'reference-contracts.json')
    reconciliation = load(bundle / 'reconciliation.json')
    validate_descriptors(descriptors)
    if (inventory['source']['revision'] != extract.PIN
            or reconciliation['descriptor_sha256'] != extract.digest(extract.encode(descriptors))):
        raise ValueError('stale reference/overlay reconciliation')
    if set(reconciliation['policy_inputs_sha256']) != {'schemas/ip-address-policy.json', 'internal/catalog/collections.json'}:
        raise ValueError('unexpected runtime policy bindings')
    for path, expected in reconciliation['policy_inputs_sha256'].items():
        if extract.digest((ROOT / path).read_bytes()) != expected:
            raise ValueError('stale reviewed runtime policy')
    current = index(descriptors['resources'], 'terraform_type')
    catalog = {r['resource_name']: r for r in load(ROOT / 'internal/catalog/collections.json')}
    helper_sources = ['internal/provider/collection_resource.go', 'internal/provider/ip_address_resource.go',
                      'internal/provider/firewall_order.go', 'internal/provider/firewall_actions.go',
                      'internal/provider/firewall_config_validation.go', 'internal/provider/firewall_address.go',
                      'internal/provider/recovery.go', 'internal/provider/validators.go',
                      'internal/provider/dhcp_relay.go', 'internal/provider/dns_record.go',
                      'internal/catalog/collections.go', 'internal/catalog/ip_address.go']
    codec_helpers = {'wire-string': 'payload/decodeField (or maintained IP address lifecycle)',
                     'yes/no': 'strictWireBool/decodeField', 'decimal': 'payload/decodeField',
                     'csv': 'validateCSV/payload/decodeField',
                     'placement': 'filterOrder/positionFilter (not ordinary payload)'}
    rows = []
    groups = {}
    for name, constructor in inventory['registrations']['resources'].items():
        groups.setdefault(constructor, []).append(name)
    for constructor, names in sorted(groups.items()):
        contract = inventory['constructors'][constructor]
        family, methods = lifecycle(contract)
        risks = []
        if family != 'ordinary-collection-candidate':
            risks.append(family)
        if contract['status'] != 'static-declarations-only':
            risks.append('unresolved-schema-construction')
        if 'StateUpgraders' in contract['resource_properties']:
            risks.append('sdk-state-upgrades-not-ported')
        sensitive = sorted(name for name, field in contract['fields'].items()
                           if field.get('properties', {}).get('Sensitive', {}).get('value') is True)
        nested = sorted(name for name, field in contract['fields'].items()
                        if 'Elem' in field.get('properties', {}))
        if sensitive:
            risks.append('sensitive-readback-and-state')
        if nested:
            risks.append('nested-or-set-semantics')
        path_expression = contract['metadata'].get('___path___', {}).get('expression', '')
        path_match = re.fullmatch(r'PropResourcePath\("(/[a-zA-Z0-9_/-]+)"\)', path_expression)
        path = path_match.group(1) if path_match else None
        if path is None:
            risks.append('unresolved-wire-path')
        if path and path.startswith('/ip/firewall/'):
            risks.append('ordered-firewall-and-action-applicability')
        implemented = []
        for name in sorted(set(names) & set(current)):
            resource = current[name]
            if path != resource['path']:
                raise ValueError('reviewed runtime path differs from reference declaration: ' + name)
            runtime_fields = {f['name']: f for f in catalog.get(resource['name'], {}).get('attributes', [])}
            mapped_fields = []
            for field in resource['fields']:
                if field['codec'] not in codec_helpers:
                    raise ValueError('unmapped maintained codec: ' + field['codec'])
                mapped = {key: field.get(key) for key in
                          ('name', 'wire', 'type', 'mode', 'codec', 'force_new', 'sensitive', 'default', 'validators')}
                mapped['maintained_codec_helper'] = codec_helpers[field['codec']]
                if resource['name'] == 'ip_address':
                    mapped['maintained_codec_helper'] = ('wireBool/payload/refresh' if field['codec'] == 'yes/no'
                                                       else 'wireString/payload/refresh')
                mapped['read_default'] = runtime_fields.get(field['name'], {}).get('read_default')
                mapped['conditional_read'] = runtime_fields.get(field['name'], {}).get('conditional_read', False)
                mapped_fields.append(mapped)
            implemented.append({'resource': name, 'scope': 'reviewed subset, not reference parity',
                                'lifecycle': resource['lifecycle'], 'schema_version': resource['schema_version'],
                                'migration_compatibility': resource['migration_compatibility'],
                                'field_capabilities': mapped_fields})
        rows.append({'constructor': constructor, 'resource_names': sorted(names), 'source': contract['source'],
                     'family': family, 'callbacks': methods, 'wire_path_candidate': path,
                     'id_expression': contract['metadata'].get('___id___', {}).get('expression'),
                     'risks': sorted(risks), 'sensitive_fields': sensitive, 'nested_fields': nested,
                     'related_test_sources': contract['related_test_sources'], 'implemented': implemented,
                     'semantic_review_required': True, 'automatic_exposure_authorized': False})
    return {'format': 'routeros-capability-matrix@1', 'reference_sha': extract.PIN,
            'bundle_files_sha256': manifest['files_sha256'],
            'descriptor_sha256': extract.digest(extract.encode(descriptors)),
            'classifier_sha256': extract.digest(Path(__file__).read_bytes()),
            'maintained_helper_sources_sha256': {p: extract.digest((ROOT / p).read_bytes()) for p in helper_sources},
            'reviewed_policy_inputs_sha256': reconciliation['policy_inputs_sha256'],
            'counts': {'resource_names': sum(len(row['resource_names']) for row in rows),
                       'resource_constructors': len(rows), 'implemented_subsets': sum(len(r['implemented']) for r in rows),
                       'families': dict(sorted(Counter(r['family'] for r in rows).items()))},
            'scope': 'static triage and maintained helper mapping; not executed SDK behavior or exposure approval',
            'constructors': rows}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest='command', required=True)
    triage = sub.add_parser('matrix')
    triage.add_argument('--bundle', type=Path, default=ROOT / 'schemas/reference-contracts')
    triage.add_argument('--output', type=Path, required=True)
    delta = sub.add_parser('delta')
    delta.add_argument('--baseline', type=Path, required=True)
    delta.add_argument('--candidate', type=Path, required=True)
    delta.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    try:
        if args.command == 'matrix':
            report = matrix(args.bundle, load(ROOT / 'schemas/wire-descriptors.json'))
        else:
            report = assess(load(args.baseline), load(args.candidate), load(POLICY))
        args.output.parent.mkdir(parents=True, exist_ok=True)
        extract.os.replace(_temporary(args.output, extract.encode(report)), args.output)
        return 2 if report.get('decision') == 'review-required' else 0
    except (ValueError, KeyError, TypeError, OSError) as error:
        print('Capability assessment failed: ' + str(error), file=sys.stderr)
        return 1


def _temporary(path, raw):
    with extract.tempfile.NamedTemporaryFile(dir=path.parent, delete=False) as stream:
        stream.write(raw)
        stream.flush()
        extract.os.fsync(stream.fileno())
        return stream.name


if __name__ == '__main__':
    sys.exit(main())
