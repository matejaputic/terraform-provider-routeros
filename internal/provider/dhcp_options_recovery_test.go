// SPDX-License-Identifier: MPL-2.0
package provider

import (
	"context"
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

func TestDHCPOptionFailedCreateReadRecovery(t *testing.T) {
	for _, name := range dhcpOptionNames {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			r := dhcpOptionResource(t, name)
			config := dhcpOptionValues(r)
			config["id"] = types.StringNull()
			raw, err := r.object(config).ToTerraformValue(ctx)
			if err != nil {
				t.Fatal(err)
			}
			planned := map[string]attr.Value{}
			for _, field := range r.policy.Fields {
				planned[field.Name] = config[field.Name]
				if field.Mode != "required" {
					planned[field.Name] = nullField(field)
					switch field.Type {
					case "string":
						planned[field.Name] = types.StringUnknown()
					case "boolean":
						planned[field.Name] = types.BoolUnknown()
					}
				}
			}
			plan := tfsdk.Plan{Schema: r.schema}
			if d := plan.Set(ctx, r.object(planned)); d.HasError() {
				t.Fatal(d)
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) {
				if q.Method == "PUT" {
					w.Write([]byte(`{".id":"*A"}`))
					return
				}
				w.WriteHeader(503)
			}))
			defer server.Close()
			c, err := client.New(client.Config{HostURL: server.URL, Timeout: time.Second})
			if err != nil {
				t.Fatal(err)
			}
			defer c.Close()
			r.client = c
			response := resource.CreateResponse{State: tfsdk.State{Schema: r.schema}}
			r.Create(ctx, resource.CreateRequest{Config: tfsdk.Config{Schema: r.schema, Raw: raw}, Plan: plan}, &response)
			if !response.Diagnostics.HasError() {
				t.Fatal("failed follow-up read lost diagnostic")
			}
			var recovered types.Object
			if d := response.State.Get(ctx, &recovered); d.HasError() {
				t.Fatal(d)
			}
			values := recovered.Attributes()
			if values["id"].(types.String).ValueString() != "*A" || values["code"].(types.Int64).ValueInt64() != 77 || values["value"].(types.String).ValueString() != "0x01" {
				t.Fatal("create lost recoverable identity/configuration")
			}
			for _, value := range values {
				if value.IsUnknown() {
					t.Fatal("unknown recovery state")
				}
			}
		})
	}
}
