// SPDX-License-Identifier: MPL-2.0
package provider

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	frameworkprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestCatalogPolicyMatchesActiveFrameworkSchema(t *testing.T) {
	b, err := os.ReadFile("../../schemas/ip-address-policy.json")
	if err != nil {
		t.Fatal(err)
	}
	var policy struct {
		Version int `json:"schema_version"`
		Fields  []struct {
			Name      string  `json:"name"`
			Type      string  `json:"type"`
			Mode      string  `json:"mode"`
			Sensitive bool    `json:"sensitive"`
			ForceNew  bool    `json:"force_new"`
			Validator *string `json:"validators"`
		} `json:"attributes"`
	}
	if err := json.Unmarshal(b, &policy); err != nil {
		t.Fatal(err)
	}
	var response resource.SchemaResponse
	NewIPAddressResource().Schema(context.Background(), resource.SchemaRequest{}, &response)
	if response.Schema.Version != int64(policy.Version) || len(response.Schema.Attributes) != len(policy.Fields) {
		t.Fatal("catalog/schema version or field-set mismatch")
	}
	for _, field := range policy.Fields {
		a, ok := response.Schema.Attributes[field.Name]
		if !ok {
			t.Fatalf("missing %s", field.Name)
		}
		if a.IsRequired() != (field.Mode == "required") || a.IsComputed() != (field.Mode != "required") || a.IsOptional() != (field.Mode == "computed_optional") || a.IsSensitive() != field.Sensitive {
			t.Fatalf("flags differ: %s", field.Name)
		}
		if field.ForceNew {
			t.Fatal("replacement policy has no maintained implementation")
		}
		if field.Type == "boolean" {
			if !a.GetType().Equal(types.BoolType) {
				t.Fatalf("boolean changed: %s", field.Name)
			}
			value := a.(schema.BoolAttribute)
			if value.Default != nil || len(value.PlanModifiers) != 0 {
				t.Fatal("unreviewed default/replacement behavior")
			}
			continue
		}
		if !a.GetType().Equal(types.StringType) {
			t.Fatalf("string changed: %s", field.Name)
		}
		value := a.(schema.StringAttribute)
		if value.Default != nil || len(value.PlanModifiers) != 0 {
			t.Fatal("unreviewed default/replacement behavior")
		}
		if field.Validator == nil {
			if len(value.Validators) != 0 {
				t.Fatalf("unexpected validator: %s", field.Name)
			}
			continue
		}
		if len(value.Validators) != 1 {
			t.Fatalf("missing reviewed validator: %s", field.Name)
		}
		switch field.Name {
		case "address", "network":
			v, ok := value.Validators[0].(ipValidator)
			want := "IP"
			if field.Name == "address" {
				want = "IP or CIDR"
			}
			if !ok || v.cidr != (field.Name == "address") || *field.Validator != want {
				t.Fatal("IP validation policy mismatch")
			}
		case "interface":
			if _, ok := value.Validators[0].(nonemptyValidator); !ok || *field.Validator != "nonempty, no instance enum" {
				t.Fatal("instance enum/validator mismatch")
			}
		default:
			t.Fatalf("unimplemented validator: %s", field.Name)
		}
	}
}

func TestMaintainedProviderPasswordRemainsSensitive(t *testing.T) {
	var response frameworkprovider.SchemaResponse
	New("test")().Schema(context.Background(), frameworkprovider.SchemaRequest{}, &response)
	if !response.Schema.Attributes["password"].IsSensitive() {
		t.Fatal("provider password lost sensitivity")
	}
}
