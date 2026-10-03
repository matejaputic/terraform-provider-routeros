// SPDX-License-Identifier: MPL-2.0
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/matejaputic/terraform-provider-routeros/internal/catalog"
	"io"
	"os"
)

// Reject duplicate properties independently of the CLI's permissive warnings.
func uniqueJSON(d *json.Decoder) {
	token, err := d.Token()
	if err != nil {
		panic(err)
	}
	if delim, ok := token.(json.Delim); ok {
		seen := map[string]bool{}
		for d.More() {
			if delim == '{' {
				key, err := d.Token()
				if err != nil {
					panic(err)
				}
				name, ok := key.(string)
				if !ok || seen[name] {
					panic("duplicate/invalid JSON property")
				}
				seen[name] = true
			}
			uniqueJSON(d)
		}
		if _, err := d.Token(); err != nil {
			panic(err)
		}
	}
}

func main() {
	path := "schemas/provider-code-spec.json"
	if len(os.Args) > 2 {
		panic("usage: validate [provider-code-spec.json]")
	}
	if len(os.Args) == 2 {
		path = os.Args[1]
	}
	b, e := os.ReadFile(path)
	if e != nil {
		panic(e)
	}
	decoder := json.NewDecoder(bytes.NewReader(b))
	uniqueJSON(decoder)
	if _, err := decoder.Token(); err != io.EOF {
		panic("trailing JSON")
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(b, &envelope); err != nil {
		panic(err)
	}
	for key := range envelope {
		if key != "provider" && key != "resources" && key != "data_sources" && key != "version" {
			panic("unsupported specification section: " + key)
		}
	}
	var provider map[string]json.RawMessage
	if err := json.Unmarshal(envelope["provider"], &provider); err != nil || len(provider) != 1 {
		panic("unexpected generated provider configuration")
	}
	var s struct {
		Version  string `json:"version"`
		Provider struct {
			Name string `json:"name"`
		} `json:"provider"`
		Resources []struct {
			Name   string `json:"name"`
			Schema struct {
				Attributes []map[string]json.RawMessage `json:"attributes"`
			} `json:"schema"`
		} `json:"resources"`
		DataSources []any `json:"data_sources"`
	}
	if e = json.Unmarshal(b, &s); e != nil {
		panic(e)
	}
	if s.Version != "0.1" || s.Provider.Name != "routeros" || len(s.Resources) != 1+len(catalog.Collections()) || len(s.DataSources) != 0 {
		panic("unexpected specification identity or resource set")
	}
	contracts := map[string]map[string]string{}
	for _, collection := range catalog.Collections() {
		fields := map[string]string{}
		for _, field := range collection.Fields {
			kind := map[string]string{"string": "string", "boolean": "bool", "integer": "int64", "array": "list"}[field.Type]
			if kind == "" {
				panic("unsupported catalog type")
			}
			fields[field.Name] = kind + "/" + field.Mode
		}
		contracts[collection.Name] = fields
	}
	contracts["ip_address"] = map[string]string{"address": "string/required", "interface": "string/required", "comment": "string/computed_optional", "network": "string/computed_optional", "vrf": "string/computed", "disabled": "bool/computed_optional", "id": "string/computed", "actual_interface": "string/computed", "dynamic": "bool/computed", "invalid": "bool/computed", "slave": "bool/computed"}
	for _, res := range s.Resources {
		expected, ok := contracts[res.Name]
		if !ok {
			panic("unexpected/duplicate resource: " + res.Name)
		}
		delete(contracts, res.Name)
		for _, a := range res.Schema.Attributes {
			var name string
			if e = json.Unmarshal(a["name"], &name); e != nil {
				panic(e)
			}
			want, ok := expected[name]
			if !ok {
				panic("unexpected or duplicate attribute: " + name)
			}
			if len(a) != 2 {
				panic("unsupported attribute union or metadata: " + name)
			}
			found := ""
			for _, kind := range []string{"string", "bool", "int64", "list"} {
				if raw, ok := a[kind]; ok {
					var metadata map[string]json.RawMessage
					if err := json.Unmarshal(raw, &metadata); err != nil {
						panic(err)
					}
					if kind == "list" {
						element := string(metadata["element_type"])
						var shape map[string]json.RawMessage
						if err := json.Unmarshal([]byte(element), &shape); err != nil || len(shape) != 1 || string(shape["string"]) != "{}" {
							panic("unreviewed list element type")
						}
						delete(metadata, "element_type")
					}
					if len(metadata) != 1 {
						panic("unreviewed attribute metadata or enum: " + name)
					}
					var f struct {
						Flags string `json:"computed_optional_required"`
					}
					if e = json.Unmarshal(raw, &f); e != nil {
						panic(e)
					}
					found = kind + "/" + f.Flags
				}
			}
			if found != want {
				panic(fmt.Sprintf("%s: got %s, want %s", name, found, want))
			}
			delete(expected, name)
		}
		if len(expected) != 0 {
			panic(fmt.Sprintf("missing attributes: %v", expected))
		}
	}
	if len(contracts) != 0 {
		panic("missing reviewed resources")
	}
}
