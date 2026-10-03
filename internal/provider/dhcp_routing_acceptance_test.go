// SPDX-License-Identifier: MPL-2.0
package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/matejaputic/terraform-provider-routeros/internal/client"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

func TestAccDHCPRoutingCHR(t *testing.T) {
	if os.Getenv("TF_ACC") != "1" {
		t.Skip("live CHR acceptance requires TF_ACC=1")
	}
	if os.Getenv("ROS_TEST_DISPOSABLE") != "1" || os.Getenv("ROS_HOSTURL") != "http://127.0.0.1:18780" {
		t.Fatal("only explicit disposable CHR allowed")
	}
	c, e := client.New(client.Config{HostURL: os.Getenv("ROS_HOSTURL"), Username: os.Getenv("ROS_USERNAME"), Password: os.Getenv("ROS_PASSWORD"), Timeout: 59 * time.Second})
	if e != nil {
		t.Fatal(e)
	}
	defer c.Close()
	var sys map[string]any
	if e = c.Request(context.Background(), "GET", "/system/resource", "", nil, nil, &sys); e != nil {
		t.Fatal(e)
	}
	if sys["version"] != "7.24.5 (stable)" || sys["architecture-name"] != "x86_64" {
		t.Fatal("wrong target lane")
	}
	cfg := func(comment string) string { return dhcpRoutingConfig(`provider "routeros" {}`, comment) }
	checks := []resource.TestCheckFunc{}
	for _, name := range dhcpRoutingNames {
		checks = append(checks, resource.TestCheckResourceAttrSet("routeros_"+name+".test", "id"), resource.TestCheckResourceAttr("routeros_"+name+".test", "comment", "first"))
	}
	checks = append(checks, resource.TestCheckResourceAttr("routeros_ip_dhcp_server_network.test", "dns_server.#", "2"), resource.TestCheckResourceAttr("routeros_ip_route.test", "distance", "7"), resource.TestCheckResourceAttr("routeros_ip_route.test", "static", "true"))
	steps := []resource.TestStep{{Config: cfg("first"), Check: resource.ComposeTestCheckFunc(checks...)}, {Config: cfg("first"), PlanOnly: true}, {Config: cfg("first")}, {Config: cfg("second")}}
	for _, name := range dhcpRoutingNames {
		steps = append(steps, resource.TestStep{ResourceName: "routeros_" + name + ".test", ImportState: true, ImportStateVerify: true})
	}
	changed := strings.ReplaceAll(cfg("second"), "distance = 7", "distance = 8")
	changed = strings.ReplaceAll(changed, `dns_server = ["192.0.2.53", "192.0.2.54"]`, `dns_server = ["192.0.2.55"]`)
	changed = strings.ReplaceAll(changed, "block_access = false", "block_access = true")
	changed = strings.ReplaceAll(changed, "add_arp = false", "add_arp = true")
	steps = append(steps, resource.TestStep{Config: changed, Check: resource.ComposeTestCheckFunc(resource.TestCheckResourceAttr("routeros_ip_route.test", "distance", "8"), resource.TestCheckResourceAttr("routeros_ip_dhcp_server_network.test", "dns_server.0", "192.0.2.55"), resource.TestCheckResourceAttr("routeros_ip_dhcp_server_lease.test", "block_access", "true"), resource.TestCheckResourceAttr("routeros_ip_dhcp_server.test", "add_arp", "true"))}, resource.TestStep{Config: changed, PlanOnly: true})
	empty := strings.ReplaceAll(changed, `dns_server = ["192.0.2.55"]`, `dns_server = []`)
	empty = strings.ReplaceAll(empty, `domain = "example.test"`, `domain = ""`)
	empty = strings.ReplaceAll(empty, `comment = "second"`, `comment = ""`)
	steps = append(steps, resource.TestStep{Config: empty, Check: resource.ComposeTestCheckFunc(resource.TestCheckResourceAttr("routeros_ip_dhcp_server_network.test", "dns_server.#", "0"), resource.TestCheckResourceAttr("routeros_ip_dhcp_server_network.test", "domain", ""))}, resource.TestStep{Config: empty, PlanOnly: true})
	none := strings.ReplaceAll(empty, "dns_none = false", "dns_none = true")
	none = strings.ReplaceAll(none, "02:00:00:00:00:AA", "02:00:00:00:00:aa")
	steps = append(steps, resource.TestStep{Config: none, Check: resource.ComposeTestCheckFunc(resource.TestCheckResourceAttr("routeros_ip_dhcp_server_network.test", "dns_none", "true"), resource.TestCheckResourceAttr("routeros_ip_dhcp_server_lease.test", "mac_address", "02:00:00:00:00:aa"))}, resource.TestStep{Config: none, PlanOnly: true}, resource.TestStep{Config: empty})
	var oldID string
	steps = append(steps, resource.TestStep{Config: empty, PreConfig: func() {
		var rows []map[string]any
		if e := c.Request(context.Background(), "GET", "/ip/route", "", url.Values{"dst-address": {"198.51.100.0/24"}}, nil, &rows); e != nil {
			t.Fatal(e)
		}
		if len(rows) != 1 {
			t.Fatal("test route missing")
		}
		oldID = rows[0][".id"].(string)
		if e := c.Request(context.Background(), "DELETE", "/ip/route", oldID, nil, nil, nil); e != nil {
			t.Fatal(e)
		}
	}, Check: func(s *terraform.State) error {
		if s.RootModule().Resources["routeros_ip_route.test"].Primary.ID == oldID {
			return fmt.Errorf("deleted route not recreated")
		}
		return nil
	}}, resource.TestStep{Config: empty, PlanOnly: true})
	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){"routeros": providerserver.NewProtocol6WithError(New("test")())}, Steps: steps, CheckDestroy: func(*terraform.State) error {
		for _, item := range []struct{ path, key, value string }{{"/ip/dhcp-server/network", "address", "192.0.2.0/24"}, {"/ip/dhcp-server", "name", "tf-coverage-dhcp"}, {"/ip/dhcp-server/lease", "address", "192.0.2.15"}, {"/ip/route", "dst-address", "198.51.100.0/24"}} {
			var rows []map[string]any
			if e := c.Request(context.Background(), "GET", item.path, "", url.Values{item.key: {item.value}}, nil, &rows); e != nil {
				return e
			}
			if len(rows) != 0 {
				return fmt.Errorf("test object remains in %s", item.path)
			}
		}
		return nil
	}})
}
