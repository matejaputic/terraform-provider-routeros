// SPDX-License-Identifier: MPL-2.0
package catalog

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

func TestGeneratedWireDescriptorMatchesMaintainedRuntime(t *testing.T) {
	b, err := os.ReadFile("../../schemas/wire-descriptors.json")
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Resources []struct {
			Name    string `json:"name"`
			Path    string `json:"path"`
			ID      string `json:"id_wire_key"`
			Version int    `json:"schema_version"`
			Fields  []struct {
				Name string `json:"name"`
				Wire string `json:"wire"`
			} `json:"fields"`
			RuntimeRead struct {
				Method string `json:"method"`
				Path   string `json:"path"`
				Filter string `json:"id_filter"`
			} `json:"runtime_read"`
		} `json:"resources"`
	}
	if err := json.Unmarshal(b, &manifest); err != nil {
		t.Fatal(err)
	}
	if len(manifest.Resources) != 1+len(Resources()) {
		t.Fatal("unexpected resource set")
	}
	r := manifest.Resources[0]
	if r.Name != "ip_address" || r.Path != IPAddressPath || r.ID != IDKey || r.Version != 0 {
		t.Fatal("descriptor/lifecycle identity changed")
	}
	if r.RuntimeRead.Method != "GET" || r.RuntimeRead.Path != IPAddressPath || r.RuntimeRead.Filter != IDKey {
		t.Fatal("filtered collection read contract changed")
	}
	wires := map[string]string{}
	for _, f := range r.Fields {
		if _, ok := wires[f.Name]; ok {
			t.Fatal("duplicate field")
		}
		wires[f.Name] = f.Wire
	}
	if !reflect.DeepEqual(wires, IPAddressWireNames) {
		t.Fatalf("wire mapping mismatch: %v", wires)
	}
}
