// SPDX-License-Identifier: MPL-2.0
package provider

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/matejaputic/terraform-provider-routeros/internal/client"
)

func TestAccDHCPRelayCHR(t *testing.T) {
	if os.Getenv("TF_ACC") != "1" {
		t.Skip("live CHR acceptance requires TF_ACC=1")
	}
	if os.Getenv("ROS_TEST_DISPOSABLE") != "1" || os.Getenv("ROS_HOSTURL") != "http://127.0.0.1:18780" {
		t.Fatal("only explicit disposable CHR allowed")
	}
	c, err := client.New(client.Config{HostURL: os.Getenv("ROS_HOSTURL"), Username: os.Getenv("ROS_USERNAME"), Password: os.Getenv("ROS_PASSWORD"), Timeout: 59 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	var system map[string]any
	if err := c.Request(context.Background(), "GET", "/system/resource", "", nil, nil, &system); err != nil {
		t.Fatal(err)
	}
	if system["version"] != expectedCHRVersion(t) || system["architecture-name"] != "x86_64" {
		t.Fatal("wrong target lane")
	}
	config := func(name, servers string, info bool) string {
		return fmt.Sprintf(`provider "routeros" {}
resource "routeros_ip_dhcp_relay" "test" {
 name = %q
 interface = "tf-test"
 dhcp_server = %q
 disabled = true
 local_address = "0.0.0.0"
 add_relay_info = %t
}`, name, servers, info)
	}
	first := config("tf-coverage-relay", "192.0.2.1", false)
	changed := config("tf-coverage-relay", "192.0.2.2", true)
	renamed := config("tf-coverage-relay-new", "192.0.2.2", false)
	relayID := func(name string) string {
		t.Helper()
		var rows []map[string]any
		if err := c.Request(context.Background(), "GET", "/ip/dhcp-relay", "", url.Values{"name": {name}}, nil, &rows); err != nil {
			t.Fatal(err)
		}
		if len(rows) != 1 {
			t.Fatal("expected one owned relay")
		}
		id, ok := rows[0][".id"].(string)
		if !ok || !itemID.MatchString(id) {
			t.Fatal("invalid relay ID")
		}
		return id
	}
	var deletedID, oldID string
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){"routeros": providerserver.NewProtocol6WithError(New("test")())},
		Steps: []resource.TestStep{
			{Config: first, Check: resource.TestCheckResourceAttr("routeros_ip_dhcp_relay.test", "disabled", "true")},
			{Config: first, PlanOnly: true},
			{ResourceName: "routeros_ip_dhcp_relay.test", ImportState: true, ImportStateVerify: true},
			{Config: changed, Check: resource.TestCheckResourceAttr("routeros_ip_dhcp_relay.test", "add_relay_info", "true")},
			{Config: changed, PlanOnly: true},
			{Config: changed, PreConfig: func() {
				if err := c.Request(context.Background(), "PATCH", "/ip/dhcp-relay", relayID("tf-coverage-relay"), nil, map[string]string{"dhcp-server": "192.0.2.99"}, nil); err != nil {
					t.Fatal(err)
				}
			}, Check: resource.TestCheckResourceAttr("routeros_ip_dhcp_relay.test", "dhcp_server", "192.0.2.2")},
			{Config: changed, PlanOnly: true},
			{Config: changed, PreConfig: func() {
				deletedID = relayID("tf-coverage-relay")
				if err := c.Request(context.Background(), "DELETE", "/ip/dhcp-relay", deletedID, nil, nil, nil); err != nil {
					t.Fatal(err)
				}
			}, Check: func(s *terraform.State) error {
				if s.RootModule().Resources["routeros_ip_dhcp_relay.test"].Primary.ID == deletedID {
					return fmt.Errorf("deleted relay not recreated")
				}
				return nil
			}},
			{Config: renamed, PreConfig: func() { oldID = relayID("tf-coverage-relay") }, Check: func(s *terraform.State) error {
				if s.RootModule().Resources["routeros_ip_dhcp_relay.test"].Primary.ID == oldID {
					return fmt.Errorf("name replacement retained ID")
				}
				return nil
			}},
			{Config: renamed, PlanOnly: true},
			{ResourceName: "routeros_ip_dhcp_relay.test", ImportState: true, ImportStateVerify: true},
		},
		CheckDestroy: func(*terraform.State) error {
			for _, name := range []string{"tf-coverage-relay", "tf-coverage-relay-new"} {
				var rows []map[string]any
				if err := c.Request(context.Background(), "GET", "/ip/dhcp-relay", "", url.Values{"name": {name}}, nil, &rows); err != nil {
					return err
				}
				if len(rows) != 0 {
					return fmt.Errorf("owned relay remains")
				}
			}
			return nil
		},
	})
}
