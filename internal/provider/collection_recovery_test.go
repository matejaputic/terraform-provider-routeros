// SPDX-License-Identifier: MPL-2.0
package provider

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/matejaputic/terraform-provider-routeros/internal/client"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCollectionRecoveryAndConditionalConfiguration(t *testing.T) {
	ctx := context.Background()
	r := collectionConstructors()[5]().(*collectionResource)
	values := map[string]attr.Value{}
	for _, f := range r.policy.Fields {
		values[f.Name] = nullField(f)
	}
	values["name"] = types.StringValue("tf-recovery")
	values["ranges"] = types.ListValueMust(types.StringType, []attr.Value{types.StringValue("192.0.2.10-192.0.2.20")})
	raw, e := r.object(values).ToTerraformValue(ctx)
	if e != nil {
		t.Fatal(e)
	}
	planned := map[string]attr.Value{}
	for key, v := range values {
		planned[key] = v
	}
	planned["id"] = types.StringUnknown()
	planned["comment"] = types.StringUnknown()
	planned["next_pool"] = types.StringUnknown()
	plan := tfsdk.Plan{Schema: r.schema}
	if d := plan.Set(ctx, r.object(planned)); d.HasError() {
		t.Fatal(d)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) {
		if q.Method == "PUT" {
			w.Write([]byte(`{".id":"*A"}`))
			return
		}
		w.WriteHeader(503)
	}))
	defer srv.Close()
	c, _ := client.New(client.Config{HostURL: srv.URL, Timeout: time.Second})
	defer c.Close()
	r.client = c
	response := resource.CreateResponse{State: tfsdk.State{Schema: r.schema}}
	r.Create(ctx, resource.CreateRequest{Config: tfsdk.Config{Schema: r.schema, Raw: raw}, Plan: plan}, &response)
	if !response.Diagnostics.HasError() {
		t.Fatal("failed follow-up read lost error")
	}
	var recovered types.Object
	if d := response.State.Get(ctx, &recovered); d.HasError() {
		t.Fatal(d)
	}
	if recovered.Attributes()["id"].(types.String).ValueString() != "*A" {
		t.Fatal("created ID was lost")
	}
	for _, v := range recovered.Attributes() {
		if v.IsUnknown() {
			t.Fatal("unknown recovery state")
		}
	}
	bridge := collectionConstructors()[0]().(*collectionResource)
	cfg := map[string]attr.Value{}
	for _, f := range bridge.policy.Fields {
		cfg[f.Name] = nullField(f)
	}
	cfg["name"] = types.StringValue("b")
	cfg["pvid"] = types.Int64Value(10)
	cfg["vlan_filtering"] = types.BoolValue(false)
	if bridge.validate(ctx, cfg, false) == nil {
		t.Fatal("unreadable bridge PVID accepted")
	}
	cfg["vlan_filtering"] = types.BoolValue(true)
	if e := bridge.validate(ctx, cfg, true); e != nil {
		t.Fatal(e)
	}
}
func TestStrictCollectionBooleanTokens(t *testing.T) {
	for _, v := range []any{nil, "", "unknown", 1.0, map[string]any{}} {
		if _, e := strictWireBool(v); e == nil {
			t.Fatal("invalid ownership boolean accepted")
		}
	}
}
