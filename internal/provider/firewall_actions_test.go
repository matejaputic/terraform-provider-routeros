// SPDX-License-Identifier: MPL-2.0
package provider

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"testing"
)

func TestFirewallActionContracts(t *testing.T) {
	for _, makeResource := range collectionConstructors() {
		r := makeResource().(*collectionResource)
		if !r.orderedFirewall() {
			continue
		}
		values := map[string]attr.Value{}
		for _, f := range r.policy.Fields {
			values[f.Name] = nullField(f)
		}
		values["chain"] = types.StringValue("test")
		values["action"] = types.StringValue("return")
		if _, e := r.payload(context.Background(), values); e != nil {
			t.Fatal(e)
		}
		values["action"] = types.StringValue("jump")
		if _, e := r.payload(context.Background(), values); e == nil {
			t.Fatal("unimplemented jump accepted")
		}
		switch r.policy.Name {
		case "ip_firewall_nat":
			values["action"] = types.StringValue("src-nat")
			if _, e := r.payload(context.Background(), values); e == nil {
				t.Fatal("missing NAT target accepted")
			}
			for _, bad := range []string{"::1", "192.0.2.0/24", "host.example", "192.0.2.2-192.0.2.1"} {
				values["to_addresses"] = types.StringValue(bad)
				if _, e := r.payload(context.Background(), values); e == nil {
					t.Fatal("invalid NAT target accepted")
				}
			}
			values["to_addresses"] = types.StringValue("192.0.2.1-192.0.2.2")
			if _, e := r.payload(context.Background(), values); e != nil {
				t.Fatal(e)
			}
			values["action"] = types.StringValue("masquerade")
			if _, e := r.payload(context.Background(), values); e == nil {
				t.Fatal("inapplicable target accepted")
			}
		case "ip_firewall_mangle":
			values["action"] = types.StringValue("mark-connection")
			if _, e := r.payload(context.Background(), values); e == nil {
				t.Fatal("missing mark accepted")
			}
			values["new_connection_mark"] = types.StringValue("test-mark")
			if _, e := r.payload(context.Background(), values); e != nil {
				t.Fatal(e)
			}
			values["action"] = types.StringValue("return")
			if _, e := r.payload(context.Background(), values); e == nil {
				t.Fatal("inapplicable mark accepted")
			}
			values["new_connection_mark"] = types.StringNull()
			values["passthrough"] = types.BoolValue(false)
			if _, e := r.payload(context.Background(), values); e == nil {
				t.Fatal("hidden passthrough intent accepted")
			}
		case "ip_firewall_raw":
			values["action"] = types.StringValue("notrack")
			if _, e := r.payload(context.Background(), values); e != nil {
				t.Fatal(e)
			}
		}
		// A forged catalog path must never authorize an arbitrary /move command.
		r.policy.Path = "/system/script"
		if r.orderedFirewall() {
			t.Fatal("arbitrary action binding accepted")
		}
	}
}
