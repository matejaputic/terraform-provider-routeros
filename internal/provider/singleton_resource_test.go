// SPDX-License-Identifier: MPL-2.0
package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	frameworkresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/matejaputic/terraform-provider-routeros/internal/catalog"
	"github.com/matejaputic/terraform-provider-routeros/internal/client"
)

// Test-only schemas exercise maintained lifecycle mechanics. They are NOT
// provider registrations, officially generated schemas or resource completion.
func singletonFixture(t *testing.T, handler http.HandlerFunc) *singletonResource {
	t.Helper()
	p := catalog.Collection{Name: "singleton_fixture", Path: "/fixture/settings", Fields: []catalog.Field{
		{Name: "id", Wire: ".id", Type: "string", Mode: "computed"},
		{Name: "name", Wire: "name", Type: "string", Mode: "required"},
		{Name: "note", Wire: "note", Type: "string", Mode: "computed_optional"},
		{Name: "enabled", Wire: "enabled", Type: "boolean", Mode: "computed_optional"},
		{Name: "port", Wire: "port", Type: "integer", Mode: "computed_optional"},
		{Name: "status", Wire: "status", Type: "string", Mode: "computed"},
	}}
	s := schema.Schema{Attributes: map[string]schema.Attribute{
		"id": schema.StringAttribute{Computed: true}, "name": schema.StringAttribute{Required: true},
		"note": schema.StringAttribute{Optional: true, Computed: true}, "enabled": schema.BoolAttribute{Optional: true, Computed: true},
		"port": schema.Int64Attribute{Optional: true, Computed: true}, "status": schema.StringAttribute{Computed: true},
	}}
	r := newSingleton(p, s).(*singletonResource)
	if handler != nil {
		server := httptest.NewServer(handler)
		t.Cleanup(server.Close)
		c, e := client.New(client.Config{HostURL: server.URL, Timeout: time.Second})
		if e != nil {
			t.Fatal(e)
		}
		t.Cleanup(c.Close)
		r.core.client = c
	}
	return r
}
func singletonValues(r *singletonResource) map[string]attr.Value {
	return map[string]attr.Value{"id": types.StringValue(r.identity()), "name": types.StringValue("original"), "note": types.StringNull(), "enabled": types.BoolValue(false), "port": types.Int64Value(3799), "status": types.StringValue("idle")}
}
func singletonSDKState(t *testing.T, r *singletonResource, v map[string]attr.Value) tfsdk.State {
	t.Helper()
	raw, e := r.core.object(v).ToTerraformValue(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	return tfsdk.State{Schema: r.core.schema, Raw: raw}
}
func TestSingletonLifecycleAndUnmanage(t *testing.T) {
	observed := map[string]any{"name": "original", "note": "", "enabled": "no", "port": "3799", "status": "idle"}
	var calls []string
	r := singletonFixture(t, func(w http.ResponseWriter, q *http.Request) {
		calls = append(calls, q.Method+" "+q.URL.Path)
		switch {
		case q.Method == http.MethodGet && q.URL.Path == "/rest/fixture/settings":
			json.NewEncoder(w).Encode(observed)
		case q.Method == http.MethodPost && q.URL.Path == "/rest/fixture/settings/set":
			var b map[string]any
			if e := json.NewDecoder(q.Body).Decode(&b); e != nil {
				t.Error(e)
				w.WriteHeader(400)
				return
			}
			if _, exists := b[".id"]; exists {
				t.Error("ID leaked into /set")
			}
			if _, exists := b["status"]; exists {
				t.Error("computed field mutated")
			}
			for k, v := range b {
				observed[k] = v
			}
			w.Write([]byte(`[]`))
		default:
			t.Errorf("unreviewed operation %s %s", q.Method, q.URL.Path)
			w.WriteHeader(500)
		}
	})
	v := singletonValues(r)
	v["name"] = types.StringValue("created")
	v["note"] = types.StringValue("line one\nline two")
	v["enabled"] = types.BoolValue(true)
	sdk := singletonSDKState(t, r, v)
	create := frameworkresource.CreateResponse{State: tfsdk.State{Schema: r.core.schema}}
	r.Create(context.Background(), frameworkresource.CreateRequest{Config: tfsdk.Config{Schema: r.core.schema, Raw: sdk.Raw}, Plan: tfsdk.Plan{Schema: r.core.schema, Raw: sdk.Raw}}, &create)
	if create.Diagnostics.HasError() {
		t.Fatal(create.Diagnostics)
	}
	if !reflect.DeepEqual(calls, []string{"GET /rest/fixture/settings", "POST /rest/fixture/settings/set", "GET /rest/fixture/settings"}) {
		t.Fatal(calls)
	}
	if observed["enabled"] != "yes" || observed["port"] != "3799" || observed["note"] != "line one\nline two" {
		t.Fatal("invalid singleton encoding")
	}
	v["name"] = types.StringValue("updated")
	v["note"] = types.StringValue("")
	v["enabled"] = types.BoolValue(false)
	sdk = singletonSDKState(t, r, v)
	update := frameworkresource.UpdateResponse{State: create.State}
	r.Update(context.Background(), frameworkresource.UpdateRequest{State: create.State, Config: tfsdk.Config{Schema: r.core.schema, Raw: sdk.Raw}, Plan: tfsdk.Plan{Schema: r.core.schema, Raw: sdk.Raw}}, &update)
	if update.Diagnostics.HasError() || observed["name"] != "updated" || observed["note"] != "" || observed["enabled"] != "no" {
		t.Fatal(update.Diagnostics)
	}
	before := len(calls)
	r.core.client.Close()
	r.core.client = nil
	deletion := frameworkresource.DeleteResponse{State: update.State}
	r.Delete(context.Background(), frameworkresource.DeleteRequest{State: update.State}, &deletion)
	if deletion.Diagnostics.HasError() || len(calls) != before || !deletion.State.Raw.IsNull() || len(deletion.Diagnostics) != 1 {
		t.Fatal("destroy must only unmanage, including offline")
	}
}
func TestSingletonMalformedRefreshIsAtomic(t *testing.T) {
	for _, body := range []string{`{}`, `null`, `[]`, `{"name":"original","enabled":"maybe","port":"3799"}`, `{"name":"original","enabled":"no","port":"3.5"}`, `{"name":"original","enabled":"no"}`, `{"name":42,"enabled":"no","port":"3799"}`} {
		t.Run(body, func(t *testing.T) {
			r := singletonFixture(t, func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, body) })
			v := singletonValues(r)
			sdk := singletonSDKState(t, r, v)
			read := frameworkresource.ReadResponse{State: sdk}
			r.Read(context.Background(), frameworkresource.ReadRequest{State: sdk}, &read)
			if !read.Diagnostics.HasError() || !read.State.Raw.Equal(sdk.Raw) {
				t.Fatal("malformed refresh changed state", read.Diagnostics)
			}
		})
	}
}
func TestSingletonIdentityAndUnavailableRead(t *testing.T) {
	for _, id := range []types.String{types.StringValue("*A"), types.StringValue("other.settings"), types.StringNull(), types.StringUnknown()} {
		requests := 0
		r := singletonFixture(t, func(w http.ResponseWriter, _ *http.Request) { requests++; w.WriteHeader(404) })
		v := singletonValues(r)
		v["id"] = id
		if next, e := r.refresh(context.Background(), v); e == nil || next != nil || requests != 0 {
			t.Fatal("invalid ID reached network")
		}
	}
	for _, code := range []int{401, 403, 404, 500} {
		r := singletonFixture(t, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(code) })
		v := singletonValues(r)
		sdk := singletonSDKState(t, r, v)
		read := frameworkresource.ReadResponse{State: sdk}
		r.Read(context.Background(), frameworkresource.ReadRequest{State: sdk}, &read)
		if !read.Diagnostics.HasError() || !read.State.Raw.Equal(sdk.Raw) {
			t.Fatal("unavailable singleton treated as deleted")
		}
	}
}
func TestSingletonFailedMutationRetainsRecoveryState(t *testing.T) {
	for _, failure := range []string{"write", "refresh"} {
		wrote := false
		r := singletonFixture(t, func(w http.ResponseWriter, q *http.Request) {
			if q.Method == http.MethodPost {
				wrote = true
				if failure == "write" {
					w.WriteHeader(500)
				}
				return
			}
			if wrote && failure == "refresh" {
				w.Write([]byte(`{}`))
				return
			}
			w.Write([]byte(`{"name":"original","enabled":"no","port":"3799"}`))
		})
		v := singletonValues(r)
		v["name"] = types.StringValue("desired")
		sdk := singletonSDKState(t, r, v)
		create := frameworkresource.CreateResponse{State: tfsdk.State{Schema: r.core.schema}}
		r.Create(context.Background(), frameworkresource.CreateRequest{Config: tfsdk.Config{Schema: r.core.schema, Raw: sdk.Raw}, Plan: tfsdk.Plan{Schema: r.core.schema, Raw: sdk.Raw}}, &create)
		if !create.Diagnostics.HasError() || create.State.Raw.IsNull() {
			t.Fatal("failed create lost singleton recovery state")
		}
		var retained types.Object
		d := create.State.Get(context.Background(), &retained)
		if d.HasError() || retained.Attributes()["id"] != types.StringValue(r.identity()) || retained.Attributes()["name"] != types.StringValue("original") {
			t.Fatal("invalid recovery baseline")
		}
	}
}
func TestSingletonPayloadOmissionAndValidation(t *testing.T) {
	r := singletonFixture(t, nil)
	v := singletonValues(r)
	body, e := r.payload(v)
	if e != nil || len(body) != 3 || body["enabled"] != "no" {
		t.Fatal(body, e)
	}
	v["enabled"] = types.BoolNull()
	body, e = r.payload(v)
	if e != nil || len(body) != 2 {
		t.Fatal("omission must not write a default")
	}
	v["name"] = types.StringUnknown()
	if _, e = r.payload(v); e == nil {
		t.Fatal("unknown mutation accepted")
	}
	v["name"] = types.StringValue("")
	if _, e = r.payload(v); e == nil {
		t.Fatal("empty required field accepted")
	}
	v["name"] = types.StringValue("x\n")
	if _, e = r.payload(v); e == nil {
		t.Fatal("unsafe name accepted")
	}
	v["name"] = types.StringValue("x")
	v["note"] = types.StringValue("\x00")
	if _, e = r.payload(v); e == nil {
		t.Fatal("NUL accepted")
	}
	r.core.policy.Fields[1].ForceNew = true
	v["note"] = types.StringNull()
	if _, e = r.payload(v); e == nil {
		t.Fatal("collection replacement policy accepted")
	}
}
func TestSingletonImportIdentity(t *testing.T) {
	r := singletonFixture(t, nil)
	for _, id := range []string{"*A", "other.settings", "", r.identity()} {
		initial := singletonValues(r)
		for _, f := range r.core.policy.Fields {
			initial[f.Name] = nullField(f)
		}
		out := frameworkresource.ImportStateResponse{State: singletonSDKState(t, r, initial)}
		r.ImportState(context.Background(), frameworkresource.ImportStateRequest{ID: id}, &out)
		if out.Diagnostics.HasError() == (id == r.identity()) {
			t.Fatal("wrong import identity outcome", id, out.Diagnostics)
		}
	}
}
