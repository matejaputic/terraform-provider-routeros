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
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/matejaputic/terraform-provider-routeros/internal/client"
)

func relayValues(r *collectionResource) map[string]attr.Value {
	v := map[string]attr.Value{}
	for _, f := range r.policy.Fields {
		v[f.Name] = nullField(f)
	}
	v["id"] = types.StringValue("*A")
	v["name"] = types.StringValue("tf-relay")
	v["interface"] = types.StringValue("tf-disconnected")
	v["dhcp_server"] = types.StringValue("192.0.2.1,192.0.2.2")
	return v
}
func TestDHCPRelayReviewedPayload(t *testing.T) {
	r := dhcpOptionResource(t, "ip_dhcp_relay")
	v := relayValues(r)
	v["disabled"] = types.BoolValue(true)
	v["add_relay_info"] = types.BoolValue(false)
	v["local_address"] = types.StringValue("0.0.0.0")
	p, err := r.payload(context.Background(), v)
	if err != nil || p["dhcp-server"] != "192.0.2.1,192.0.2.2" || p["disabled"] != "yes" || p["add-relay-info"] != "no" || len(p) != 6 {
		t.Fatalf("payload: %v %v", p, err)
	}
	for field, bad := range map[string][]attr.Value{
		"dhcp_server":   {types.StringValue(""), types.StringValue("192.0.2.1, 192.0.2.2"), types.StringValue("192.0.2.1,192.0.2.1"), types.StringValue("::1"), types.StringValue("0.0.0.0"), types.StringValue("224.0.0.1"), types.StringUnknown()},
		"local_address": {types.StringValue(""), types.StringValue("192.0.2.1/24"), types.StringValue("::1"), types.StringValue("255.255.255.255")},
		"interface":     {types.StringValue(""), types.StringValue("ether1\n"), types.StringNull()},
	} {
		old := v[field]
		for _, value := range bad {
			v[field] = value
			if _, err := r.payload(context.Background(), v); err == nil {
				t.Errorf("accepted invalid %s", field)
			}
		}
		v[field] = old
	}
}
func TestDHCPRelayAtomicRefreshOwnershipAndFailure(t *testing.T) {
	for _, body := range []string{
		`[{".id":"*A","name":"tf-relay","interface":"tf-disconnected","dhcp-server":"bad"}]`,
		`[{".id":"*A","name":"tf-relay","interface":"tf-disconnected","dhcp-server":"192.0.2.1","disabled":"bad"}]`,
		`[{".id":"*A","name":"tf-relay","interface":"tf-disconnected","dhcp-server":"192.0.2.1","dynamic":"yes"}]`,
		`[{".id":"*A","name":"tf-relay","interface":"tf-disconnected","dhcp-server":"192.0.2.1","builtin":null}]`,
		`[{".id":"*B","name":"tf-relay","interface":"tf-disconnected","dhcp-server":"192.0.2.1"}]`,
		`[{".id":"*A","name":"tf-relay","dhcp-server":"192.0.2.1"}]`,
		`invalid-json`,
	} {
		t.Run(body, func(t *testing.T) {
			writes := 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) {
				if q.Method != "GET" {
					writes++
					w.WriteHeader(500)
					return
				}
				fmt.Fprint(w, body)
			}))
			defer srv.Close()
			c, err := client.New(client.Config{HostURL: srv.URL, Timeout: time.Second})
			if err != nil {
				t.Fatal(err)
			}
			defer c.Close()
			r := dhcpOptionResource(t, "ip_dhcp_relay")
			r.client = c
			v := relayValues(r)
			if next, found, err := r.refresh(context.Background(), v); err == nil || found || next != nil {
				t.Fatal("malformed or unowned read accepted")
			}
			raw, err := r.object(v).ToTerraformValue(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			state := tfsdk.State{Schema: r.schema, Raw: raw}
			u := frameworkresource.UpdateResponse{State: state}
			r.Update(context.Background(), frameworkresource.UpdateRequest{State: state, Plan: tfsdk.Plan{Schema: r.schema, Raw: raw}, Config: tfsdk.Config{Schema: r.schema, Raw: raw}}, &u)
			d := frameworkresource.DeleteResponse{State: state}
			r.Delete(context.Background(), frameworkresource.DeleteRequest{State: state}, &d)
			if !u.Diagnostics.HasError() || !d.Diagnostics.HasError() || writes != 0 {
				t.Fatal("mutation after invalid ownership/read")
			}
		})
	}
}

func TestTerraformDHCPRelayMockLifecycle(t *testing.T) {
	var mu sync.Mutex
	rows := map[string]map[string]string{}
	counter, creates, updates, deletes := 0, 0, 0, 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch q.URL.Path {
		case "/rest/system/resource":
			json.NewEncoder(w).Encode(map[string]string{"version": "7.24.5 (stable)"})
			return
		case "/rest/system/package":
			json.NewEncoder(w).Encode([]map[string]string{{"name": "routeros", "version": "7.24.5"}})
			return
		}
		path := strings.TrimPrefix(q.URL.Path, "/rest")
		id := ""
		if i := strings.LastIndex(path, "/*"); i >= 0 {
			id = path[i+1:]
			path = path[:i]
		}
		if path != "/ip/dhcp-relay" {
			t.Errorf("unexpected path %s", path)
			w.WriteHeader(404)
			return
		}
		if q.Method == "GET" {
			result := []map[string]string{}
			for key, row := range rows {
				if wanted := q.URL.Query().Get(".id"); wanted != "" && wanted != key {
					continue
				}
				if name := q.URL.Query().Get("name"); name != "" && name != row["name"] {
					continue
				}
				result = append(result, row)
			}
			json.NewEncoder(w).Encode(result)
			return
		}
		if q.Method == "DELETE" {
			deletes++
			delete(rows, id)
			w.WriteHeader(204)
			return
		}
		body := map[string]string{}
		if err := json.NewDecoder(q.Body).Decode(&body); err != nil {
			t.Error(err)
			w.WriteHeader(400)
			return
		}
		for key := range body {
			if !oneOf(key, "name", "interface", "dhcp-server", "disabled", "local-address", "add-relay-info") {
				t.Errorf("unreviewed mutation %s", key)
			}
		}
		switch q.Method {
		case "PUT":
			counter++
			creates++
			id = fmt.Sprintf("*%X", counter)
			rows[id] = map[string]string{".id": id, "disabled": "yes", "local-address": "0.0.0.0", "add-relay-info": "no", "invalid": "false"}
		case "PATCH":
			updates++
			if rows[id] == nil {
				w.WriteHeader(404)
				return
			}
			if _, ok := body["name"]; ok {
				t.Error("replacement name in PATCH")
			}
		default:
			w.WriteHeader(405)
			return
		}
		for key, value := range body {
			rows[id][key] = value
		}
		json.NewEncoder(w).Encode(rows[id])
	}))
	defer srv.Close()
	provider := fmt.Sprintf(`provider "routeros" {
 hosturl = %q
 username = "mock-user"
 password = "mock-only"
}`, srv.URL)
	config := func(name, servers string, info bool) string {
		return provider + fmt.Sprintf(`
resource "routeros_ip_dhcp_relay" "test" {
 name = %q
 interface = "tf-disconnected"
 dhcp_server = %q
 disabled = true
 local_address = "0.0.0.0"
 add_relay_info = %t
}`, name, servers, info)
	}
	first := config("tf-relay", "192.0.2.1,192.0.2.2", false)
	changed := config("tf-relay", "192.0.2.3", true)
	renamed := config("tf-relay-new", "192.0.2.3", false)
	var deletedID, oldID string
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){"routeros": providerserver.NewProtocol6WithError(New("test")())},
		Steps: []resource.TestStep{
			{Config: first, Check: resource.TestCheckResourceAttr("routeros_ip_dhcp_relay.test", "disabled", "true")},
			{Config: first, PlanOnly: true},
			{ResourceName: "routeros_ip_dhcp_relay.test", ImportState: true, ImportStateVerify: true},
			{Config: changed, Check: resource.TestCheckResourceAttr("routeros_ip_dhcp_relay.test", "add_relay_info", "true")},
			{Config: changed, PlanOnly: true},
			{Config: changed, PreConfig: func() {
				mu.Lock()
				defer mu.Unlock()
				for _, row := range rows {
					row["dhcp-server"] = "192.0.2.99"
				}
			}, Check: resource.TestCheckResourceAttr("routeros_ip_dhcp_relay.test", "dhcp_server", "192.0.2.3")},
			{Config: changed, PlanOnly: true},
			{Config: changed, PreConfig: func() {
				mu.Lock()
				defer mu.Unlock()
				for id := range rows {
					deletedID = id
					delete(rows, id)
				}
			}, Check: func(s *terraform.State) error {
				if s.RootModule().Resources["routeros_ip_dhcp_relay.test"].Primary.ID == deletedID {
					return fmt.Errorf("deleted relay not recreated")
				}
				return nil
			}},
			{Config: renamed, PreConfig: func() {
				mu.Lock()
				defer mu.Unlock()
				for id := range rows {
					oldID = id
				}
			}, Check: func(s *terraform.State) error {
				if s.RootModule().Resources["routeros_ip_dhcp_relay.test"].Primary.ID == oldID {
					return fmt.Errorf("replacement retained ID")
				}
				return nil
			}},
			{Config: renamed, PlanOnly: true},
			{ResourceName: "routeros_ip_dhcp_relay.test", ImportState: true, ImportStateVerify: true},
		},
		CheckDestroy: func(*terraform.State) error {
			mu.Lock()
			defer mu.Unlock()
			if len(rows) != 0 {
				return fmt.Errorf("owned relay remains")
			}
			return nil
		},
	})
	mu.Lock()
	defer mu.Unlock()
	if creates != 3 || updates < 2 || deletes != 2 {
		t.Fatalf("missing lifecycle operations: %d/%d/%d", creates, updates, deletes)
	}
}
