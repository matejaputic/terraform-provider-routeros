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
	"github.com/matejaputic/terraform-provider-routeros/internal/catalog"
	"github.com/matejaputic/terraform-provider-routeros/internal/client"
)

func TestAccCollectionsCHR(t *testing.T) {
	if os.Getenv("TF_ACC") != "1" {
		t.Skip("live CHR acceptance requires TF_ACC=1")
	}
	if os.Getenv("ROS_TEST_DISPOSABLE") != "1" || os.Getenv("ROS_HOSTURL") != "http://127.0.0.1:18780" {
		t.Fatal("only the explicit loopback disposable CHR is allowed")
	}
	c, e := client.New(client.Config{HostURL: os.Getenv("ROS_HOSTURL"), Username: os.Getenv("ROS_USERNAME"), Password: os.Getenv("ROS_PASSWORD"), Timeout: 59 * time.Second})
	if e != nil {
		t.Fatal(e)
	}
	defer c.Close()
	var system map[string]any
	if e := c.Request(context.Background(), "GET", "/system/resource", "", nil, nil, &system); e != nil {
		t.Fatal(e)
	}
	if system["version"] != expectedCHRVersion(t) || system["architecture-name"] != "x86_64" {
		t.Fatal("unexpected target lane")
	}
	cfg := func(comment string) string { return networkingConfig(`provider "routeros" {}`, comment) }
	checks := []resource.TestCheckFunc{}
	for _, name := range collectionTestNames {
		checks = append(checks, resource.TestCheckResourceAttrSet("routeros_"+name+".test", "id"), resource.TestCheckResourceAttr("routeros_"+name+".test", "comment", "first"))
	}
	checks = append(checks, resource.TestCheckResourceAttr("routeros_interface_vlan.test", "vlan_id", "123"), resource.TestCheckResourceAttr("routeros_interface_bridge_port.test", "pvid", "10"), resource.TestCheckResourceAttr("routeros_interface_list.test", "builtin", "false"), resource.TestCheckResourceAttr("routeros_ip_pool.test", "ranges.#", "2"))
	steps := []resource.TestStep{{Config: cfg("first"), Check: resource.ComposeTestCheckFunc(checks...)}, {Config: cfg("first"), PlanOnly: true}, {Config: cfg("first")}, {Config: cfg("second")}}
	for _, name := range collectionTestNames {
		steps = append(steps, resource.TestStep{ResourceName: "routeros_" + name + ".test", ImportState: true, ImportStateVerify: true})
	}
	steps = append(steps, resource.TestStep{Config: cfg("second"), PlanOnly: true}, resource.TestStep{Config: cfg("second")})
	// Exercise a real address-range/number/boolean update, not just comments.
	changed := strings.ReplaceAll(cfg("second"), "vlan_id = 123", "vlan_id = 124")
	changed = strings.ReplaceAll(changed, "pvid = 10", "pvid = 20")
	changed = strings.ReplaceAll(changed, "disabled = false", "disabled = true")
	changed = strings.ReplaceAll(changed, "192.0.2.10-192.0.2.20", "192.0.2.11-192.0.2.21")
	steps = append(steps, resource.TestStep{Config: changed, Check: resource.ComposeTestCheckFunc(resource.TestCheckResourceAttr("routeros_interface_vlan.test", "vlan_id", "124"), resource.TestCheckResourceAttr("routeros_interface_bridge_port.test", "pvid", "20"), resource.TestCheckResourceAttr("routeros_interface_list_member.test", "disabled", "true"), resource.TestCheckResourceAttr("routeros_ip_pool.test", "ranges.0", "192.0.2.11-192.0.2.21"))}, resource.TestStep{Config: changed, PlanOnly: true})
	var deletedID string
	steps = append(steps, resource.TestStep{Config: changed, PreConfig: func() {
		var rows []map[string]any
		if e := c.Request(context.Background(), "GET", "/ip/pool", "", url.Values{"name": {"tf-coverage-pool"}}, nil, &rows); e != nil {
			t.Fatal(e)
		}
		if len(rows) != 1 {
			t.Fatal("missing test pool")
		}
		deletedID = rows[0][".id"].(string)
		if e := c.Request(context.Background(), "DELETE", "/ip/pool", deletedID, nil, nil, nil); e != nil {
			t.Fatal(e)
		}
	}, Check: func(s *terraform.State) error {
		if s.RootModule().Resources["routeros_ip_pool.test"].Primary.ID == deletedID {
			return fmt.Errorf("externally deleted pool was not recreated")
		}
		return nil
	}})
	// Name replacement is intentional reference behavior, not in-place renaming.
	renamed := strings.ReplaceAll(changed, `name = "tf-coverage-pool"`, `name = "tf-coverage-pool-renamed"`)
	steps = append(steps, resource.TestStep{Config: renamed}, resource.TestStep{Config: renamed, PlanOnly: true})
	emptyComment := strings.ReplaceAll(renamed, `comment = "second"`, `comment = ""`)
	emptyChecks := []resource.TestCheckFunc{}
	for _, name := range collectionTestNames {
		emptyChecks = append(emptyChecks, resource.TestCheckResourceAttr("routeros_"+name+".test", "comment", ""))
	}
	steps = append(steps, resource.TestStep{Config: emptyComment, Check: resource.ComposeTestCheckFunc(emptyChecks...)}, resource.TestStep{Config: emptyComment, PlanOnly: true})
	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){"routeros": providerserver.NewProtocol6WithError(New("test")())}, Steps: steps, CheckDestroy: func(*terraform.State) error {
		for _, policy := range catalog.Collections() {
			var rows []map[string]any
			if e := c.Request(context.Background(), "GET", policy.Path, "", nil, nil, &rows); e != nil {
				return e
			}
			for _, row := range rows {
				if name, ok := row["name"].(string); ok && strings.HasPrefix(name, "tf-coverage-") {
					return fmt.Errorf("test object remains in %s", policy.Path)
				}
				if row["interface"] == "tf-port" && policy.Name == "interface_bridge_port" {
					return fmt.Errorf("test bridge port remains")
				}
			}
		}
		return nil
	}})
}
