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
	"os"
	"strings"
	"testing"
	"time"
)

// Explicit disposable-target opt-in prevents accidental mutation of a real router.
func TestAccIPAddressCHR(t *testing.T) {
	if os.Getenv("TF_ACC") != "1" {
		t.Skip("live CHR acceptance requires TF_ACC=1")
	}
	if os.Getenv("ROS_TEST_DISPOSABLE") != "1" {
		t.Fatal("set ROS_TEST_DISPOSABLE=1 only for a disposable CHR")
	}
	host := os.Getenv("ROS_HOSTURL")
	if host != "http://127.0.0.1:18780" {
		t.Fatal("this fixture only permits the loopback disposable CHR")
	}
	c, e := client.New(client.Config{HostURL: host, Username: os.Getenv("ROS_USERNAME"), Password: os.Getenv("ROS_PASSWORD"), Timeout: 59 * time.Second})
	if e != nil {
		t.Fatal(e)
	}
	var target map[string]any
	if e = c.Request(context.Background(), "GET", "/system/resource", "", nil, nil, &target); e != nil {
		t.Fatal(e)
	}
	if target["version"] != "7.24.5 (stable)" || target["architecture-name"] != "x86_64" {
		t.Fatalf("unexpected acceptance target: version=%v arch=%v", target["version"], target["architecture-name"])
	}
	t.Logf("CHR acceptance target: version=%v architecture=%v board=%v", target["version"], target["architecture-name"], target["board-name"])
	var firstID string
	config := func(comment string, disabled bool) string {
		return fmt.Sprintf(`provider "routeros" {}
resource "routeros_ip_address" "test" {
 address = "192.0.2.1/24"
 interface = "tf-test"
 comment = %q
 disabled = %t
}`, comment, disabled)
	}
	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){"routeros": providerserver.NewProtocol6WithError(New("test")())}, CheckDestroy: func(s *terraform.State) error {
		var rows []map[string]any
		if e := c.Request(context.Background(), "GET", "/ip/address", "", nil, nil, &rows); e != nil {
			return e
		}
		for _, row := range rows {
			if row["interface"] == "tf-test" {
				return fmt.Errorf("test IP address remains after destroy")
			}
		}
		return nil
	}, Steps: []resource.TestStep{
		{Config: config("terraform-acceptance-first", false), Check: resource.ComposeTestCheckFunc(resource.TestCheckResourceAttrSet("routeros_ip_address.test", "id"), resource.TestCheckResourceAttr("routeros_ip_address.test", "actual_interface", "tf-test"), resource.TestCheckResourceAttr("routeros_ip_address.test", "dynamic", "false"), resource.TestCheckResourceAttr("routeros_ip_address.test", "network", "192.0.2.0"),
			resource.TestCheckResourceAttr("routeros_ip_address.test", "vrf", "main"),
			resource.TestCheckResourceAttr("routeros_ip_address.test", "invalid", "false"),
			func(s *terraform.State) error {
				firstID = s.RootModule().Resources["routeros_ip_address.test"].Primary.ID
				return nil
			})},
		{Config: config("terraform-acceptance-first", false), PlanOnly: true},
		// A second real apply must also be empty, not only a plan-only check.
		{Config: config("terraform-acceptance-first", false)},
		{Config: config("terraform-acceptance-updated", true), Check: resource.ComposeTestCheckFunc(resource.TestCheckResourceAttr("routeros_ip_address.test", "disabled", "true"), resource.TestCheckResourceAttr("routeros_ip_address.test", "comment", "terraform-acceptance-updated"))},
		{ResourceName: "routeros_ip_address.test", ImportState: true, ImportStateVerify: true},
		{Config: config("terraform-acceptance-updated", true), PlanOnly: true},
		{Config: config("terraform-acceptance-updated", true)},
		// Delete outside Terraform, then prove refresh removes stale state and
		// apply recreates it without damaging the management interface.
		{Config: config("terraform-acceptance-updated", true), PreConfig: func() {
			var rows []map[string]any
			if e := c.Request(context.Background(), "GET", "/ip/address", "", nil, nil, &rows); e != nil {
				t.Fatal(e)
			}
			for _, row := range rows {
				if row["interface"] == "tf-test" {
					if e := c.Request(context.Background(), "DELETE", "/ip/address", fmt.Sprint(row[".id"]), nil, nil, nil); e != nil {
						t.Fatal(e)
					}
				}
			}
		}, Check: func(s *terraform.State) error {
			id := s.RootModule().Resources["routeros_ip_address.test"].Primary.ID
			if id == "" || id == firstID {
				return fmt.Errorf("recreated object must have a new unique ID")
			}
			return nil
		}},
		{Config: config("terraform-acceptance-updated", true), PlanOnly: true},
		{Config: strings.Replace(config("terraform-acceptance-updated", true), "192.0.2.1/24", "192.0.2.2", 1), Check: resource.TestCheckResourceAttr("routeros_ip_address.test", "address", "192.0.2.2")},
		{Config: strings.Replace(config("terraform-acceptance-updated", true), "192.0.2.1/24", "192.0.2.2", 1), PlanOnly: true},
	}})
}
