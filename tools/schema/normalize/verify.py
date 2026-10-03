#!/usr/bin/env python3
# SPDX-License-Identifier: MPL-2.0
"""Verify an adaptation bundle's final marker before official generation."""
import argparse
from pathlib import Path
import sys

import normalize as n

def verify(directory):
    report = n.discover.parse((directory / 'adaptation-report.json').read_bytes())
    n.check(report.get('format') == n.FORMAT and (report.get('supported_resources') == ['ip_address'] or isinstance(report.get('supported_resources'), list) and len(report['supported_resources']) == len(n.APPROVED_PATHS) and set(report['supported_resources']) == set(n.APPROVED_PATHS)) and report.get('data_sources') == [], 'unexpected adaptation identity/resource set')
    for name, key in [('upstream-input.json', 'input_sha256'), ('ip-address.openapi.json', 'normalized_sha256'),
                      ('generator-config.yml', 'config_sha256'), ('wire-descriptors.json', 'wire_descriptor_sha256')]:
        n.check(n.sha((directory / name).read_bytes()) == report.get(key), 'partial/tampered adaptation artifact: ' + name)
    if 'companion_input_sha256' in report:
        n.check(n.sha((directory / 'upstream-extra-input.json').read_bytes()) == report['companion_input_sha256'], 'partial/tampered extra publication')
        extra = n.SchemaSnapshot((directory / 'upstream-extra-input.json').read_bytes())
        n.check(extra.version == report['version'], 'companion version mismatch')
    spec = n.discover.parse((directory / 'ip-address.openapi.json').read_bytes())
    n.check(spec['info']['version'] == report['version'], 'adaptation version mismatch')
    return report

def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--output', type=Path, default=n.ROOT / 'schemas')
    args = parser.parse_args()
    try:
        verify(args.output)
    except (n.AdaptationError, n.discover.DiscoveryError, OSError, KeyError, TypeError) as e:
        print('adaptation-bundle-failure: ' + str(e), file=sys.stderr)
        return 1
    return 0

if __name__ == '__main__': sys.exit(main())
