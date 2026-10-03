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
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/matejaputic/terraform-provider-routeros/internal/client"
)

func dnsRecordValues(r *collectionResource) map[string]attr.Value {
	v := map[string]attr.Value{}
	for _, f := range r.policy.Fields {
		v[f.Name] = nullField(f)
	}
	v["id"] = types.StringValue("*A")
	v["name"] = types.StringValue("tf-record.invalid")
	v["type"] = types.StringValue("A")
	v["address"] = types.StringValue("192.0.2.1")
	return v
}
func TestDNSRecordReviewedValidation(t *testing.T) {
	r := dhcpOptionResource(t, "ip_dns_record")
	v := dnsRecordValues(r)
	if p, err := r.payload(context.Background(), v); err != nil || len(p) != 3 || p["type"] != "A" {
		t.Fatalf("payload: %v %v", p, err)
	}
	for field, entries := range map[string][]string{"name": {"UPPER.invalid", "a..invalid", "-a.invalid", "a_.invalid", "a.invalid."}, "type": {"CNAME", "FWD", "a"}, "address": {"::1", "bad", "192.0.2.1/24", "192.000.2.1"}} {
		old := v[field]
		for _, entry := range entries {
			v[field] = types.StringValue(entry)
			if _, err := r.payload(context.Background(), v); err == nil {
				t.Errorf("accepted %s", field)
			}
		}
		v[field] = old
	}
	v["type"] = types.StringValue("AAAA")
	v["address"] = types.StringValue("2001:db8::1")
	if _, err := r.payload(context.Background(), v); err != nil {
		t.Fatal(err)
	}
}
func TestDNSRecordAtomicReadAndOwnership(t *testing.T) {
	for _, extra := range []string{`,"regexp":".*"`, `,"address-list":"unsafe"`, `,"dynamic":"true"`, `,"builtin":null`, `,"address":false`, `,"type":"FWD"`} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) {
			fmt.Fprintf(w, `[{".id":"*A","name":"tf-record.invalid","type":"A","address":"192.0.2.1"%s}]`, extra)
		}))
		c, err := client.New(client.Config{HostURL: srv.URL, Timeout: time.Second})
		if err != nil {
			t.Fatal(err)
		}
		r := dhcpOptionResource(t, "ip_dns_record")
		r.client = c
		v := dnsRecordValues(r)
		if next, found, err := r.refresh(context.Background(), v); err == nil || found || next != nil {
			t.Errorf("unsafe read accepted: %s", extra)
		}
		if v["address"].(types.String).ValueString() != "192.0.2.1" {
			t.Fatal("failed read altered prior state")
		}
		c.Close()
		srv.Close()
	}
}
func TestTerraformDNSRecordMockLifecycle(t *testing.T) {
	var mu sync.Mutex
	rows := map[string]map[string]string{}
	counter, creates, updates, deletes := 0, 0, 0, 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if q.URL.Path == "/rest/system/resource" {
			json.NewEncoder(w).Encode(map[string]string{"version": "7.24.5 (stable)"})
			return
		}
		if q.URL.Path == "/rest/system/package" {
			json.NewEncoder(w).Encode([]map[string]string{{"name": "routeros", "version": "7.24.5"}})
			return
		}
		path := strings.TrimPrefix(q.URL.Path, "/rest")
		id := ""
		if i := strings.LastIndex(path, "/*"); i >= 0 {
			id = path[i+1:]
			path = path[:i]
		}
		if path != "/ip/dns/static" {
			t.Errorf("unexpected path %s", path)
			w.WriteHeader(404)
			return
		}
		if q.Method == "GET" {
			out := []map[string]string{}
			for key, row := range rows {
				if x := q.URL.Query().Get(".id"); x != "" && x != key {
					continue
				}
				if x := q.URL.Query().Get("name"); x != "" && x != row["name"] {
					continue
				}
				out = append(out, row)
			}
			json.NewEncoder(w).Encode(out)
			return
		}
		if q.Method == "DELETE" {
			deletes++
			delete(rows, id)
			w.WriteHeader(204)
			return
		}
		p := map[string]string{}
		if err := json.NewDecoder(q.Body).Decode(&p); err != nil {
			t.Error(err)
			w.WriteHeader(400)
			return
		}
		for key := range p {
			if !oneOf(key, "name", "type", "address", "comment", "disabled", "match-subdomain") {
				t.Errorf("unexpected mutation field %s", key)
			}
		}
		switch q.Method {
		case "PUT":
			counter++
			creates++
			id = fmt.Sprintf("*%X", counter)
			rows[id] = map[string]string{".id": id, "dynamic": "false", "disabled": "yes", "match-subdomain": "false"}
		case "PATCH":
			updates++
			if rows[id] == nil {
				w.WriteHeader(404)
				return
			}
			for _, key := range []string{"name", "type"} {
				if _, ok := p[key]; ok {
					t.Errorf("replacement field %s in PATCH", key)
				}
			}
		default:
			w.WriteHeader(405)
			return
		}
		for key, value := range p {
			rows[id][key] = value
		}
		if rows[id]["comment"] == "" {
			delete(rows[id], "comment")
		}
		if oneOf(rows[id]["match-subdomain"], "no", "false") {
			delete(rows[id], "match-subdomain")
		}
		json.NewEncoder(w).Encode(rows[id])
	}))
	defer srv.Close()
	config := func(kind, address, comment string) string {
		return fmt.Sprintf(`provider "routeros" {
 hosturl = %q
 username = "mock-user"
 password = "mock-only"
}
resource "routeros_ip_dns_record" "test" {
 name = "tf-record.invalid"
 type = %q
 address = %q
 comment = %q
 disabled = true
 match_subdomain = %t
}`, srv.URL, kind, address, comment, kind == "A" && comment == "")
	}
	first := config("A", "192.0.2.1", "first")
	changed := config("A", "192.0.2.2", "")
	replaced := config("AAAA", "2001:db8::1", "")
	var deletedID, oldID string
	resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){"routeros": providerserver.NewProtocol6WithError(New("test")())}, Steps: []resource.TestStep{
		{Config: first, Check: resource.TestCheckResourceAttr("routeros_ip_dns_record.test", "address", "192.0.2.1")},
		{Config: first, PlanOnly: true},
		{ResourceName: "routeros_ip_dns_record.test", ImportState: true, ImportStateVerify: true},
		{Config: changed, Check: resource.ComposeTestCheckFunc(resource.TestCheckResourceAttr("routeros_ip_dns_record.test", "address", "192.0.2.2"), resource.TestCheckResourceAttr("routeros_ip_dns_record.test", "comment", ""))},
		{Config: changed, PlanOnly: true},
		{Config: changed, PreConfig: func() {
			mu.Lock()
			defer mu.Unlock()
			for _, row := range rows {
				row["address"] = "192.0.2.99"
			}
		}, Check: resource.TestCheckResourceAttr("routeros_ip_dns_record.test", "address", "192.0.2.2")},
		{Config: changed, PreConfig: func() {
			mu.Lock()
			defer mu.Unlock()
			for id := range rows {
				deletedID = id
				delete(rows, id)
			}
		}, Check: func(s *terraform.State) error {
			if s.RootModule().Resources["routeros_ip_dns_record.test"].Primary.ID == deletedID {
				return fmt.Errorf("deleted DNS record not recreated")
			}
			return nil
		}},
		{Config: replaced, PreConfig: func() {
			mu.Lock()
			defer mu.Unlock()
			for id := range rows {
				oldID = id
			}
		}, Check: func(s *terraform.State) error {
			if s.RootModule().Resources["routeros_ip_dns_record.test"].Primary.ID == oldID {
				return fmt.Errorf("type change did not replace")
			}
			return nil
		}},
		{Config: replaced, PlanOnly: true},
		{ResourceName: "routeros_ip_dns_record.test", ImportState: true, ImportStateVerify: true},
	}, CheckDestroy: func(*terraform.State) error {
		mu.Lock()
		defer mu.Unlock()
		if len(rows) != 0 {
			return fmt.Errorf("DNS record remains")
		}
		return nil
	}})
	mu.Lock()
	defer mu.Unlock()
	if creates != 3 || updates < 2 || deletes != 2 {
		t.Fatalf("missing lifecycle: %d/%d/%d", creates, updates, deletes)
	}
}
