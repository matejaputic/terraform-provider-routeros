// SPDX-License-Identifier: MPL-2.0
package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/matejaputic/terraform-provider-routeros/internal/catalog"
	"github.com/matejaputic/terraform-provider-routeros/internal/generated"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestPayload(t *testing.T) {
	m := generated.IpAddressModel{Address: types.StringValue("192.0.2.1/24"), Interface: types.StringValue("ether1"), Disabled: types.BoolValue(false), Comment: types.StringUnknown(), Network: types.StringNull(), Vrf: types.StringValue("main")}
	if e := validate(m); e != nil {
		t.Fatal(e)
	}
	p := payload(m)
	if p["disabled"] != "no" {
		t.Fatal(p)
	}
	if _, ok := p["vrf"]; ok {
		t.Fatal("computed-only vrf serialized")
	}
	if _, ok := p["comment"]; ok {
		t.Fatal("unknown serialized")
	}
	if _, ok := p["network"]; ok {
		t.Fatal("null serialized")
	}
	m.Address = types.StringValue("192.0.2.2")
	if payload(m)["address"] != "192.0.2.2/32" {
		t.Fatal("bare address must not inherit an old netmask")
	}
	m.Address = types.StringValue("invalid")
	if validate(m) == nil {
		t.Fatal("invalid address accepted")
	}
}
func TestProtocolSchema(t *testing.T) {
	s := providerserver.NewProtocol6(New("test")())()
	resp, e := s.GetProviderSchema(context.Background(), &tfprotov6.GetProviderSchemaRequest{})
	if e != nil {
		t.Fatal(e)
	}
	for _, d := range resp.Diagnostics {
		if d.Severity == tfprotov6.DiagnosticSeverityError {
			t.Fatal(d.Summary, d.Detail)
		}
	}
	if len(resp.ResourceSchemas) != 1+len(catalog.Resources()) || resp.ResourceSchemas["routeros_ip_address"] == nil {
		t.Fatal("unexpected resource set")
	}
	for _, policy := range catalog.Resources() {
		if resp.ResourceSchemas["routeros_"+policy.Name] == nil {
			t.Fatalf("missing reviewed resource %s", policy.Name)
		}
	}
	if len(resp.DataSourceSchemas) != 0 {
		t.Fatal("unexpected data source")
	}
}

// A real Terraform CLI drives the Framework protocol against a disposable HTTP
// fixture. This is offline lifecycle coverage, NOT live RouterOS acceptance.
func TestTerraformMockLifecycle(t *testing.T) {
	var mu sync.Mutex
	var row map[string]string
	creates, deletes, updates := 0, 0, 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if u, p, ok := r.BasicAuth(); !ok || u != "admin" || p != "secret" {
			w.WriteHeader(401)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if r.Method == "GET" && r.URL.Path == "/rest/system/resource" {
			json.NewEncoder(w).Encode(map[string]string{"version": "7.24.5 (stable)"})
			return
		}
		if r.Method == "GET" && r.URL.Path == "/rest/system/package" {
			json.NewEncoder(w).Encode([]map[string]string{{"name": "routeros", "version": "7.24.5", "disabled": "false"}})
			return
		}
		switch r.Method {
		case "PUT":
			creates++
			json.NewDecoder(r.Body).Decode(&row)
			row[".id"] = "*2"
			row["network"] = "192.0.2.0"
			row["actual-interface"] = row["interface"]
			row["dynamic"] = "false"
			row["invalid"] = "false"
			row["slave"] = "false"
			json.NewEncoder(w).Encode(row)
		case "GET":
			if row == nil {
				w.Write([]byte("[]"))
				return
			}
			if r.URL.Query().Get(".id") != "*2" {
				t.Error("missing filtered collection read")
			}
			json.NewEncoder(w).Encode([]map[string]string{row})
		case "PATCH":
			updates++
			if r.URL.EscapedPath() != "/rest/ip/address/*2" {
				t.Error("update ID path")
			}
			var p map[string]string
			json.NewDecoder(r.Body).Decode(&p)
			for k, v := range p {
				row[k] = v
			}
			w.WriteHeader(204)
		case "DELETE":
			if r.URL.EscapedPath() != "/rest/ip/address/*2" {
				t.Error("delete ID must retain literal star")
			}
			deletes++
			row = nil
			w.WriteHeader(204)
		default:
			w.WriteHeader(405)
		}
	}))
	defer srv.Close()
	config := func(comment string) string {
		return fmt.Sprintf(`provider "routeros" {
 hosturl = %q
 username = "admin"
 password = "secret"
 }
 resource "routeros_ip_address" "test" {
 address = "192.0.2.1/24"
 interface = "ether1"
 comment = %q
 }`, srv.URL, comment)
	}
	resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){"routeros": providerserver.NewProtocol6WithError(New("test")())}, Steps: []resource.TestStep{
		{Config: config("first"), Check: resource.ComposeTestCheckFunc(resource.TestCheckResourceAttr("routeros_ip_address.test", "id", "*2"), resource.TestCheckResourceAttr("routeros_ip_address.test", "disabled", "false"), resource.TestCheckResourceAttr("routeros_ip_address.test", "actual_interface", "ether1"))},
		{Config: config("second"), Check: resource.TestCheckResourceAttr("routeros_ip_address.test", "comment", "second")},
		{ResourceName: "routeros_ip_address.test", ImportState: true, ImportStateVerify: true},
		{Config: config("second"), PlanOnly: true},
	}})
	mu.Lock()
	defer mu.Unlock()
	if creates != 1 || updates != 1 || deletes != 1 {
		t.Fatalf("operations create=%d update=%d delete=%d", creates, updates, deletes)
	}
}
