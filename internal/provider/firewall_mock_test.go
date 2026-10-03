// SPDX-License-Identifier: MPL-2.0
package provider

import (
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestTerraformFirewallMockLifecycle(t *testing.T) {
	for _, family := range []string{"filter", "nat", "mangle", "raw"} {
		t.Run(family, func(t *testing.T) { testFirewallMock(t, family) })
	}
}
func testFirewallMock(t *testing.T, family string) {
	var mu sync.Mutex
	rows := map[string][]map[string]string{}
	counter := 0
	moves := 0
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		path := strings.TrimPrefix(q.URL.Path, "/rest")
		if path == "/system/resource" {
			w.Write([]byte(`{"version":"7.24.5"}`))
			return
		}
		if path == "/system/package" {
			w.Write([]byte(`[{"name":"routeros","version":"7.24.5"}]`))
			return
		}
		if path == "/ip/firewall/"+family+"/move" {
			var b map[string]string
			json.NewDecoder(q.Body).Decode(&b)
			all := rows["/ip/firewall/"+family]
			var node map[string]string
			kept := []map[string]string{}
			for _, v := range all {
				if v[".id"] == b["numbers"] {
					node = v
				} else {
					kept = append(kept, v)
				}
			}
			if node == nil {
				t.Error("missing move subject")
				w.WriteHeader(400)
				return
			}
			placed := false
			out := []map[string]string{}
			for _, v := range kept {
				if v[".id"] == b["destination"] {
					out = append(out, node)
					placed = true
				}
				out = append(out, v)
			}
			if !placed {
				out = append(out, node)
			}
			rows["/ip/firewall/"+family] = out
			moves++
			w.WriteHeader(204)
			return
		}
		id := ""
		if i := strings.LastIndex(path, "/*"); i >= 0 {
			id = path[i+1:]
			path = path[:i]
		}
		switch q.Method {
		case "PUT":
			var b map[string]string
			json.NewDecoder(q.Body).Decode(&b)
			if _, ok := b["place-before"]; ok {
				t.Error("placement leaked into mutation")
			}
			counter++
			b[".id"] = fmt.Sprintf("*%X", counter)
			b["dynamic"] = "false"
			if path == "/ip/firewall/"+family {
				b["invalid"] = "false"
			}
			rows[path] = append(rows[path], b)
			json.NewEncoder(w).Encode(b)
		case "GET":
			out := []map[string]string{}
			for _, row := range rows[path] {
				if wanted := q.URL.Query().Get(".id"); wanted == "" || row[".id"] == wanted {
					out = append(out, row)
				}
			}
			json.NewEncoder(w).Encode(out)
		case "PATCH":
			var b map[string]string
			json.NewDecoder(q.Body).Decode(&b)
			if _, ok := b["place-before"]; ok {
				t.Error("placement leaked into PATCH")
			}
			for _, row := range rows[path] {
				if row[".id"] == id {
					for k, v := range b {
						row[k] = v
					}
				}
			}
			w.WriteHeader(204)
		case "DELETE":
			out := []map[string]string{}
			for _, row := range rows[path] {
				if row[".id"] != id {
					out = append(out, row)
				}
			}
			rows[path] = out
			w.WriteHeader(204)
		default:
			w.WriteHeader(405)
		}
	}))
	defer s.Close()
	p := fmt.Sprintf("provider \"routeros\" {\n hosturl = %q\n username = \"admin\"\n password = \"secret\"\n}\n", s.URL)
	cfg := func(comment, before string) string { return firewallFamilyConfig(family, p, comment, before) }
	anchor := "routeros_ip_firewall_" + family + ".tail.id"
	steps := []resource.TestStep{{Config: cfg("first", anchor)}, {Config: cfg("first", anchor), PlanOnly: true}, {Config: cfg("second", anchor)}}
	for _, name := range []string{"routeros_ip_firewall_addr_list.test", "routeros_ip_firewall_" + family + ".head", "routeros_ip_firewall_" + family + ".tail"} {
		steps = append(steps, resource.TestStep{ResourceName: name, ImportState: true, ImportStateVerify: true})
	}
	steps = append(steps, resource.TestStep{Config: cfg("second", `""`)}, resource.TestStep{Config: cfg("second", `""`), PlanOnly: true}, resource.TestStep{Config: cfg("second", anchor)}, resource.TestStep{Config: cfg("second", anchor), PlanOnly: true})
	resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){"routeros": providerserver.NewProtocol6WithError(New("test")())}, Steps: steps})
	mu.Lock()
	defer mu.Unlock()
	if moves < 3 {
		t.Fatal("ordering was not exercised")
	}
	for _, r := range rows {
		if len(r) > 0 {
			t.Fatal("destroy left firewall objects")
		}
	}
}
