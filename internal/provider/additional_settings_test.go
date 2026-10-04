// SPDX-License-Identifier: MPL-2.0
package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	fr "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/matejaputic/terraform-provider-routeros/internal/catalog"
	"github.com/matejaputic/terraform-provider-routeros/internal/client"
)

// Independent reviewed safe cases: no radio activation, external provisioning,
// payment request, mail send, server activation or management-listener mutation.
var additionalSettingsCases = map[string][2]map[string]any{
	"capsman_aaa":               {{"mac_mode": "as-username", "mac_format": "XX:XX:XX:XX:XX:XX"}, {"mac_mode": "as-username-and-password", "mac_format": "XX:XX:XX:XX:XX:XX"}},
	"capsman_manager":           {{"enabled": false, "require_peer_certificate": false, "upgrade_policy": "none"}, {"enabled": false, "require_peer_certificate": false, "upgrade_policy": "suggest-same-version"}},
	"container_config":          {{"registry_url": "https://registry.invalid/initial", "username": "tf-owned-registry", "password": "mock-secret-one"}, {"registry_url": "https://registry.invalid/updated", "username": "", "password": ""}},
	"disk_settings":             {{"auto_media_sharing": false, "auto_smb_sharing": false, "auto_media_interface": "none"}, {"auto_media_sharing": false, "auto_smb_sharing": false, "auto_media_interface": "tf-test"}},
	"ip_dns":                    {{"servers": "192.0.2.53,198.51.100.53", "allow_remote_requests": false, "cache_size": 4096}, {"servers": "", "allow_remote_requests": false, "cache_size": 8192}},
	"interface_detect_internet": {{"detect_interface_list": "none", "internet_interface_list": "none", "lan_interface_list": "none", "wan_interface_list": "none"}, {"detect_interface_list": "none", "internet_interface_list": "none", "lan_interface_list": "static", "wan_interface_list": "none"}},
	"interface_l2tp_server":     {{"enabled": false, "max_mtu": 1400, "max_mru": 1400, "authentication": "mschap2"}, {"enabled": false, "max_mtu": 1450, "max_mru": 1450, "authentication": "mschap2"}},
	"interface_sstp_server":     {{"enabled": false, "max_mtu": 1400, "max_mru": 1400, "authentication": "mschap2", "pfs": "no"}, {"enabled": false, "max_mtu": 1450, "max_mru": 1450, "authentication": "mschap2", "pfs": "required"}},
	"interface_wireless_cap":    {{"enabled": false, "caps_man_addresses": "192.0.2.1", "lock_to_caps_man": false}, {"enabled": false, "caps_man_addresses": "", "lock_to_caps_man": false}},
	"ip_cloud":                  {{"ddns_enabled": "auto", "update_time": false}, {"ddns_enabled": "auto", "update_time": true}},
	"ip_cloud_advanced":         {{"use_local_address": false}, {"use_local_address": true}},
	"ip_smb":                    {{"enabled": "no", "interfaces": "tf-test", "domain": "tf-owned-initial"}, {"enabled": "no", "interfaces": "tf-test", "domain": "tf-owned-updated"}},
	"ip_ssh_server":             {{"strong_crypto": true, "forwarding_enabled": "no"}, {"strong_crypto": false, "forwarding_enabled": "no"}},
	"ppp_aaa":                   {{"use_radius": false, "accounting": false}, {"use_radius": false, "accounting": true}},
	"snmp":                      {{"enabled": false, "contact": "tf-owned-contact", "location": "tf-owned-initial", "trap_version": 2}, {"enabled": false, "contact": "", "location": "tf-owned-updated", "trap_version": 2}},
	"system_clock":              {{"time_zone_name": "Etc/UTC", "time_zone_autodetect": false}, {"time_zone_name": "Europe/Belgrade", "time_zone_autodetect": false}},
	"system_led_settings":       {{"all_leds_off": "never"}, {"all_leds_off": "immediate"}},
	"system_user_settings":      {{"minimum_password_length": 8, "minimum_categories": 1}, {"minimum_password_length": 10, "minimum_categories": 1}},
	"tool_bandwidth_server":     {{"enabled": false, "authenticate": true, "max_sessions": 10}, {"enabled": false, "authenticate": true, "max_sessions": 20}},
	"tool_email":                {{"server": "192.0.2.25", "port": 25, "tls": "no", "user": "", "password": ""}, {"server": "198.51.100.25", "port": 2525, "tls": "no", "user": "", "password": ""}},
	"ip_upnp":                   {{"enabled": false, "allow_disable_external_interface": false, "show_dummy_rule": false}, {"enabled": false, "allow_disable_external_interface": false, "show_dummy_rule": true}},
	"system_user_aaa":           {{"use_radius": false, "accounting": false, "default_group": "read"}, {"use_radius": false, "accounting": true, "default_group": "read"}},
	"user_manager_advanced":     {{"paypal_allow": false, "paypal_use_sandbox": false}, {"paypal_allow": false, "paypal_use_sandbox": true}},
	"user_manager_database":     {{"db_path": "user-manager"}, {"db_path": "tf-owned-user-manager"}},
	"user_manager_settings":     {{"enabled": false, "use_profiles": false, "authentication_port": 1812, "accounting_port": 1813}, {"enabled": false, "use_profiles": false, "authentication_port": 1912, "accounting_port": 1913}},
	"wifi_cap":                  {{"enabled": false, "caps_man_addresses": "192.0.2.1", "slaves_static": false}, {"enabled": false, "caps_man_addresses": "", "slaves_static": true}},
	"wifi_capsman":              {{"enabled": false, "interfaces": "none", "require_peer_certificate": false}, {"enabled": false, "interfaces": "none", "require_peer_certificate": true}},
}

func registeredAdditionalSettings(t *testing.T, name string) *singletonResource {
	t.Helper()
	for _, constructor := range additionalSettingsConstructors() {
		r := constructor().(*singletonResource)
		if r.core.policy.Name == name {
			return r
		}
	}
	t.Fatalf("missing reviewed settings %s", name)
	return nil
}
func additionalSettingsBaseline(p catalog.Collection) map[string]any {
	row := singletonBaseline(p)
	for _, f := range p.Fields {
		if f.Codec == "csv-string-set" {
			row[f.Wire] = ""
		}
	}
	if p.Name == "system_clock" {
		row["time-zone-name"] = "Etc/UTC"
	}
	if p.Name == "snmp" {
		row["src-address"] = "0.0.0.0"
	}
	if p.Name == "interface_l2tp_server" || p.Name == "interface_sstp_server" {
		row["authentication"] = "mschap2"
	}
	for name, value := range additionalSettingsCases[p.Name][0] {
		row[strings.ReplaceAll(name, "_", "-")] = fmt.Sprint(value)
	}
	return row
}
func TestAdditionalSettingsContractsAtomicFailuresAndSecrets(t *testing.T) {
	if len(catalog.AdditionalSettings()) != len(additionalSettingsCases) {
		t.Fatal("settings evidence omits a reviewed constructor")
	}
	for _, p := range catalog.AdditionalSettings() {
		t.Run(p.Name, func(t *testing.T) {
			r := registeredAdditionalSettings(t, p.Name)
			values := registeredSingletonValues(t, r, additionalSettingsBaseline(p))
			if err := r.validate(values, true); err != nil {
				t.Fatal(err)
			}
			for _, f := range p.Fields {
				a := r.core.schema.Attributes[f.Name]
				if a == nil || a.IsSensitive() != f.Sensitive || a.IsComputed() != (f.Mode != "required") || a.IsOptional() != (f.Mode == "computed_optional") {
					t.Fatal("official schema contract mismatch")
				}
				old := values[f.Name]
				if len(f.Choices) > 0 {
					values[f.Name] = types.StringValue("unreviewed-enum")
					if r.validate(values, true) == nil {
						t.Fatal("invalid enum accepted")
					}
				}
				if f.Minimum != nil {
					values[f.Name] = types.Int64Value(*f.Minimum - 1)
					if r.validate(values, true) == nil {
						t.Fatal("below bound accepted")
					}
				}
				if f.Maximum != nil {
					values[f.Name] = types.Int64Value(*f.Maximum + 1)
					if r.validate(values, true) == nil {
						t.Fatal("above bound accepted")
					}
				}
				values[f.Name] = old
			}
			row := additionalSettingsBaseline(p)
			malformed, failedWrite := true, false
			writes := 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) {
				if q.URL.Path == "/rest/system/resource" {
					json.NewEncoder(w).Encode(map[string]string{"version": "7.24.5 (stable)"})
					return
				}
				if q.URL.Path == "/rest/system/package" {
					json.NewEncoder(w).Encode([]map[string]string{{"name": "routeros", "version": "7.24.5"}, {"name": "container", "version": "7.24.5"}, {"name": "wireless", "version": "7.24.5"}, {"name": "user-manager", "version": "7.24.5"}})
					return
				}
				if q.Method == "GET" {
					if malformed {
						fmt.Fprint(w, `{}`)
					} else {
						json.NewEncoder(w).Encode(row)
					}
					return
				}
				if q.Method != "POST" || q.URL.Path != "/rest"+p.Path+"/set" {
					t.Error("unsupported mutation method/path")
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
			c.Packages = []string{"routeros", "container", "wireless", "user-manager"}
			r.core.client = c
			raw, err := r.core.object(values).ToTerraformValue(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			state := tfsdk.State{Schema: r.core.schema, Raw: raw}
			read := fr.ReadResponse{State: state}
			r.Read(context.Background(), fr.ReadRequest{State: state}, &read)
			if !read.Diagnostics.HasError() || !read.State.Raw.Equal(raw) {
				t.Fatal("malformed read changed state")
			}
			malformed = false
			next, err := r.refresh(context.Background(), values)
			if err != nil {
				t.Fatal(err)
			}
			for _, f := range p.Fields {
				if !f.Sensitive {
					continue
				}
				original := row[f.Wire]
				delete(row, f.Wire)
				next, err = r.refresh(context.Background(), values)
				if err != nil || !next[f.Name].Equal(values[f.Name]) {
					t.Fatal("omitted secret not preserved")
				}
				row[f.Wire] = "**hidden**"
				next, err = r.refresh(context.Background(), values)
				if err != nil || !next[f.Name].Equal(values[f.Name]) {
					t.Fatal("masked secret not preserved")
				}
				row[f.Wire] = 42
				if next, err = r.refresh(context.Background(), values); err == nil || next != nil {
					t.Fatal("malformed secret read committed")
				}
				row[f.Wire] = original
			}
			failedWrite = true
			req := fr.CreateRequest{Config: tfsdk.Config{Schema: r.core.schema, Raw: raw}, Plan: tfsdk.Plan{Schema: r.core.schema, Raw: raw}}
			creation := fr.CreateResponse{State: tfsdk.State{Schema: r.core.schema}}
			r.Create(context.Background(), req, &creation)
			if !creation.Diagnostics.HasError() {
				t.Fatal("failed create missing diagnostic")
			}
			var recovered types.Object
			if d := creation.State.Get(context.Background(), &recovered); d.HasError() || !recovered.Equal(r.core.object(values)) {
				t.Fatal("failed create lost baseline recovery state")
			}
			update := fr.UpdateResponse{State: state}
			r.Update(context.Background(), fr.UpdateRequest{Config: req.Config, Plan: req.Plan, State: state}, &update)
			if !update.Diagnostics.HasError() || !update.State.Raw.Equal(raw) {
				t.Fatal("failed update lost prior state")
			}
			r.core.client = nil
			deletion := fr.DeleteResponse{State: state}
			r.Delete(context.Background(), fr.DeleteRequest{State: state}, &deletion)
			if deletion.Diagnostics.HasError() || writes != 2 {
				t.Fatal("unmanage destroy mutated settings")
			}
			var imported fr.ImportStateResponse
			r.ImportState(context.Background(), fr.ImportStateRequest{ID: "*A"}, &imported)
			if !imported.Diagnostics.HasError() {
				t.Fatal("collection ID accepted")
			}
		})
	}
}
func additionalSettingsConfig(policies []catalog.Collection, provider string, index int) string {
	var b strings.Builder
	b.WriteString(provider)
	for _, p := range policies {
		fmt.Fprintf(&b, "\nresource %q %q {\n", "routeros_"+p.Name, "test")
		if p.Name == "user_manager_settings" || p.Name == "user_manager_advanced" {
			b.WriteString(" depends_on = [routeros_user_manager_database.test]\n")
		}
		keys := []string{}
		for name := range additionalSettingsCases[p.Name][index] {
			keys = append(keys, name)
		}
		sort.Strings(keys)
		for _, name := range keys {
			encoded, _ := json.Marshal(additionalSettingsCases[p.Name][index][name])
			fmt.Fprintf(&b, " %s = %s\n", name, encoded)
		}
		b.WriteString("}\n")
	}
	return b.String()
}
func additionalSettingsSteps(t *testing.T, c *client.Client, policies []catalog.Collection, provider string) []resource.TestStep {
	t.Helper()
	first, changed := additionalSettingsConfig(policies, provider, 0), additionalSettingsConfig(policies, provider, 1)
	check := func(index int) resource.TestCheckFunc {
		checks := []resource.TestCheckFunc{}
		for _, p := range policies {
			address := "routeros_" + p.Name + ".test"
			checks = append(checks, resource.TestCheckResourceAttr(address, "id", strings.ReplaceAll(strings.TrimPrefix(p.Path, "/"), "/", ".")))
			for name, value := range additionalSettingsCases[p.Name][index] {
				checks = append(checks, resource.TestCheckResourceAttr(address, name, fmt.Sprint(value)))
			}
		}
		return resource.ComposeTestCheckFunc(checks...)
	}
	steps := []resource.TestStep{{Config: first, Check: check(0)}, {Config: first, PlanOnly: true}, {Config: changed, Check: check(1)}, {Config: changed, PlanOnly: true}}
	for _, p := range policies {
		ignored := []string{}
		for _, f := range p.Fields {
			if f.Sensitive {
				ignored = append(ignored, f.Name)
			}
		}
		steps = append(steps, resource.TestStep{ResourceName: "routeros_" + p.Name + ".test", ImportState: true, ImportStateId: strings.ReplaceAll(strings.TrimPrefix(p.Path, "/"), "/", "."), ImportStateVerify: true, ImportStateVerifyIgnore: ignored})
	}
	steps = append(steps, resource.TestStep{Config: changed, PreConfig: func() {
		for _, p := range policies {
			body := map[string]any{}
			for name, value := range additionalSettingsCases[p.Name][0] {
				body[strings.ReplaceAll(name, "_", "-")] = fmt.Sprint(value)
			}
			if err := c.Request(context.Background(), "POST", p.Path+"/set", "", nil, body, nil); err != nil {
				t.Fatal(err)
			}
		}
	}, Check: check(1)}, resource.TestStep{Config: changed, PlanOnly: true}, resource.TestStep{Config: first, Check: check(0)}, resource.TestStep{Config: first, PlanOnly: true})
	return steps
}
func additionalSettingsDestroyRetains(c *client.Client, policies []catalog.Collection) resource.TestCheckFunc {
	return func(*terraform.State) error {
		for _, p := range policies {
			var row map[string]any
			if err := c.Request(context.Background(), "GET", p.Path, "", nil, nil, &row); err != nil {
				return err
			}
			for _, f := range p.Fields {
				expected, configured := additionalSettingsCases[p.Name][0][f.Name]
				if !configured || f.Sensitive {
					continue
				}
				value, exists := row[f.Wire]
				if !exists {
					return fmt.Errorf("retained field missing %s.%s", p.Name, f.Name)
				}
				observed, err := decodeField(context.Background(), f, value)
				if err != nil {
					return err
				}
				expectedField, err := decodeField(context.Background(), f, fmt.Sprint(expected))
				if err != nil || !preserveReviewedSpelling(f, expectedField, observed).Equal(expectedField) {
					return fmt.Errorf("destroy reset settings %s.%s", p.Name, f.Name)
				}
			}
		}
		return nil
	}
}
func TestTerraformAdditionalSettingsMockLifecycle(t *testing.T) {
	var mu sync.Mutex
	rows := map[string]map[string]any{}
	policies := map[string]catalog.Collection{}
	writes := map[string]int{}
	for _, p := range catalog.AdditionalSettings() {
		rows[p.Path] = additionalSettingsBaseline(p)
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
			json.NewEncoder(w).Encode([]map[string]string{{"name": "routeros", "version": "7.24.5"}, {"name": "container", "version": "7.24.5"}, {"name": "wireless", "version": "7.24.5"}, {"name": "user-manager", "version": "7.24.5"}})
			return
		}
		path := strings.TrimPrefix(q.URL.Path, "/rest")
		if q.Method == "GET" {
			if rows[path] == nil {
				w.WriteHeader(404)
				return
			}
			result := map[string]any{}
			for k, v := range rows[path] {
				result[k] = v
			}
			for _, f := range policies[path].Fields {
				if f.Sensitive {
					delete(result, f.Wire)
				}
			}
			json.NewEncoder(w).Encode(result)
			return
		}
		if q.Method != "POST" || !strings.HasSuffix(path, "/set") {
			t.Error("unsupported mutation")
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
			return
		}
		allowed := map[string]bool{}
		for _, f := range p.Fields {
			if f.Mode != "computed" {
				allowed[f.Wire] = true
			}
		}
		for name, value := range body {
			if !allowed[name] {
				t.Error("unreviewed writable field")
			}
			rows[path][name] = value
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
	provider := fmt.Sprintf("provider \"routeros\" {\n hosturl = %q\n username = \"mock-user\"\n password = \"mock-only\"\n}\n", srv.URL)
	resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){"routeros": providerserver.NewProtocol6WithError(New("test")())}, Steps: additionalSettingsSteps(t, c, catalog.AdditionalSettings(), provider), CheckDestroy: additionalSettingsDestroyRetains(c, catalog.AdditionalSettings())})
	mu.Lock()
	defer mu.Unlock()
	for _, p := range catalog.AdditionalSettings() {
		if writes[p.Path] < 5 {
			t.Fatalf("missing create/update/drift/clearing evidence for %s: writes=%d", p.Name, writes[p.Path])
		}
	}
}
