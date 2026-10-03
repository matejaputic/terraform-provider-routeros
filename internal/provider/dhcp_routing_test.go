// SPDX-License-Identifier: MPL-2.0
package provider

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/matejaputic/terraform-provider-routeros/internal/catalog"
	"github.com/matejaputic/terraform-provider-routeros/internal/client"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDHCPRoutingValidation(t *testing.T) {
	ctx := context.Background()
	for _, makeResource := range collectionConstructors() {
		r := makeResource().(*collectionResource)
		if r.policy.Name != "ip_dhcp_server_network" && r.policy.Name != "ip_dhcp_server_lease" && r.policy.Name != "ip_route" {
			continue
		}
		values := map[string]attr.Value{}
		for _, f := range r.policy.Fields {
			values[f.Name] = nullField(f)
		}
		switch r.policy.Name {
		case "ip_dhcp_server_network":
			values["address"] = types.StringValue("192.0.2.0/24")
			values["dns_server"] = types.ListValueMust(types.StringType, []attr.Value{types.StringValue("192.0.2.53")})
			values["dns_none"] = types.BoolValue(false)
		case "ip_dhcp_server_lease":
			values["address"] = types.StringValue("192.0.2.15")
			values["mac_address"] = types.StringValue("02:00:00:00:00:aa")
		case "ip_route":
			values["dst_address"] = types.StringValue("198.51.100.0/24")
			values["gateway"] = types.StringValue("192.0.2.254")
		}
		if e := r.validate(ctx, values, true); e != nil {
			t.Fatal(e)
		}
		var bad map[string]attr.Value
		switch r.policy.Name {
		case "ip_dhcp_server_network":
			bad = map[string]attr.Value{"address": types.StringValue("192.0.2.1/24"), "gateway": types.StringValue("::1"), "netmask": types.Int64Value(33), "dns_none": types.BoolValue(true), "dns_server": types.ListValueMust(types.StringType, []attr.Value{types.StringNull()})}
		case "ip_dhcp_server_lease":
			bad = map[string]attr.Value{"address": types.StringValue("192.0.2.0/24"), "mac_address": types.StringValue("02:00:00:00:00:00:00:01")}
		case "ip_route":
			bad = map[string]attr.Value{"dst_address": types.StringValue("::/0"), "distance": types.Int64Value(0), "scope": types.Int64Value(256), "target_scope": types.Int64Value(-1)}
		}
		for key, v := range bad {
			old := values[key]
			values[key] = v
			if r.validate(ctx, values, true) == nil {
				t.Errorf("accepted invalid %s.%s", r.policy.Name, key)
			}
			values[key] = old
		}
	}
	field := catalog.Field{Name: "dns_server", Type: "array"}
	for _, bad := range []string{"::1", "192.0.2.1,192.0.2.1", "192.0.2.0/24", ","} {
		if _, e := decodeField(ctx, field, bad); e == nil {
			t.Fatal("invalid DNS CSV accepted")
		}
	}
	v, e := decodeField(ctx, field, "")
	if e != nil || len(v.(types.List).Elements()) != 0 {
		t.Fatal("empty DNS read not typed empty")
	}
}
func TestRouteUpdateRechecksOwnership(t *testing.T) {
	ctx := context.Background()
	writes := 0
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) {
		if q.Method != "GET" {
			writes++
			w.WriteHeader(500)
			return
		}
		w.Write([]byte(`[{".id":"*A","static":"true","dynamic":"true"}]`))
	}))
	defer s.Close()
	c, _ := client.New(client.Config{HostURL: s.URL, Timeout: time.Second})
	defer c.Close()
	r := collectionConstructors()[9]().(*collectionResource)
	r.client = c
	values := map[string]attr.Value{}
	for _, f := range r.policy.Fields {
		values[f.Name] = nullField(f)
	}
	values["id"] = types.StringValue("*A")
	values["dst_address"] = types.StringValue("198.51.100.0/24")
	values["gateway"] = types.StringValue("192.0.2.254")
	obj := r.object(values)
	raw, e := obj.ToTerraformValue(ctx)
	if e != nil {
		t.Fatal(e)
	}
	response := resource.UpdateResponse{State: tfsdk.State{Schema: r.schema}}
	r.Update(ctx, resource.UpdateRequest{State: tfsdk.State{Schema: r.schema, Raw: raw}, Plan: tfsdk.Plan{Schema: r.schema, Raw: raw}, Config: tfsdk.Config{Schema: r.schema, Raw: raw}}, &response)
	if !response.Diagnostics.HasError() || writes != 0 {
		t.Fatal("update wrote an unowned route")
	}
}
func TestRouteOwnershipFailsClosedAndLeaseMACRefresh(t *testing.T) {
	ctx := context.Background()
	for _, wire := range []string{`[{".id":"*A","static":"false"}]`, `[{".id":"*A"}]`, `[{".id":"*A","static":"true","dynamic":"true"}]`} {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) { w.Write([]byte(wire)) }))
		c, _ := client.New(client.Config{HostURL: s.URL, Timeout: time.Second})
		r := collectionConstructors()[9]().(*collectionResource)
		r.client = c
		values := map[string]attr.Value{"id": types.StringValue("*A")}
		if _, _, e := r.refresh(ctx, values); e == nil {
			t.Fatal("unowned route accepted")
		}
		c.Close()
		s.Close()
	}
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) {
		w.Write([]byte(`[{".id":"*A","address":"192.0.2.15","mac-address":"02:00:00:00:00:AA","dynamic":"false"}]`))
	}))
	defer s.Close()
	c, _ := client.New(client.Config{HostURL: s.URL, Timeout: time.Second})
	defer c.Close()
	r := collectionConstructors()[8]().(*collectionResource)
	r.client = c
	values := map[string]attr.Value{}
	for _, f := range r.policy.Fields {
		values[f.Name] = nullField(f)
	}
	values["id"] = types.StringValue("*A")
	values["mac_address"] = types.StringValue("02:00:00:00:00:aa")
	out, found, e := r.refresh(ctx, values)
	if e != nil || !found {
		t.Fatal(e)
	}
	if out["mac_address"].(types.String).ValueString() != "02:00:00:00:00:aa" {
		t.Fatal("equivalent MAC spelling not retained")
	}
	if out["block_access"].(types.Bool).ValueBool() {
		t.Fatal("absent block_access not false")
	}
}
