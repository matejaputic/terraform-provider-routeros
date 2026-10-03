// SPDX-License-Identifier: MPL-2.0
package provider

import (
	"context"
	"encoding/hex"
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
	"github.com/matejaputic/terraform-provider-routeros/internal/client"
)

var dhcpOptionNames = []string{"ip_dhcp_client_option", "ip_dhcp_server_option"}

func dhcpOptionResource(t *testing.T, name string) *collectionResource {
	t.Helper()
	for _, constructor := range collectionConstructors() {
		r := constructor().(*collectionResource)
		if r.policy.Name == name {
			return r
		}
	}
	t.Fatalf("missing reviewed resource %s", name)
	return nil
}
func dhcpOptionValues(r *collectionResource) map[string]attr.Value {
	values := map[string]attr.Value{}
	for _, f := range r.policy.Fields {
		values[f.Name] = nullField(f)
	}
	values["id"] = types.StringValue("*A")
	values["name"] = types.StringValue("tf-coverage-option")
	values["code"] = types.Int64Value(77)
	values["value"] = types.StringValue("0x01")
	return values
}
func TestDHCPOptionValidationAndPayload(t *testing.T) {
	for _, name := range dhcpOptionNames {
		r := dhcpOptionResource(t, name)
		values := dhcpOptionValues(r)
		values["raw_value"] = types.StringValue("01")
		payload, err := r.payload(context.Background(), values)
		if err != nil || payload["code"] != "77" || payload["value"] != "0x01" || len(payload) != 3 {
			t.Fatalf("invalid option payload: %v %v", payload, err)
		}
		bad := map[string][]attr.Value{
			"code":  {types.Int64Value(0), types.Int64Value(255), types.Int64Value(-1), types.Int64Null(), types.Int64Unknown()},
			"value": {types.StringValue(""), types.StringValue(" \t"), types.StringValue("0x01\n"), types.StringNull(), types.StringUnknown()},
			"name":  {types.StringValue(""), types.StringNull(), types.StringUnknown()},
		}
		for field, entries := range bad {
			old := values[field]
			for _, entry := range entries {
				values[field] = entry
				if _, err := r.payload(context.Background(), values); err == nil {
					t.Errorf("accepted invalid %s.%s: %v", name, field, entry)
				}
			}
			values[field] = old
		}
		for _, code := range []int64{1, 254} {
			values["code"] = types.Int64Value(code)
			if err := r.validate(context.Background(), values, true); err != nil {
				t.Fatal(err)
			}
		}
	}
}
func TestDHCPClientDefaultOwnershipBeforeMutation(t *testing.T) {
	for _, flag := range []string{`"true"`, `true`, `"invalid"`, `null`} {
		t.Run(flag, func(t *testing.T) {
			writes := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) {
				if q.Method != "GET" {
					writes++
					w.WriteHeader(500)
					return
				}
				fmt.Fprintf(w, `[{".id":"*A","default":%s}]`, flag)
			}))
			defer server.Close()
			c, err := client.New(client.Config{HostURL: server.URL, Timeout: time.Second})
			if err != nil {
				t.Fatal(err)
			}
			defer c.Close()
			r := dhcpOptionResource(t, "ip_dhcp_client_option")
			r.client = c
			values := dhcpOptionValues(r)
			if next, found, err := r.refresh(context.Background(), values); err == nil || found || next != nil {
				t.Fatal("default or malformed ownership accepted")
			}
			raw, err := r.object(values).ToTerraformValue(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			state := tfsdk.State{Schema: r.schema, Raw: raw}
			update := frameworkresource.UpdateResponse{State: state}
			r.Update(context.Background(), frameworkresource.UpdateRequest{State: state, Plan: tfsdk.Plan{Schema: r.schema, Raw: raw}, Config: tfsdk.Config{Schema: r.schema, Raw: raw}}, &update)
			deletion := frameworkresource.DeleteResponse{State: state}
			r.Delete(context.Background(), frameworkresource.DeleteRequest{State: state}, &deletion)
			if !update.Diagnostics.HasError() || !deletion.Diagnostics.HasError() || writes != 0 {
				t.Fatal("unowned option mutation authorized")
			}
		})
	}
}
func TestDHCPOptionAtomicRefreshAndDefaults(t *testing.T) {
	for _, name := range dhcpOptionNames {
		for _, body := range []string{
			`[{".id":"*A","name":"tf-coverage-option","code":"bad","value":"0x01","raw-value":"01"}]`,
			`[{".id":"*A","name":"tf-coverage-option","code":"77","raw-value":"01"}]`,
			`[{".id":"*A","name":"tf-coverage-option","code":"77","value":"0x01","raw-value":12}]`,
			`[{".id":"*A","name":"tf-coverage-option","code":"255","value":"0x01","raw-value":"01"}]`,
		} {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) { w.Write([]byte(body)) }))
			c, err := client.New(client.Config{HostURL: server.URL, Timeout: time.Second})
			if err != nil {
				t.Fatal(err)
			}
			r := dhcpOptionResource(t, name)
			r.client = c
			values := dhcpOptionValues(r)
			next, found, err := r.refresh(context.Background(), values)
			if err == nil || found || next != nil || values["code"].(types.Int64).ValueInt64() != 77 {
				t.Fatal("invalid refresh changed state")
			}
			c.Close()
			server.Close()
		}
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) {
		w.Write([]byte(`[{".id":"*A","name":"tf-coverage-option","code":"77","value":"0x01","raw-value":"01"}]`))
	}))
	defer server.Close()
	c, err := client.New(client.Config{HostURL: server.URL, Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	r := dhcpOptionResource(t, "ip_dhcp_server_option")
	r.client = c
	values := dhcpOptionValues(r)
	values["force"] = types.BoolValue(false)
	values["comment"] = types.StringValue("")
	next, found, err := r.refresh(context.Background(), values)
	if err != nil || !found || next["force"].(types.Bool).ValueBool() || next["comment"].(types.String).ValueString() != "" {
		t.Fatal("reviewed omission defaults not preserved", err)
	}
}

func TestTerraformDHCPOptionsMockLifecycle(t *testing.T) {
	var mu sync.Mutex
	rows := map[string]map[string]map[string]string{}
	counter := 0
	mutations := map[string]int{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if u, p, ok := q.BasicAuth(); !ok || u != "admin" || p != "secret" {
			w.WriteHeader(401)
			return
		}
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
		if path != "/ip/dhcp-client/option" && path != "/ip/dhcp-server/option" {
			t.Errorf("unexpected path %s", path)
			w.WriteHeader(404)
			return
		}
		if rows[path] == nil {
			rows[path] = map[string]map[string]string{}
		}
		if q.Method == "GET" {
			result := []map[string]string{}
			for key, row := range rows[path] {
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
		mutations[path+q.Method]++
		if q.Method == "DELETE" {
			delete(rows[path], id)
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
			if !oneOf(key, "name", "code", "value", "comment", "force") {
				t.Errorf("unreviewed/computed mutation field %s", key)
			}
		}
		if q.Method == "PATCH" {
			if _, ok := body["name"]; ok {
				t.Error("replacement name echoed in PATCH")
			}
			if rows[path][id] == nil {
				w.WriteHeader(404)
				return
			}
		} else if q.Method == "PUT" {
			counter++
			id = fmt.Sprintf("*%X", counter)
			rows[path][id] = map[string]string{".id": id}
		} else {
			w.WriteHeader(405)
			return
		}
		row := rows[path][id]
		for key, value := range body {
			row[key] = value
		}
		if expression := row["value"]; strings.HasPrefix(expression, "0x") {
			row["raw-value"] = strings.TrimPrefix(expression, "0x")
		} else if strings.HasPrefix(expression, "s'") && strings.HasSuffix(expression, "'") {
			row["raw-value"] = hex.EncodeToString([]byte(expression[2 : len(expression)-1]))
		} else {
			t.Error("unexpected mock expression")
		}
		if row["force"] == "no" {
			delete(row, "force")
		}
		if row["force"] == "yes" {
			row["force"] = "true"
		}
		if row["comment"] == "" {
			delete(row, "comment")
		}
		json.NewEncoder(w).Encode(row)
	}))
	defer server.Close()
	c, err := client.New(client.Config{HostURL: server.URL, Username: "admin", Password: "secret", Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	provider := fmt.Sprintf(`provider "routeros" {
 hosturl = %q
 username = "admin"
 password = "secret"
}`, server.URL)
	resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){"routeros": providerserver.NewProtocol6WithError(New("test")())}, Steps: dhcpOptionSteps(t, c, provider), CheckDestroy: dhcpOptionDestroy(c)})
	mu.Lock()
	defer mu.Unlock()
	for _, path := range []string{"/ip/dhcp-client/option", "/ip/dhcp-server/option"} {
		if len(rows[path]) != 0 || mutations[path+"PUT"] < 2 || mutations[path+"PATCH"] < 1 || mutations[path+"DELETE"] < 2 {
			t.Fatalf("incomplete mock lifecycle %s: %v", path, mutations)
		}
	}
}
