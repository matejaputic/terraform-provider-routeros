// SPDX-License-Identifier: MPL-2.0
package provider

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/matejaputic/terraform-provider-routeros/internal/client"
	"github.com/matejaputic/terraform-provider-routeros/internal/generated"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCreateReadFailurePreservesKnownRecoveryID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			w.Write([]byte(`{".id":"*A"}`))
			return
		}
		w.WriteHeader(503)
	}))
	defer srv.Close()
	c, e := client.New(client.Config{HostURL: srv.URL, Timeout: time.Second})
	if e != nil {
		t.Fatal(e)
	}
	defer c.Close()
	r := ipAddressResource{client: c}
	var s resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &s)
	config := generated.IpAddressModel{Address: types.StringValue("192.0.2.1/24"), Interface: types.StringValue("tf-test")}
	state := tfsdk.State{Schema: s.Schema}
	if d := state.Set(context.Background(), config); d.HasError() {
		t.Fatal(d)
	}
	model := config
	model.Id = types.StringUnknown()
	model.Comment = types.StringUnknown()
	model.Network = types.StringUnknown()
	model.Disabled = types.BoolUnknown()
	model.ActualInterface = types.StringUnknown()
	model.Dynamic = types.BoolUnknown()
	model.Invalid = types.BoolUnknown()
	model.Slave = types.BoolUnknown()
	model.Vrf = types.StringUnknown()
	plan := tfsdk.Plan{Schema: s.Schema}
	if d := plan.Set(context.Background(), model); d.HasError() {
		t.Fatal(d)
	}
	response := resource.CreateResponse{State: tfsdk.State{Schema: s.Schema}}
	r.Create(context.Background(), resource.CreateRequest{Plan: plan, Config: tfsdk.Config{Schema: s.Schema, Raw: state.Raw}}, &response)
	if !response.Diagnostics.HasError() {
		t.Fatal("failed read lost diagnostic")
	}
	var recovered generated.IpAddressModel
	if d := response.State.Get(context.Background(), &recovered); d.HasError() {
		t.Fatal(d)
	}
	if recovered.Id.ValueString() != "*A" || recovered.Comment.IsUnknown() || recovered.Disabled.IsUnknown() || !recovered.Vrf.IsNull() {
		t.Fatal("unrecoverable partial state")
	}
}
