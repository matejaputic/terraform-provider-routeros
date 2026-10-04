// SPDX-License-Identifier: MPL-2.0
package provider

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
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
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/matejaputic/terraform-provider-routeros/internal/catalog"
	"github.com/matejaputic/terraform-provider-routeros/internal/client"
	"golang.org/x/crypto/ssh"
)

type extendedResourceFixture struct{ secret, key, wgKey, fingerprint string }

func newExtendedResourceFixture(t *testing.T) extendedResourceFixture {
	t.Helper()
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal("ephemeral test key generation failed")
	}
	key, err := ssh.NewPublicKey(pub)
	if err != nil {
		t.Fatal("ephemeral test key encoding failed")
	}
	nonce := make([]byte, 24)
	if _, err := rand.Read(nonce); err != nil {
		t.Fatal("ephemeral fixture generation failed")
	}
	return extendedResourceFixture{secret: hex.EncodeToString(nonce), key: strings.TrimSpace(string(ssh.MarshalAuthorizedKey(key))), wgKey: base64.StdEncoding.EncodeToString(pub), fingerprint: ssh.FingerprintSHA256(key)}
}
func (fixture extendedResourceFixture) value(p catalog.Collection, f catalog.Field, phase int) any {
	index := phase
	if f.ForceNew && !p.ReplacementOnly {
		index = 0
	}
	if f.Name == "disabled" {
		return true
	}
	if f.Type == "boolean" {
		return index%2 == 0
	}
	if f.Type == "integer" {
		value := int64(10)
		if f.Minimum != nil {
			value = *f.Minimum
		}
		if index > 0 && (f.Maximum == nil || value < *f.Maximum) {
			value++
		}
		if f.Name == "prefix_length" || f.Name == "pool_prefix_length" {
			value = 64 + int64(index)
		}
		return value
	}
	if len(f.Choices) > 0 {
		return f.Choices[index%len(f.Choices)]
	}
	if f.Name == "name" {
		return fmt.Sprintf("tf-%s-%s-%d", p.Name, fixture.secret[:8], index)
	}
	if f.Name == "key" {
		return fixture.key
	}
	if f.Name == "public_key" {
		return fixture.wgKey
	}
	if f.Constraint == "url" {
		return fmt.Sprintf("https://invalid.invalid/%s/%d", fixture.secret, index)
	}
	if f.Sensitive && f.Name != "on_event" {
		return fixture.secret
	}
	if f.Name == "on_event" {
		return ":nothing"
	}
	if f.Name == "value" && strings.Contains(p.Name, "option") {
		return fmt.Sprintf("0x%02x", index+1)
	}
	switch f.Constraint {
	case "ip", "ip4":
		return fmt.Sprintf("192.0.2.%d", index+10)
	case "ip6", "ip6_or_prefix":
		return fmt.Sprintf("2001:db8::%d", index+10)
	case "address6":
		return fmt.Sprintf("2001:db8::%d/64", index+10)
	case "prefix6":
		return fmt.Sprintf("2001:db8:%d::/48", index+100)
	case "prefix", "prefix4":
		return fmt.Sprintf("198.51.%d.0/24", index+100)
	case "mac":
		return fmt.Sprintf("02:00:00:00:00:%02x", index+10)
	case "rate_pair":
		return fmt.Sprintf("%d/%d", (index+1)*1000000, (index+1)*1000000)
	case "priority_pair":
		return fmt.Sprintf("%d/%d", 8-index, 8-index)
	case "vlan_ids":
		return fmt.Sprintf("%d-%d", 100+index*100, 101+index*100)
	case "duration":
		return fmt.Sprintf("%dm", index+1)
	}
	if f.Constraint == "user_policy" {
		if index == 0 {
			return "read,ssh"
		}
		return "read"
	}
	if f.Codec == "csv-string-set" {
		if f.Name == "address_families" {
			return "ip"
		}
		return fmt.Sprintf("tf-entry-%d", index)
	}
	return fmt.Sprintf("tf-%s-%d", f.Name, index)
}
func extendedResource(t *testing.T, name string) *collectionResource {
	t.Helper()
	for _, constructor := range extendedCollectionConstructors() {
		r := constructor().(*collectionResource)
		if r.policy.Name == name {
			return r
		}
	}
	t.Fatalf("missing reviewed binding %s", name)
	return nil
}
func (fixture extendedResourceFixture) values(r *collectionResource, phase int) map[string]attr.Value {
	v := map[string]attr.Value{}
	for _, f := range r.policy.Fields {
		v[f.Name] = nullField(f)
		if f.Name == "id" {
			v[f.Name] = types.StringValue("*A")
			continue
		}
		if f.Mode == "computed" {
			continue
		}
		value := fixture.value(r.policy, f, phase)
		switch x := value.(type) {
		case string:
			v[f.Name] = types.StringValue(x)
		case bool:
			v[f.Name] = types.BoolValue(x)
		case int64:
			v[f.Name] = types.Int64Value(x)
		}
	}
	return v
}
func (fixture extendedResourceFixture) config(provider string, phase int) string {
	var b strings.Builder
	b.WriteString(provider)
	for _, p := range catalog.ExtendedCollections() {
		fmt.Fprintf(&b, "\nresource %q %q {\n", "routeros_"+p.Name, "test")
		for _, f := range p.Fields {
			if f.Mode == "computed" {
				continue
			}
			value := fixture.value(p, f, phase)
			encoded, _ := json.Marshal(value)
			fmt.Fprintf(&b, " %s = %s\n", f.Name, encoded)
		}
		b.WriteString("}\n")
	}
	return b.String()
}
func (fixture extendedResourceFixture) completeRow(p catalog.Collection, row map[string]any) {
	for _, f := range p.Fields {
		if f.Name == "id" || f.Mode != "computed" {
			continue
		}
		switch f.Type {
		case "boolean":
			row[f.Wire] = "false"
		case "integer":
			row[f.Wire] = "1"
		case "string":
			row[f.Wire] = "observed"
		}
	}
	if p.Name == "ipv6_route" {
		row["static"] = "true"
	}
	if p.Name == "system_user_sshkeys" {
		row["fingerprint"] = fixture.fingerprint
		row["key-type"] = "ed25519"
		row["bits"] = "256"
		delete(row, "key")
	}
	for _, f := range p.Fields {
		if f.Sensitive && f.PreserveSecretOnOmission {
			delete(row, f.Wire)
		}
	}
}
func TestExtendedResourceCollectionContractsAndAtomicFailures(t *testing.T) {
	fixture := newExtendedResourceFixture(t)
	if len(catalog.ExtendedCollections()) != 86 {
		t.Fatal("fixed collection set changed")
	}
	for _, p := range catalog.ExtendedCollections() {
		t.Run(p.Name, func(t *testing.T) {
			r := extendedResource(t, p.Name)
			v := fixture.values(r, 0)
			payload, err := r.payload(context.Background(), v)
			if err != nil {
				t.Fatal(err)
			}
			if len(r.schema.Attributes) != len(p.Fields) {
				t.Fatal("official field set differs")
			}
			for _, f := range p.Fields {
				if len(f.Choices) > 0 && validateReviewedField(f, types.StringValue("unsupported-enum")) == nil {
					t.Fatalf("unknown enum accepted: %s", f.Name)
				}
				if f.Minimum != nil && validateReviewedField(f, types.Int64Value(*f.Minimum-1)) == nil {
					t.Fatalf("below bound accepted: %s", f.Name)
				}
				if f.Maximum != nil && validateReviewedField(f, types.Int64Value(*f.Maximum+1)) == nil {
					t.Fatalf("above bound accepted: %s", f.Name)
				}
				a := r.schema.Attributes[f.Name]
				if a == nil || a.IsSensitive() != f.Sensitive || a.IsRequired() != (f.Mode == "required") || a.IsOptional() != (f.Mode == "computed_optional") || a.IsComputed() != (f.Mode != "required") {
					t.Fatalf("official flags differ: %s", f.Name)
				}
				if f.Mode == "computed" {
					if _, exists := payload[f.Wire]; exists {
						t.Fatal("computed mutation field")
					}
				}
				if f.Mode == "required" {
					old := v[f.Name]
					v[f.Name] = nullField(f)
					if _, err := r.payload(context.Background(), v); err == nil {
						t.Fatalf("required field accepted null: %s", f.Name)
					}
					v[f.Name] = old
				}
			}
			row := map[string]any{".id": "*A"}
			for key, value := range payload {
				row[key] = value
			}
			fixture.completeRow(p, row)
			malformed := false
			writes := 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) {
				if q.Method != "GET" {
					writes++
					w.WriteHeader(503)
					return
				}
				if malformed {
					fmt.Fprint(w, `[{".id":"*A","dynamic":"malformed"}]`)
					return
				}
				json.NewEncoder(w).Encode([]map[string]any{row})
			}))
			defer srv.Close()
			c, err := client.New(client.Config{HostURL: srv.URL, Username: "mock-user", Timeout: time.Second})
			if err != nil {
				t.Fatal(err)
			}
			defer c.Close()
			c.Packages = []string{"routeros", "container"}
			r.client = c
			if _, found, err := r.refresh(context.Background(), v); err != nil || !found {
				t.Fatal("positive refresh failed", err)
			}
			raw, err := r.object(v).ToTerraformValue(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			state := tfsdk.State{Schema: r.schema, Raw: raw}
			malformed = true
			read := frameworkresource.ReadResponse{State: state}
			r.Read(context.Background(), frameworkresource.ReadRequest{State: state}, &read)
			if !read.Diagnostics.HasError() || !read.State.Raw.Equal(raw) {
				t.Fatal("malformed read changed state")
			}
			deletion := frameworkresource.DeleteResponse{State: state}
			r.Delete(context.Background(), frameworkresource.DeleteRequest{State: state}, &deletion)
			if !deletion.Diagnostics.HasError() || writes != 0 {
				t.Fatal("mutation after malformed ownership")
			}
			malformed = false
			row["default"] = "true"
			if next, found, err := r.refresh(context.Background(), v); err == nil || found || next != nil {
				t.Fatal("default ownership accepted")
			}
			delete(row, "default")
			update := frameworkresource.UpdateResponse{State: state}
			r.Update(context.Background(), frameworkresource.UpdateRequest{State: state, Config: tfsdk.Config{Schema: r.schema, Raw: raw}, Plan: tfsdk.Plan{Schema: r.schema, Raw: raw}}, &update)
			if !update.Diagnostics.HasError() || !update.State.Raw.Equal(raw) {
				t.Fatal("failed update changed state")
			}
			expected := 1
			if p.ReplacementOnly {
				expected = 0
			}
			if writes != expected {
				t.Fatal("unsupported mutation method")
			}
			failedDelete := frameworkresource.DeleteResponse{State: state}
			r.Delete(context.Background(), frameworkresource.DeleteRequest{State: state}, &failedDelete)
			if !failedDelete.Diagnostics.HasError() || !failedDelete.State.Raw.Equal(raw) || writes != expected+1 {
				t.Fatal("failed delete lost state or diagnostic")
			}
		})
	}
}
func TestTerraformExtendedCollectionsMockLifecycle(t *testing.T) {
	fixture := newExtendedResourceFixture(t)
	var mu sync.Mutex
	rows := map[string]map[string]map[string]any{}
	policies := map[string]catalog.Collection{}
	counts := map[string]map[string]int{}
	counter := 0
	for _, p := range catalog.ExtendedCollections() {
		rows[p.Path] = map[string]map[string]any{}
		policies[p.Path] = p
		counts[p.Name] = map[string]int{}
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
			json.NewEncoder(w).Encode([]map[string]string{{"name": "routeros", "version": "7.24.5"}, {"name": "container", "version": "7.24.5"}})
			return
		}
		path := strings.TrimPrefix(q.URL.Path, "/rest")
		id := ""
		if i := strings.LastIndex(path, "/*"); i >= 0 {
			id = path[i+1:]
			path = path[:i]
		}
		p, ok := policies[path]
		if !ok {
			t.Errorf("unreviewed path %s", path)
			w.WriteHeader(404)
			return
		}
		if q.Method == "GET" {
			out := []map[string]any{}
			for key, row := range rows[path] {
				if wanted := q.URL.Query().Get(".id"); wanted != "" && wanted != key {
					continue
				}
				if wanted := q.URL.Query().Get("name"); wanted != "" && wanted != row["name"] {
					continue
				}
				out = append(out, row)
			}
			json.NewEncoder(w).Encode(out)
			return
		}
		counts[p.Name][q.Method]++
		if q.Method == "DELETE" {
			delete(rows[path], id)
			w.WriteHeader(204)
			return
		}
		body := map[string]string{}
		if err := json.NewDecoder(q.Body).Decode(&body); err != nil {
			t.Fatal("invalid mutation encoding")
		}
		allowed := map[string]catalog.Field{}
		for _, f := range p.Fields {
			if f.Mode != "computed" {
				allowed[f.Wire] = f
			}
		}
		for key := range body {
			f, exists := allowed[key]
			if !exists {
				t.Errorf("unreviewed mutation field %s.%s", p.Name, key)
			}
			if q.Method == "PATCH" && (p.ReplacementOnly || f.ForceNew) {
				t.Errorf("unsupported PATCH field %s.%s", p.Name, key)
			}
		}
		switch q.Method {
		case "PUT":
			counter++
			id = fmt.Sprintf("*%X", counter)
			rows[path][id] = map[string]any{".id": id}
		case "PATCH":
			if rows[path][id] == nil {
				w.WriteHeader(404)
				return
			}
		default:
			w.WriteHeader(405)
			return
		}
		for key, value := range body {
			rows[path][id][key] = value
		}
		fixture.completeRow(p, rows[path][id])
		json.NewEncoder(w).Encode(rows[path][id])
	}))
	defer srv.Close()
	provider := fmt.Sprintf(`provider "routeros" {
 hosturl = %q
 username = "mock-user"
 password = "mock-only"
}`, srv.URL)
	first, changed := fixture.config(provider, 0), fixture.config(provider, 1)
	checks := []resource.TestCheckFunc{}
	for _, p := range catalog.ExtendedCollections() {
		checks = append(checks, resource.TestCheckResourceAttrSet("routeros_"+p.Name+".test", "id"))
	}
	steps := []resource.TestStep{{Config: first, Check: resource.ComposeTestCheckFunc(checks...)}, {Config: first, PlanOnly: true}, {Config: changed}, {Config: changed, PlanOnly: true}}
	for _, p := range catalog.ExtendedCollections() {
		ignore := []string{}
		for _, f := range p.Fields {
			if f.PreserveSecretOnOmission {
				ignore = append(ignore, f.Name)
			}
		}
		steps = append(steps, resource.TestStep{ResourceName: "routeros_" + p.Name + ".test", ImportState: true, ImportStateVerify: true, ImportStateVerifyIgnore: ignore})
	}
	steps = append(steps, resource.TestStep{Config: changed, PreConfig: func() {
		mu.Lock()
		defer mu.Unlock()
		for _, p := range catalog.ExtendedCollections() {
			for id := range rows[p.Path] {
				delete(rows[p.Path], id)
			}
		}
	}}, resource.TestStep{Config: changed, PlanOnly: true})
	resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){"routeros": providerserver.NewProtocol6WithError(New("test")())}, Steps: steps, CheckDestroy: func(*terraform.State) error {
		mu.Lock()
		defer mu.Unlock()
		for _, p := range catalog.ExtendedCollections() {
			if len(rows[p.Path]) != 0 {
				return fmt.Errorf("owned object remains: %s", p.Name)
			}
		}
		return nil
	}})
	mu.Lock()
	defer mu.Unlock()
	for _, p := range catalog.ExtendedCollections() {
		if counts[p.Name]["PUT"] < 2 || counts[p.Name]["DELETE"] < 1 {
			t.Fatalf("missing lifecycle for %s", p.Name)
		}
		if p.ReplacementOnly {
			if counts[p.Name]["PATCH"] != 0 {
				t.Fatal("SSH keys used unsupported PATCH")
			}
		} else if counts[p.Name]["PATCH"] < 1 {
			t.Fatalf("missing update for %s", p.Name)
		}
	}
}
