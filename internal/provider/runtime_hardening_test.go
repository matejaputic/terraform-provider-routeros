// SPDX-License-Identifier: MPL-2.0
package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	frameworkprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/matejaputic/terraform-provider-routeros/internal/client"
	"github.com/matejaputic/terraform-provider-routeros/internal/generated"
)

func aliasConfig(t *testing.T, url, password string) (frameworkprovider.Provider, frameworkprovider.ConfigureRequest) {
	t.Helper()
	p := New("test")()
	var s frameworkprovider.SchemaResponse
	p.Schema(context.Background(), frameworkprovider.SchemaRequest{}, &s)
	object, d := types.ObjectValueFrom(context.Background(), map[string]attr.Type{"hosturl": types.StringType, "username": types.StringType, "password": types.StringType, "ca_certificate": types.StringType, "insecure": types.BoolType, "rest_timeout": types.Int64Type}, configModel{HostURL: types.StringValue(url), Username: types.StringValue("admin"), Password: types.StringValue(password), CA: types.StringNull(), Insecure: types.BoolValue(false), Timeout: types.Int64Null()})
	if d.HasError() {
		t.Fatal(d)
	}
	raw, e := object.ToTerraformValue(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	return p, frameworkprovider.ConfigureRequest{Config: tfsdk.Config{Schema: s.Schema, Raw: raw}}
}
func TestConcurrentAliasesKeepVersionAndCredentialsLocal(t *testing.T) {
	versions := []string{"7.24.5 (stable)", "7.25beta5 (testing)"}
	var wg sync.WaitGroup
	for i, version := range versions {
		password := []string{"alias-one", "alias-two"}[i]
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u, p, ok := r.BasicAuth()
			if !ok || u != "admin" || p != password {
				t.Error("credentials crossed aliases")
			}
			if r.URL.Path == "/rest/system/package" {
				json.NewEncoder(w).Encode([]map[string]string{{"name": "routeros", "version": strings.Split(version, " ")[0]}})
				return
			}
			json.NewEncoder(w).Encode(map[string]string{"version": version, "password": "********"})
		}))
		defer srv.Close()
		p, q := aliasConfig(t, srv.URL, password)
		wg.Add(1)
		go func() {
			defer wg.Done()
			var s frameworkprovider.ConfigureResponse
			p.Configure(context.Background(), q, &s)
			if s.Diagnostics.HasError() {
				t.Error(s.Diagnostics)
				return
			}
			c := s.ResourceData.(*client.Client)
			defer c.Close()
			if len(c.Packages) != 1 || c.Packages[0] != "routeros" {
				t.Error("missing alias package capabilities")
			}
			if c.Version != version {
				t.Error("global version contamination")
			}
			for j := 0; j < 10; j++ {
				if e := c.Request(context.Background(), "GET", "system/resource", "", nil, nil, nil); e != nil {
					t.Error(e)
				}
			}
		}()
	}
	wg.Wait()
}
func TestUnsupportedVersionDoesNotShareClient(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{"version":"6.49.1"}`)) }))
	defer srv.Close()
	p, q := aliasConfig(t, srv.URL, "secret")
	var s frameworkprovider.ConfigureResponse
	p.Configure(context.Background(), q, &s)
	if !s.Diagnostics.HasError() || s.ResourceData != nil || s.DataSourceData != nil {
		t.Fatal("unsupported version was shared")
	}
}
func TestRefreshFailureDoesNotMutateState(t *testing.T) {
	for _, body := range []string{`[{".id":"*2","address":"bad","interface":"ether1"}]`, `[{".id":"*2","address":"192.0.2.2/24","interface":"ether1","slave":"bad"}]`, `[{".id":"*2","address":{},"interface":"ether1"}]`, `null`, ``} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(body)) }))
		c, _ := client.New(client.Config{HostURL: srv.URL, Timeout: time.Second})
		r := ipAddressResource{client: c}
		m := generated.IpAddressModel{Id: types.StringValue("*2"), Address: types.StringValue("192.0.2.1/24")}
		old := m
		found, e := r.refresh(context.Background(), &m)
		if e == nil || found || m != old {
			t.Error("malformed response changed successful state")
		}
		c.Close()
		srv.Close()
	}
}
func TestImportIDsAndIdempotentDelete(t *testing.T) {
	r := ipAddressResource{}
	var schemaResponse resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &schemaResponse)
	for _, id := range []string{"", "ether1", "../system/reset", "*2?x"} {
		var s resource.ImportStateResponse
		r.ImportState(context.Background(), resource.ImportStateRequest{ID: id}, &s)
		if !s.Diagnostics.HasError() {
			t.Fatal("unsafe import accepted")
		}
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) {
		if q.Method != http.MethodDelete || q.URL.EscapedPath() != "/rest/ip/address/*2" {
			t.Error("wrong delete binding")
		}
		w.WriteHeader(404)
	}))
	defer srv.Close()
	c, _ := client.New(client.Config{HostURL: srv.URL, Timeout: time.Second})
	defer c.Close()
	r.client = c
	state := tfsdk.State{Schema: schemaResponse.Schema}
	d := state.Set(context.Background(), generated.IpAddressModel{Id: types.StringValue("*2")})
	if d.HasError() {
		t.Fatal(d)
	}
	var s resource.DeleteResponse
	r.Delete(context.Background(), resource.DeleteRequest{State: state}, &s)
	if s.Diagnostics.HasError() {
		t.Fatal("typed 404 delete was not idempotent")
	}
}
func TestPayloadEmptyNullUnknownAndComputed(t *testing.T) {
	m := generated.IpAddressModel{Address: types.StringValue("2001:db8::1"), Interface: types.StringValue("tf-test"), Comment: types.StringValue(""), Network: types.StringUnknown(), Disabled: types.BoolNull(), Vrf: types.StringValue("computed-secret")}
	p := payload(m)
	if p["address"] != "2001:db8::1/128" || p["comment"] != "" {
		t.Fatal("empty/IPv6 representation lost")
	}
	for _, key := range []string{"network", "disabled", "vrf", ".id"} {
		if _, ok := p[key]; ok {
			t.Fatal("null/unknown/computed value written")
		}
	}
}
