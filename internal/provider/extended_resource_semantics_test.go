// SPDX-License-Identifier: MPL-2.0
package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/matejaputic/terraform-provider-routeros/internal/catalog"
	"github.com/matejaputic/terraform-provider-routeros/internal/client"
)

func TestExtendedResourceCollectionFailedCreateRetainsAllocatedIdentity(t *testing.T) {
	fixture := newExtendedResourceFixture(t)
	for _, p := range catalog.ExtendedCollections() {
		t.Run(p.Name, func(t *testing.T) {
			r := extendedResource(t, p.Name)
			v := fixture.values(r, 0)
			v["id"] = types.StringNull()
			ctx := context.Background()
			raw, err := r.object(v).ToTerraformValue(ctx)
			if err != nil {
				t.Fatal(err)
			}
			planned := map[string]attr.Value{}
			for _, f := range p.Fields {
				planned[f.Name] = v[f.Name]
				if f.Mode == "computed" {
					switch f.Type {
					case "string":
						planned[f.Name] = types.StringUnknown()
					case "integer":
						planned[f.Name] = types.Int64Unknown()
					case "boolean":
						planned[f.Name] = types.BoolUnknown()
					}
				}
			}
			plan := tfsdk.Plan{Schema: r.schema}
			if d := plan.Set(ctx, r.object(planned)); d.HasError() {
				t.Fatal(d)
			}
			writes := 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) {
				if q.Method == "PUT" {
					writes++
					w.Write([]byte(`{".id":"*A"}`))
					return
				}
				w.WriteHeader(503)
			}))
			defer srv.Close()
			c, err := client.New(client.Config{HostURL: srv.URL, Username: "mock-user", Timeout: time.Second})
			if err != nil {
				t.Fatal(err)
			}
			defer c.Close()
			c.Packages = []string{"routeros", "container"}
			r.client = c
			response := resource.CreateResponse{State: tfsdk.State{Schema: r.schema}}
			r.Create(ctx, resource.CreateRequest{Config: tfsdk.Config{Schema: r.schema, Raw: raw}, Plan: plan}, &response)
			if !response.Diagnostics.HasError() || writes != 1 {
				t.Fatal("failed create read lost recovery diagnostic")
			}
			var recovered types.Object
			if d := response.State.Get(ctx, &recovered); d.HasError() {
				t.Fatal(d)
			}
			if recovered.Attributes()["id"].(types.String).ValueString() != "*A" {
				t.Fatal("allocated identity lost")
			}
			for _, f := range p.Fields {
				value := recovered.Attributes()[f.Name]
				if value.IsUnknown() {
					t.Fatal("unknown partial state")
				}
				if f.Mode == "required" && !value.Equal(v[f.Name]) {
					t.Fatalf("required configuration lost: %s", f.Name)
				}
			}
		})
	}
}
func TestExtendedResourceConstructorRegistrationMatchesFrozenRoster(t *testing.T) {
	raw, err := os.ReadFile("../../schemas/resource-plan.json")
	if err != nil {
		t.Fatal(err)
	}
	var ledger struct {
		Resources []struct {
			Constructor string `json:"constructor"`
			Name        string `json:"canonical_name_proposal"`
		}
	}
	if err = json.Unmarshal(raw, &ledger); err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, constructor := range New("test")().Resources(context.Background()) {
		r := constructor()
		response := resource.MetadataResponse{}
		r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "routeros"}, &response)
		if names[response.TypeName] {
			t.Fatal("duplicate public resource")
		}
		names[response.TypeName] = true
	}
	if len(names) != 125+len(catalog.AdditionalSettings()) || len(ledger.Resources) != 107 {
		t.Fatal("fixed milestone accounting changed")
	}
	seen := map[string]bool{}
	for _, row := range ledger.Resources {
		if !names[row.Name] || seen[row.Constructor] {
			t.Fatal("missing or duplicate frozen constructor")
		}
		seen[row.Constructor] = true
	}
}
func TestExtendedResourceReviewedScalarSemantics(t *testing.T) {
	for _, text := range []string{"NaN:00:00", "Inf:00:00", "-1s", "1garbage", "9999999999999999w", "", "1sbad"} {
		if _, err := routerDuration(text); err == nil {
			t.Fatal("malformed duration accepted")
		}
	}
	a, e := routerDuration("1m")
	b, err := routerDuration("00:01:00")
	if e != nil || err != nil || a != b {
		t.Fatal("equivalent durations differ")
	}
	f := catalog.Field{Name: "policy", Constraint: "user_policy", Codec: "csv-string-set"}
	value := preserveReviewedSpelling(f, types.StringValue("read,ssh"), types.StringValue("ssh,read,!write,!policy"))
	if !value.Equal(types.StringValue("read,ssh")) {
		t.Fatal("equivalent permission readback differs")
	}
	if _, err = userPolicySet("read,!read"); err == nil {
		t.Fatal("contradictory permissions accepted")
	}
	for _, codec := range []string{"auto-decimal"} {
		v, err := decodeField(context.Background(), catalog.Field{Name: "mtu", Type: "integer", Codec: codec}, "auto")
		if err != nil || !v.IsNull() {
			t.Fatal("automatic numeric default became a fabricated number")
		}
	}
	r := extendedResource(t, "interface_veth")
	c, err := client.New(client.Config{HostURL: "http://127.0.0.1:1", Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	c.Packages = []string{"routeros"}
	r.client = c
	response := resource.CreateResponse{}
	r.Create(context.Background(), resource.CreateRequest{}, &response)
	if !response.Diagnostics.HasError() {
		t.Fatal("VETH used a base guest")
	}
}
