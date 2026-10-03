// SPDX-License-Identifier: MPL-2.0
package provider

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/matejaputic/terraform-provider-routeros/internal/client"
	"github.com/matejaputic/terraform-provider-routeros/internal/generated"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRefreshDeletionAndIDs(t *testing.T) {
	for _, tc := range []struct {
		name, body     string
		status         int
		found, wantErr bool
	}{
		{"deleted", "[]", 200, false, false}, {"not found", "", 404, false, false},
		{"unauthorized", "secret", 401, false, true},
		{"wrong id", `[{".id":"*3"}]`, 200, false, true},
		{"duplicates", `[{".id":"*2"},{".id":"*2"}]`, 200, false, true},
		{"invalid boolean", `[{".id":"*2","disabled":"maybe"}]`, 200, false, true},
		{"valid", `[{".id":"*2","address":"192.0.2.1/24","interface":"ether1","disabled":"yes","dynamic":false,"invalid":"no","slave":"false"}]`, 200, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(tc.status); w.Write([]byte(tc.body)) }))
			defer srv.Close()
			c, e := client.New(client.Config{HostURL: srv.URL, Timeout: time.Second})
			if e != nil {
				t.Fatal(e)
			}
			r := ipAddressResource{client: c}
			m := generated.IpAddressModel{Id: types.StringValue("*2")}
			found, e := r.refresh(context.Background(), &m)
			if found != tc.found || (e != nil) != tc.wantErr {
				t.Fatalf("found %v error %v", found, e)
			}
			if found && !m.Disabled.ValueBool() {
				t.Fatal("yes boolean was not decoded")
			}
		})
	}
}
func TestEnvironmentPrecedence(t *testing.T) {
	t.Setenv("ROS_HOSTURL", "first")
	t.Setenv("MIKROTIK_HOST", "second")
	if got := fallback(types.StringNull(), "ROS_HOSTURL", "MIKROTIK_HOST"); got != "first" {
		t.Fatal(got)
	}
	if got := fallback(types.StringValue(""), "ROS_HOSTURL"); got != "" {
		t.Fatal("explicit empty config was overwritten")
	}
	if got := fallback(types.StringValue("configured"), "ROS_HOSTURL"); got != "configured" {
		t.Fatal(got)
	}
}
