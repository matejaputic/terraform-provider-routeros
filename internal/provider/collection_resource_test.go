// SPDX-License-Identifier: MPL-2.0
package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	frameworkresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/matejaputic/terraform-provider-routeros/internal/client"
)

func networkingConfig(provider, comment string) string {
	return provider + fmt.Sprintf(`
resource "routeros_interface_bridge" "test" {
 name = "tf-coverage-bridge"
 comment = %[1]q
 protocol_mode = "none"
 vlan_filtering = true
 pvid = 1
}
resource "routeros_interface_bridge_port" "test" {
 bridge = routeros_interface_bridge.test.name
 interface = "tf-port"
 comment = %[1]q
 pvid = 10
 edge = "yes"
 hw = false
}
resource "routeros_interface_vlan" "test" {
 name = "tf-coverage-vlan"
 interface = routeros_interface_bridge.test.name
 vlan_id = 123
 comment = %[1]q
 use_service_tag = false
 mtu = 1500
}
resource "routeros_interface_list" "test" {
 name = "tf-coverage-list"
 comment = %[1]q
 include = ""
 exclude = ""
}
resource "routeros_interface_list_member" "test" {
 list = routeros_interface_list.test.name
 interface = routeros_interface_vlan.test.name
 comment = %[1]q
 disabled = false
}
resource "routeros_ip_pool" "test" {
 name = "tf-coverage-pool"
 ranges = ["192.0.2.10-192.0.2.20", "192.0.2.30-192.0.2.40"]
 comment = %[1]q
 next_pool = "none"
}
`, comment)
}

var collectionTestNames = []string{"interface_bridge", "interface_bridge_port", "interface_vlan", "interface_list", "interface_list_member", "ip_pool"}

func TestTerraformCollectionsMockLifecycle(t *testing.T) {
	var mu sync.Mutex
	rows := map[string]map[string]string{}
	creates, updates, deletes := map[string]int{}, map[string]int{}, map[string]int{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if u, p, ok := r.BasicAuth(); !ok || u != "admin" || p != "secret" {
			w.WriteHeader(401)
			return
		}
		if r.URL.Path == "/rest/system/resource" {
			json.NewEncoder(w).Encode(map[string]string{"version": "7.24.5 (stable)"})
			return
		}
		if r.URL.Path == "/rest/system/package" {
			json.NewEncoder(w).Encode([]map[string]string{{"name": "routeros", "version": "7.24.5"}})
			return
		}
		path := strings.TrimPrefix(r.URL.Path, "/rest")
		if i := strings.LastIndex(path, "/*"); i >= 0 {
			if path[i:] != "/*A" {
				t.Error("incorrect item ID")
			}
			path = path[:i]
		}
		switch r.Method {
		case "PUT":
			row := map[string]string{}
			if e := json.NewDecoder(r.Body).Decode(&row); e != nil {
				t.Error(e)
			}
			if _, ok := row[".id"]; ok {
				t.Error("computed ID leaked into payload")
			}
			row[".id"] = "*A"
			if path != "/ip/pool" {
				if row["disabled"] == "" && path != "/interface/list" {
					row["disabled"] = "false"
				}
				if path == "/interface/list" {
					row["builtin"] = "false"
					row["dynamic"] = "false"
				}
				if path == "/interface/list/member" || path == "/interface/bridge/port" {
					row["dynamic"] = "false"
				}
				if path == "/interface/bridge/port" {
					row["inactive"] = "false"
				}
				if path == "/interface/bridge" || path == "/interface/vlan" {
					row["running"] = "true"
				}
			}
			if strings.HasPrefix(path, "/ip/dhcp-server") || path == "/ip/route" {
				row["dynamic"] = "false"
			}
			if path == "/ip/dhcp-server" {
				row["invalid"] = "false"
			}
			if path == "/ip/dhcp-server/lease" {
				row["status"] = "waiting"
			}
			if path == "/ip/route" {
				row["static"] = "true"
				row["active"] = "false"
			}
			if path == "/ip/address" {
				row["actual-interface"] = row["interface"]
				row["network"] = "192.0.2.0"
				row["dynamic"] = "false"
				row["invalid"] = "false"
				row["slave"] = "false"
			}
			rows[path] = row
			creates[path]++
			json.NewEncoder(w).Encode(row)
		case "GET":
			if r.URL.Query().Get(".id") != "*A" {
				t.Error("missing collection ID filter")
			}
			if rows[path] == nil {
				w.Write([]byte(`[]`))
			} else {
				json.NewEncoder(w).Encode([]map[string]string{rows[path]})
			}
		case "PATCH":
			body := map[string]string{}
			if e := json.NewDecoder(r.Body).Decode(&body); e != nil {
				t.Error(e)
			}
			for k, v := range body {
				if k == ".id" || k == "running" || k == "dynamic" || k == "builtin" || k == "inactive" {
					t.Error("computed field echoed")
				}
				rows[path][k] = v
			}
			updates[path]++
			w.WriteHeader(204)
		case "DELETE":
			delete(rows, path)
			deletes[path]++
			w.WriteHeader(204)
		default:
			w.WriteHeader(405)
		}
	}))
	defer srv.Close()
	provider := fmt.Sprintf(`provider "routeros" {
 hosturl = %q
 username = "admin"
 password = "secret"
}`, srv.URL)
	steps := []resource.TestStep{{Config: dhcpRoutingConfig(provider, "first")}, {Config: dhcpRoutingConfig(provider, "first"), PlanOnly: true}, {Config: dhcpRoutingConfig(provider, "second")}}
	for _, name := range append(collectionTestNames, dhcpRoutingNames...) {
		steps = append(steps, resource.TestStep{ResourceName: "routeros_" + name + ".test", ImportState: true, ImportStateVerify: true})
	}
	steps = append(steps, resource.TestStep{Config: dhcpRoutingConfig(provider, "second"), PlanOnly: true})
	resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){"routeros": providerserver.NewProtocol6WithError(New("test")())}, Steps: steps})
	mu.Lock()
	defer mu.Unlock()
	if len(rows) != 0 {
		t.Fatal("destroy left objects")
	}
	for path, n := range creates {
		if n != 1 || updates[path] != 1 || deletes[path] != 1 {
			t.Errorf("%s CRUD counts %d/%d/%d", path, n, updates[path], deletes[path])
		}
	}
}
func TestCollectionSchemaMatchesReviewedCatalog(t *testing.T) {
	for _, constructor := range collectionConstructors() {
		r := constructor().(*collectionResource)
		var s frameworkresource.SchemaResponse
		r.Schema(context.Background(), frameworkresource.SchemaRequest{}, &s)
		if len(s.Schema.Attributes) != len(r.policy.Fields) || s.Schema.Version != 0 {
			t.Fatal("schema/catalog mismatch")
		}
		for _, f := range r.policy.Fields {
			a := s.Schema.Attributes[f.Name]
			if a == nil || !a.GetType().Equal(r.attributeTypes()[f.Name]) || a.IsRequired() != (f.Mode == "required") || a.IsComputed() != (f.Mode != "required") || a.IsOptional() != (f.Mode == "computed_optional") || a.IsSensitive() {
				t.Fatalf("schema drift %s.%s", r.policy.Name, f.Name)
			}
			if f.ForceNew {
				if len(a.(schema.StringAttribute).PlanModifiers) != 1 {
					t.Fatal("missing maintained replacement policy")
				}
			}
		}
	}
}
func TestPoolRangesAndWireCodecs(t *testing.T) {
	for _, values := range [][]string{nil, {"192.0.2.20-192.0.2.10"}, {"2001:db8::1"}, {"192.0.2.1-192.0.2.10", "192.0.2.10-192.0.2.20"}, {"192.0.2.1", "192.0.2.1"}, {" 192.0.2.1"}} {
		if validateRanges(values) == nil {
			t.Fatal("invalid ranges accepted")
		}
	}
	old := types.ListValueMust(types.StringType, []attr.Value{types.StringValue("192.0.2.30"), types.StringValue("192.0.2.10-192.0.2.20")})
	observed := types.ListValueMust(types.StringType, []attr.Value{types.StringValue("192.0.2.10-192.0.2.20"), types.StringValue("192.0.2.30-192.0.2.30")})
	if !preserveRanges(old, observed).Equal(old) {
		t.Fatal("equivalent server ordering caused drift")
	}
	changed := types.ListValueMust(types.StringType, []attr.Value{types.StringValue("192.0.2.11-192.0.2.20"), types.StringValue("192.0.2.30")})
	if preserveRanges(old, changed).Equal(old) {
		t.Fatal("real range drift hidden")
	}
}
func TestCollectionBuiltinOwnershipAndAtomicRead(t *testing.T) {
	constructor := collectionConstructors()[3]
	r := constructor().(*collectionResource)
	for _, body := range []string{`[{".id":"*A","name":"all","builtin":"true"}]`, `[{".id":"*A","name":"x","builtin":"false","dynamic":"bad"}]`, `null`} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) {
			if q.Method != http.MethodGet {
				t.Error("ownership failure authorized mutation")
			}
			w.Write([]byte(body))
		}))
		c, _ := client.New(client.Config{HostURL: srv.URL, Timeout: time.Second})
		r.client = c
		values := map[string]attr.Value{}
		for _, f := range r.policy.Fields {
			values[f.Name] = nullField(f)
		}
		values["id"] = types.StringValue("*A")
		values["name"] = types.StringValue("original")
		_, found, e := r.refresh(context.Background(), values)
		if e == nil || found || values["name"].(types.String).ValueString() != "original" {
			t.Fatal("malformed/built-in read mutated state")
		}
		c.Close()
		srv.Close()
	}
}
