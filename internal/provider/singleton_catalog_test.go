// SPDX-License-Identifier: MPL-2.0
package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strconv"
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
	"github.com/matejaputic/terraform-provider-routeros/internal/catalog"
	"github.com/matejaputic/terraform-provider-routeros/internal/client"
)

func registeredSingleton(t *testing.T, name string) *singletonResource {
	t.Helper()
	for _, constructor := range singletonConstructors() {
		r := constructor().(*singletonResource)
		if r.core.policy.Name == name {
			return r
		}
	}
	t.Fatalf("missing singleton binding %s", name)
	return nil
}
func singletonBaseline(p catalog.Collection) map[string]any {
	row := map[string]any{}
	for _, f := range p.Fields {
		if f.Name == "id" {
			continue
		}
		switch f.Type {
		case "boolean":
			row[f.Wire] = "false"
		case "integer":
			v := int64(5)
			if f.Minimum != nil {
				v = *f.Minimum
			}
			row[f.Wire] = strconv.FormatInt(v, 10)
		case "string":
			v := "mock-baseline"
			if len(f.Choices) > 0 {
				v = f.Choices[0]
			}
			if singletonAddressList(p.Name, f.Name) {
				v = ""
			}
			row[f.Wire] = v
		}
	}
	return row
}
func registeredSingletonValues(t *testing.T, r *singletonResource, row map[string]any) map[string]attr.Value {
	t.Helper()
	values := map[string]attr.Value{"id": types.StringValue(r.identity())}
	for _, f := range r.core.policy.Fields {
		if f.Name == "id" {
			continue
		}
		v, e := decodeField(context.Background(), f, row[f.Wire])
		if e != nil {
			t.Fatal(e)
		}
		values[f.Name] = v
	}
	return values
}
func TestSingletonRegisteredContractAndNegatives(t *testing.T) {
	if len(catalog.Singletons()) != 19 || len(singletonCases) != 19 {
		t.Fatal("fixed nineteen singleton set changed")
	}
	for _, p := range catalog.Singletons() {
		t.Run(p.Name, func(t *testing.T) {
			r := registeredSingleton(t, p.Name)
			if len(r.core.schema.Attributes) != len(p.Fields) {
				t.Fatal("official schema field set differs")
			}
			row := singletonBaseline(p)
			values := registeredSingletonValues(t, r, row)
			for _, f := range p.Fields {
				a := r.core.schema.Attributes[f.Name]
				if a == nil || a.IsRequired() != (f.Mode == "required") || a.IsComputed() != (f.Mode != "required") || a.IsOptional() != (f.Mode == "computed_optional") || a.IsSensitive() {
					t.Fatalf("schema contract mismatch %s", f.Name)
				}
				if len(f.Choices) > 0 {
					old := values[f.Name]
					values[f.Name] = types.StringValue("unreviewed-enum")
					if r.validate(values, true) == nil {
						t.Fatal("invalid enum accepted")
					}
					values[f.Name] = old
				}
				if f.Minimum != nil {
					old := values[f.Name]
					values[f.Name] = types.Int64Value(*f.Minimum - 1)
					if r.validate(values, true) == nil {
						t.Fatal("below-minimum accepted")
					}
					values[f.Name] = old
				}
				if f.Maximum != nil {
					old := values[f.Name]
					values[f.Name] = types.Int64Value(*f.Maximum + 1)
					if r.validate(values, true) == nil {
						t.Fatal("above-maximum accepted")
					}
					values[f.Name] = old
				}
			}
			if err := r.validate(values, true); err != nil {
				t.Fatal(err)
			}
			original := values["id"]
			values["id"] = types.StringValue("*A")
			if _, err := r.refresh(context.Background(), values); err == nil {
				t.Fatal("item identity accepted")
			}
			values["id"] = original
			first := p.Fields[1]
			old := values[first.Name]
			values[first.Name] = nullField(first)
			switch first.Type {
			case "string":
				values[first.Name] = types.StringUnknown()
			case "boolean":
				values[first.Name] = types.BoolUnknown()
			case "integer":
				values[first.Name] = types.Int64Unknown()
			}
			if _, err := r.payload(values); err == nil {
				t.Fatal("unknown mutation accepted")
			}
			values[first.Name] = old
			var response frameworkresource.ImportStateResponse
			r.ImportState(context.Background(), frameworkresource.ImportStateRequest{ID: "*A"}, &response)
			if !response.Diagnostics.HasError() {
				t.Fatal("invalid import accepted")
			}
		})
	}
}
func TestSingletonRegisteredAtomicReadsAndRecovery(t *testing.T) {
	for _, p := range catalog.Singletons() {
		t.Run(p.Name, func(t *testing.T) {
			r := registeredSingleton(t, p.Name)
			row := singletonBaseline(p)
			values := registeredSingletonValues(t, r, row)
			ctx := context.Background()
			malformed, failedWrite := true, false
			writes := 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) {
				if q.Method == "GET" {
					if malformed {
						fmt.Fprint(w, `{}`)
					} else {
						json.NewEncoder(w).Encode(row)
					}
					return
				}
				if q.Method != "POST" || q.URL.Path != "/rest"+p.Path+"/set" {
					t.Errorf("unsupported settings mutation")
					w.WriteHeader(405)
					return
				}
				writes++
				if failedWrite {
					w.WriteHeader(503)
				} else {
					w.WriteHeader(200)
				}
			}))
			defer srv.Close()
			c, err := client.New(client.Config{HostURL: srv.URL, Timeout: time.Second})
			if err != nil {
				t.Fatal(err)
			}
			defer c.Close()
			r.core.client = c
			if next, err := r.refresh(ctx, values); err == nil || next != nil {
				t.Fatal("malformed global read committed")
			}
			raw, err := r.core.object(values).ToTerraformValue(ctx)
			if err != nil {
				t.Fatal(err)
			}
			state := tfsdk.State{Schema: r.core.schema, Raw: raw}
			read := frameworkresource.ReadResponse{State: state}
			r.Read(ctx, frameworkresource.ReadRequest{State: state}, &read)
			if !read.Diagnostics.HasError() || !read.State.Raw.Equal(raw) {
				t.Fatal("failed read changed state")
			}
			malformed = false
			failedWrite = true
			req := frameworkresource.CreateRequest{Config: tfsdk.Config{Schema: r.core.schema, Raw: raw}, Plan: tfsdk.Plan{Schema: r.core.schema, Raw: raw}}
			create := frameworkresource.CreateResponse{State: tfsdk.State{Schema: r.core.schema}}
			r.Create(ctx, req, &create)
			if !create.Diagnostics.HasError() {
				t.Fatal("failed settings write lost diagnostic")
			}
			var recovered types.Object
			if d := create.State.Get(ctx, &recovered); d.HasError() {
				t.Fatal(d)
			}
			if !recovered.Equal(r.core.object(values)) {
				t.Fatal("failed create lost baseline recovery state")
			}
			update := frameworkresource.UpdateResponse{State: state}
			r.Update(ctx, frameworkresource.UpdateRequest{State: state, Config: req.Config, Plan: req.Plan}, &update)
			if !update.Diagnostics.HasError() || !update.State.Raw.Equal(raw) {
				t.Fatal("failed update changed state")
			}
			r.core.client = nil
			deletion := frameworkresource.DeleteResponse{State: state}
			r.Delete(ctx, frameworkresource.DeleteRequest{State: state}, &deletion)
			if deletion.Diagnostics.HasError() || writes != 2 {
				t.Fatal("destroy performed a mutation")
			}
		})
	}
}

func singletonSteps(t *testing.T, c *client.Client, provider string) []resource.TestStep {
	t.Helper()
	first, changed := singletonConfig(provider, false), singletonConfig(provider, true)
	check := func(changed bool) resource.TestCheckFunc {
		checks := []resource.TestCheckFunc{}
		index := 0
		if changed {
			index = 1
		}
		for _, p := range catalog.Singletons() {
			address := "routeros_" + p.Name + ".test"
			checks = append(checks, resource.TestCheckResourceAttr(address, "id", strings.ReplaceAll(strings.TrimPrefix(p.Path, "/"), "/", ".")))
			keys := []string{}
			for field := range singletonCases[p.Name][index] {
				keys = append(keys, field)
			}
			sort.Strings(keys)
			for _, field := range keys {
				checks = append(checks, resource.TestCheckResourceAttr(address, field, fmt.Sprint(singletonCases[p.Name][index][field])))
			}
		}
		return resource.ComposeTestCheckFunc(checks...)
	}
	steps := []resource.TestStep{{Config: first, Check: check(false)}, {Config: first, PlanOnly: true}, {Config: changed, Check: check(true)}, {Config: changed, PlanOnly: true}}
	for _, p := range catalog.Singletons() {
		steps = append(steps, resource.TestStep{ResourceName: "routeros_" + p.Name + ".test", ImportState: true, ImportStateId: strings.ReplaceAll(strings.TrimPrefix(p.Path, "/"), "/", "."), ImportStateVerify: true})
	}
	steps = append(steps, resource.TestStep{Config: changed, PreConfig: func() {
		for _, p := range catalog.Singletons() {
			body := map[string]any{}
			for field, value := range singletonCases[p.Name][0] {
				body[strings.ReplaceAll(field, "_", "-")] = fmt.Sprint(value)
			}
			if err := c.Request(context.Background(), "POST", p.Path+"/set", "", nil, body, nil); err != nil {
				t.Fatal(err)
			}
		}
	}, Check: check(true)}, resource.TestStep{Config: changed, PlanOnly: true}, resource.TestStep{Config: first, Check: check(false)}, resource.TestStep{Config: first, PlanOnly: true})
	return steps
}
func singletonDestroyRetains(c *client.Client) resource.TestCheckFunc {
	return func(*terraform.State) error {
		for _, p := range catalog.Singletons() {
			var row map[string]any
			if err := c.Request(context.Background(), "GET", p.Path, "", nil, nil, &row); err != nil {
				return err
			}
			for field, expected := range singletonCases[p.Name][0] {
				value, exists := row[strings.ReplaceAll(field, "_", "-")]
				if !exists {
					return fmt.Errorf("retained settings field missing %s.%s", p.Name, field)
				}
				f := catalog.Field{Type: "string"}
				switch expected.(type) {
				case bool:
					f.Type = "boolean"
				case int:
					f.Type = "integer"
				}
				observed, err := decodeField(context.Background(), f, value)
				if err != nil {
					return err
				}
				actual := ""
				switch v := observed.(type) {
				case types.String:
					actual = v.ValueString()
				case types.Bool:
					actual = strconv.FormatBool(v.ValueBool())
				case types.Int64:
					actual = strconv.FormatInt(v.ValueInt64(), 10)
				}
				if fmt.Sprint(expected) != actual {
					return fmt.Errorf("destroy reset setting %s.%s", p.Name, field)
				}
			}
		}
		return nil
	}
}
func TestTerraformSingletonCatalogMockLifecycle(t *testing.T) {
	var mu sync.Mutex
	rows := map[string]map[string]any{}
	policies := map[string]catalog.Collection{}
	writes := map[string]int{}
	for _, p := range catalog.Singletons() {
		rows[p.Path] = singletonBaseline(p)
		policies[p.Path] = p
	}
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
		if q.Method == "GET" {
			if rows[path] == nil {
				w.WriteHeader(404)
				return
			}
			json.NewEncoder(w).Encode(rows[path])
			return
		}
		if q.Method != "POST" || !strings.HasSuffix(path, "/set") {
			t.Errorf("unexpected singleton method/path %s %s", q.Method, path)
			w.WriteHeader(405)
			return
		}
		path = strings.TrimSuffix(path, "/set")
		p, ok := policies[path]
		if !ok {
			w.WriteHeader(404)
			return
		}
		body := map[string]string{}
		if err := json.NewDecoder(q.Body).Decode(&body); err != nil {
			t.Error(err)
			w.WriteHeader(400)
			return
		}
		allowed := map[string]bool{}
		for _, f := range p.Fields {
			if f.Mode != "computed" {
				allowed[f.Wire] = true
			}
		}
		for key, value := range body {
			if !allowed[key] {
				t.Errorf("unreviewed singleton mutation %s", key)
			}
			rows[path][key] = value
		}
		writes[path]++
		w.WriteHeader(200)
	}))
	defer srv.Close()
	c, err := client.New(client.Config{HostURL: srv.URL, Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	provider := fmt.Sprintf(`provider "routeros" {
 hosturl = %q
 username = "mock-user"
 password = "mock-only"
}`, srv.URL)
	resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){"routeros": providerserver.NewProtocol6WithError(New("test")())}, Steps: singletonSteps(t, c, provider), CheckDestroy: singletonDestroyRetains(c)})
	mu.Lock()
	defer mu.Unlock()
	for _, p := range catalog.Singletons() {
		if writes[p.Path] < 5 {
			t.Fatalf("missing create/update/drift/clearing for %s", p.Name)
		}
	}
}
