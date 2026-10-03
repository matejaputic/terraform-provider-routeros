// SPDX-License-Identifier: MPL-2.0
package provider

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/matejaputic/terraform-provider-routeros/internal/client"
)

func dhcpOptionsConfig(provider string, changed bool) string {
	code, value := 77, "s'192.0.2.22'"
	extra := `comment = "first"`
	if changed {
		code, value = 78, "0x01020304"
		extra = "comment = \"second\"\n force = true"
	}
	return provider + fmt.Sprintf(`
resource "routeros_ip_dhcp_client_option" "test" {
 name = "tf-coverage-client-option"
 code = %[1]d
 value = %[2]q
}
resource "routeros_ip_dhcp_server_option" "test" {
 name = "tf-coverage-server-option"
 code = %[1]d
 value = %[2]q
 %[3]s
}
`, code, value, extra)
}
func dhcpOptionSteps(t *testing.T, c *client.Client, provider string) []resource.TestStep {
	t.Helper()
	initial, changed := dhcpOptionsConfig(provider, false), dhcpOptionsConfig(provider, true)
	checks := []resource.TestCheckFunc{resource.TestCheckResourceAttr("routeros_ip_dhcp_server_option.test", "force", "false")}
	for _, name := range dhcpOptionNames {
		address := "routeros_" + name + ".test"
		checks = append(checks, resource.TestCheckResourceAttrSet(address, "id"), resource.TestCheckResourceAttr(address, "code", "77"), resource.TestCheckResourceAttr(address, "raw_value", "3139322e302e322e3232"))
	}
	steps := []resource.TestStep{{Config: initial, Check: resource.ComposeTestCheckFunc(checks...)}, {Config: initial, PlanOnly: true}, {Config: initial}}
	for _, name := range dhcpOptionNames {
		steps = append(steps, resource.TestStep{ResourceName: "routeros_" + name + ".test", ImportState: true, ImportStateVerify: true})
	}
	checks = []resource.TestCheckFunc{resource.TestCheckResourceAttr("routeros_ip_dhcp_server_option.test", "force", "true")}
	for _, name := range dhcpOptionNames {
		address := "routeros_" + name + ".test"
		checks = append(checks, resource.TestCheckResourceAttr(address, "code", "78"), resource.TestCheckResourceAttr(address, "value", "0x01020304"), resource.TestCheckResourceAttr(address, "raw_value", "01020304"))
	}
	steps = append(steps, resource.TestStep{Config: changed, Check: resource.ComposeTestCheckFunc(checks...)}, resource.TestStep{Config: changed, PlanOnly: true})
	cleared := strings.ReplaceAll(strings.ReplaceAll(changed, `comment = "second"`, `comment = ""`), "force = true", "force = false")
	steps = append(steps, resource.TestStep{Config: cleared, Check: resource.ComposeTestCheckFunc(resource.TestCheckResourceAttr("routeros_ip_dhcp_server_option.test", "force", "false"), resource.TestCheckResourceAttr("routeros_ip_dhcp_server_option.test", "comment", ""))}, resource.TestStep{Config: cleared, PlanOnly: true})
	for _, name := range dhcpOptionNames {
		steps = append(steps, resource.TestStep{ResourceName: "routeros_" + name + ".test", ImportState: true, ImportStateVerify: true})
	}
	optionID := func(path, name string) string {
		t.Helper()
		var rows []map[string]any
		if err := c.Request(context.Background(), "GET", path, "", url.Values{"name": {name}}, nil, &rows); err != nil {
			t.Fatal(err)
		}
		if len(rows) != 1 {
			t.Fatalf("expected one owned option in %s", path)
		}
		id, ok := rows[0][".id"].(string)
		if !ok || !itemID.MatchString(id) {
			t.Fatal("invalid option ID")
		}
		return id
	}
	steps = append(steps, resource.TestStep{Config: cleared, PreConfig: func() {
		id := optionID("/ip/dhcp-server/option", "tf-coverage-server-option")
		if err := c.Request(context.Background(), "PATCH", "/ip/dhcp-server/option", id, nil, map[string]string{"code": "80", "value": "0xdeadbeef"}, nil); err != nil {
			t.Fatal(err)
		}
	}, Check: resource.ComposeTestCheckFunc(resource.TestCheckResourceAttr("routeros_ip_dhcp_server_option.test", "code", "78"), resource.TestCheckResourceAttr("routeros_ip_dhcp_server_option.test", "raw_value", "01020304"))}, resource.TestStep{Config: cleared, PlanOnly: true})
	var deletedID string
	steps = append(steps, resource.TestStep{Config: cleared, PreConfig: func() {
		deletedID = optionID("/ip/dhcp-client/option", "tf-coverage-client-option")
		if err := c.Request(context.Background(), "DELETE", "/ip/dhcp-client/option", deletedID, nil, nil, nil); err != nil {
			t.Fatal(err)
		}
	}, Check: func(s *terraform.State) error {
		if s.RootModule().Resources["routeros_ip_dhcp_client_option.test"].Primary.ID == deletedID {
			return fmt.Errorf("deleted client option not recreated")
		}
		return nil
	}}, resource.TestStep{Config: cleared, PlanOnly: true})
	renamed := strings.ReplaceAll(cleared, "tf-coverage-client-option", "tf-coverage-client-option-new")
	renamed = strings.ReplaceAll(renamed, "tf-coverage-server-option", "tf-coverage-server-option-new")
	oldIDs := map[string]string{}
	steps = append(steps, resource.TestStep{Config: renamed, PreConfig: func() {
		oldIDs["ip_dhcp_client_option"] = optionID("/ip/dhcp-client/option", "tf-coverage-client-option")
		oldIDs["ip_dhcp_server_option"] = optionID("/ip/dhcp-server/option", "tf-coverage-server-option")
	}, Check: func(s *terraform.State) error {
		for name, id := range oldIDs {
			if s.RootModule().Resources["routeros_"+name+".test"].Primary.ID == id {
				return fmt.Errorf("name replacement retained %s ID", name)
			}
		}
		return nil
	}}, resource.TestStep{Config: renamed, PlanOnly: true}, resource.TestStep{Config: renamed})
	return steps
}
func dhcpOptionDestroy(c *client.Client) resource.TestCheckFunc {
	return func(*terraform.State) error {
		for _, item := range []struct{ path, name string }{{"/ip/dhcp-client/option", "tf-coverage-client-option"}, {"/ip/dhcp-server/option", "tf-coverage-server-option"}} {
			for _, suffix := range []string{"", "-new"} {
				var rows []map[string]any
				if err := c.Request(context.Background(), "GET", item.path, "", url.Values{"name": {item.name + suffix}}, nil, &rows); err != nil {
					return err
				}
				if len(rows) != 0 {
					return fmt.Errorf("owned option remains in %s", item.path)
				}
			}
		}
		return nil
	}
}
func TestAccDHCPOptionsCHR(t *testing.T) {
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
	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){"routeros": providerserver.NewProtocol6WithError(New("test")())}, Steps: dhcpOptionSteps(t, c, `provider "routeros" {}`), CheckDestroy: dhcpOptionDestroy(c)})
}
