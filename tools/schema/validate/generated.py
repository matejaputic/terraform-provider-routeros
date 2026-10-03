#!/usr/bin/env python3
# SPDX-License-Identifier: MPL-2.0
"""Require the exact reviewed official Framework output set before promotion."""
from pathlib import Path
import sys

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / 'normalize'))
from normalize import APPROVED_PATHS


def verify(directory):
    expected = {name + '_resource_gen.go' for name in APPROVED_PATHS}
    actual = {path.relative_to(directory).as_posix() for path in directory.rglob('*') if path.is_file()}
    if actual != expected:
        raise ValueError('generated file set differs from reviewed catalog: missing=' +
                         repr(sorted(expected - actual)) + ' unexpected=' + repr(sorted(actual - expected)))


if __name__ == '__main__':
    try:
        verify(Path(sys.argv[1]))
    except (IndexError, OSError, ValueError) as error:
        print('generated-output-failure: ' + str(error), file=sys.stderr)
        sys.exit(1)
