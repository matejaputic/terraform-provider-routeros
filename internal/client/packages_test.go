// SPDX-License-Identifier: MPL-2.0
package client

import "testing"

func TestInstalledPackageCapabilities(t *testing.T) {
	base := map[string]any{"name": "routeros", "version": "7.24.5", "disabled": "false"}
	got, e := ValidatePackages("7.24.5 (stable)", []map[string]any{base, {"name": "container", "version": "7.24.5"}, {"name": "wireless", "version": "7.20", "disabled": true}})
	if e != nil || len(got) != 2 || got[0] != "container" || got[1] != "routeros" {
		t.Fatal("installed capabilities incorrectly inferred")
	}
	for _, rows := range [][]map[string]any{nil, {base, base}, {{"name": "routeros", "version": "7.24.4"}}, {{"name": "routeros", "version": "7.24.5", "disabled": "true"}}, {base, {"name": "extra", "version": "7.25beta5"}}, {base, {"name": "extra", "version": "7.24.5", "disabled": "unknown"}}} {
		if _, e := ValidatePackages("7.24.5", rows); e == nil {
			t.Fatal("unverified/mixed capabilities accepted")
		}
	}
}
