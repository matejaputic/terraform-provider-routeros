// SPDX-License-Identifier: MPL-2.0
package provider

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/matejaputic/terraform-provider-routeros/internal/client"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestFirewallOrderingFailsClosed(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct{ rows, anchor, id string }{
		{`[{".id":"*A","chain":"other","dynamic":"false"}]`, "*A", ""},
		{`[{".id":"*A","chain":"test","dynamic":"true"}]`, "*A", ""},
		{`[{".id":"*A","chain":"test"},{".id":"*A","chain":"test"}]`, "*A", ""},
		{`[{".id":"*A","chain":"test","dynamic":null}]`, "*A", ""},
		{`[{".id":"*A","chain":"test"}]`, "*A", "*A"},
		{`[]`, "0", ""}, {`[]`, "*B", ""}, {`[]`, "", "*A"},
	} {
		writes := 0
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) {
			if q.Method != "GET" {
				writes++
				w.WriteHeader(500)
				return
			}
			w.Write([]byte(tc.rows))
		}))
		c, _ := client.New(client.Config{HostURL: s.URL, Timeout: time.Second})
		r := collectionConstructors()[11]().(*collectionResource)
		r.client = c
		values := map[string]attr.Value{"chain": types.StringValue("test"), "place_before": types.StringValue(tc.anchor)}
		if e := r.positionFilter(ctx, tc.id, values, true); e == nil {
			t.Fatal("unsafe move accepted")
		}
		if writes != 0 {
			t.Fatal("unsafe ordering mutated router")
		}
		c.Close()
		s.Close()
	}
}
func TestFirewallAddressBounds(t *testing.T) {
	for _, bad := range []string{"host.example", "::1", "192.0.2.1/24", "192.0.2.2-192.0.2.1", "300.1.1.1", " 192.0.2.1"} {
		if _, e := firewallAddressBounds(bad); e == nil {
			t.Fatal("unsupported address accepted")
		}
	}
	for _, pair := range [][2]string{{"192.0.2.0/24", "192.0.2.0-192.0.2.255"}, {"192.0.2.1/32", "192.0.2.1"}, {"0.0.0.0/0", "0.0.0.0-255.255.255.255"}} {
		a, e := firewallAddressBounds(pair[0])
		b, err := firewallAddressBounds(pair[1])
		if e != nil || err != nil || a != b {
			t.Fatal("numeric address equivalence lost")
		}
	}
}
