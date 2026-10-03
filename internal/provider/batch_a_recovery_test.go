// SPDX-License-Identifier: MPL-2.0
package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/matejaputic/terraform-provider-routeros/internal/client"
)

func TestBatchAIntegratedFailureRecovery(t *testing.T) {
	for _, name := range []string{"ip_dhcp_relay", "ip_dns_record"} {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			r := dhcpOptionResource(t, name)
			values := relayValues(r)
			if name == "ip_dns_record" {
				values = dnsRecordValues(r)
			}
			values["id"] = types.StringNull()
			raw, err := r.object(values).ToTerraformValue(ctx)
			if err != nil {
				t.Fatal(err)
			}
			planned := map[string]attr.Value{}
			for _, f := range r.policy.Fields {
				planned[f.Name] = values[f.Name]
				if f.Mode != "required" {
					switch f.Type {
					case "string":
						planned[f.Name] = types.StringUnknown()
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
			c, err := client.New(client.Config{HostURL: srv.URL, Timeout: time.Second})
			if err != nil {
				t.Fatal(err)
			}
			defer c.Close()
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
				t.Fatal("lost allocated identity")
			}
			for _, f := range r.policy.Fields {
				v := recovered.Attributes()[f.Name]
				if v.IsUnknown() {
					t.Fatal("unknown partial recovery state")
				}
				if f.Mode == "required" && !v.Equal(values[f.Name]) {
					t.Fatalf("lost required configuration %s", f.Name)
				}
			}
		})
		t.Run(name+"-mutation-failure", func(t *testing.T) {
			ctx := context.Background()
			r := dhcpOptionResource(t, name)
			v := relayValues(r)
			if name == "ip_dns_record" {
				v = dnsRecordValues(r)
			}
			raw, err := r.object(v).ToTerraformValue(ctx)
			if err != nil {
				t.Fatal(err)
			}
			row := map[string]string{".id": "*A"}
			for _, f := range r.policy.Fields {
				if value, ok := v[f.Name].(types.String); ok && !value.IsNull() && f.Name != "id" {
					row[f.Wire] = value.ValueString()
				}
			}
			writes := 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) {
				if q.Method == "GET" {
					json.NewEncoder(w).Encode([]map[string]string{row})
					return
				}
				writes++
				w.WriteHeader(503)
			}))
			defer srv.Close()
			c, err := client.New(client.Config{HostURL: srv.URL, Timeout: time.Second})
			if err != nil {
				t.Fatal(err)
			}
			defer c.Close()
			r.client = c
			state := tfsdk.State{Schema: r.schema, Raw: raw}
			u := resource.UpdateResponse{State: state}
			r.Update(ctx, resource.UpdateRequest{State: state, Config: tfsdk.Config{Schema: r.schema, Raw: raw}, Plan: tfsdk.Plan{Schema: r.schema, Raw: raw}}, &u)
			d := resource.DeleteResponse{State: state}
			r.Delete(ctx, resource.DeleteRequest{State: state}, &d)
			if !u.Diagnostics.HasError() || !d.Diagnostics.HasError() || writes != 2 {
				t.Fatal("mutation failure lost diagnostic")
			}
			var after types.Object
			if diags := u.State.Get(ctx, &after); diags.HasError() {
				t.Fatal(diags)
			}
			if !after.Equal(r.object(v)) {
				t.Fatal("failed update altered prior state")
			}
		})
	}
}
