// SPDX-License-Identifier: MPL-2.0
package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	frameworkprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	frameworkresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/matejaputic/terraform-provider-routeros/internal/catalog"
)

// A test-only provider is deliberate: no singleton is publicly registered until
// reviewed field contracts and both official generators have been integrated.
type singletonTestProvider struct {
	RouterOSProvider
	policy         catalog.Collection
	resourceSchema schema.Schema
}

var _ frameworkprovider.Provider = (*singletonTestProvider)(nil)

func (p *singletonTestProvider) Resources(context.Context) []func() frameworkresource.Resource {
	return []func() frameworkresource.Resource{func() frameworkresource.Resource { return newSingleton(p.policy, p.resourceSchema) }}
}
func TestSingletonTerraformMockLifecycle(t *testing.T) {
	fixture := singletonFixture(t, nil)
	var mu sync.Mutex
	observed := map[string]string{"name": "original", "note": "", "enabled": "no", "port": "3799", "status": "idle"}
	writes := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if q.URL.Path == "/rest/system/resource" {
			w.Write([]byte(`{"version":"7.24.5 (stable)"}`))
			return
		}
		if q.URL.Path == "/rest/system/package" {
			w.Write([]byte(`[{"name":"routeros","version":"7.24.5"}]`))
			return
		}
		if q.Method == "GET" && q.URL.Path == "/rest/fixture/settings" {
			json.NewEncoder(w).Encode(observed)
			return
		}
		if q.Method == "POST" && q.URL.Path == "/rest/fixture/settings/set" {
			var body map[string]string
			if e := json.NewDecoder(q.Body).Decode(&body); e != nil {
				t.Error(e)
				w.WriteHeader(400)
				return
			}
			for k, v := range body {
				if !oneOf(k, "name", "note", "enabled", "port") {
					t.Error("unexpected singleton write field", k)
					w.WriteHeader(400)
					return
				}
				observed[k] = v
			}
			writes++
			w.Write([]byte(`[]`))
			return
		}
		t.Errorf("unexpected operation %s %s", q.Method, q.URL.Path)
		w.WriteHeader(405)
	}))
	defer server.Close()
	factory := func() (tfprotov6.ProviderServer, error) {
		p := &singletonTestProvider{RouterOSProvider: RouterOSProvider{version: "test"}, policy: fixture.core.policy, resourceSchema: fixture.core.schema}
		return providerserver.NewProtocol6WithError(p)()
	}
	configuration := func(name string, enabled bool, note string) string {
		return fmt.Sprintf(`provider "routeros" {
 hosturl = %q
 username = "fixture"
 password = "fixture"
}
resource "routeros_singleton_fixture" "test" {
 name = %q
 enabled = %t
 port = 3799
 note = %q
}`, server.URL, name, enabled, note)
	}
	first := configuration("created", true, "line one\nline two")
	second := configuration("updated", false, "")
	resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){"routeros": factory}, Steps: []resource.TestStep{
		{Config: first, Check: resource.ComposeTestCheckFunc(resource.TestCheckResourceAttr("routeros_singleton_fixture.test", "id", "fixture.settings"), resource.TestCheckResourceAttr("routeros_singleton_fixture.test", "enabled", "true"))},
		{Config: first, PlanOnly: true},
		{ResourceName: "routeros_singleton_fixture.test", ImportState: true, ImportStateId: "fixture.settings", ImportStateVerify: true},
		{Config: second, Check: resource.TestCheckResourceAttr("routeros_singleton_fixture.test", "name", "updated")},
		{Config: second, PlanOnly: true},
		{PreConfig: func() { mu.Lock(); defer mu.Unlock(); observed["name"] = "external" }, Config: second, Check: resource.TestCheckResourceAttr("routeros_singleton_fixture.test", "name", "updated")},
	}})
	mu.Lock()
	defer mu.Unlock()
	if writes != 3 || observed["name"] != "updated" || observed["enabled"] != "no" || observed["note"] != "" {
		t.Fatal("singleton apply/import/drift/destroy semantics not preserved", writes, observed)
	}
}
