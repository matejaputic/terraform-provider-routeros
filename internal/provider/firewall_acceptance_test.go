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

func TestAccFirewallCHR(t *testing.T) { testFirewallCHR(t, "filter") }
func TestAccFirewallFamiliesCHR(t *testing.T) {
	if os.Getenv("TF_ACC") != "1" {
		t.Skip("live CHR requires TF_ACC=1")
	}
	for _, family := range []string{"nat", "mangle", "raw"} {
		t.Run(family, func(t *testing.T) { testFirewallCHR(t, family) })
	}
}
func testFirewallCHR(t *testing.T, family string) {
	if os.Getenv("TF_ACC") != "1" {
		t.Skip("live CHR requires TF_ACC=1")
	}
	if os.Getenv("ROS_TEST_DISPOSABLE") != "1" || os.Getenv("ROS_HOSTURL") != "http://127.0.0.1:18780" {
		t.Fatal("explicit disposable CHR only")
	}
	ctx := context.Background()
	c, e := client.New(client.Config{HostURL: os.Getenv("ROS_HOSTURL"), Username: os.Getenv("ROS_USERNAME"), Password: os.Getenv("ROS_PASSWORD"), Timeout: 59 * time.Second})
	if e != nil {
		t.Fatal(e)
	}
	defer c.Close()
	var sys map[string]any
	if e = c.Request(ctx, "GET", "/system/resource", "", nil, nil, &sys); e != nil {
		t.Fatal(e)
	}
	if sys["version"] != "7.24.5 (stable)" || sys["architecture-name"] != "x86_64" {
		t.Fatal("wrong lane")
	}
	cfg := func(comment, before string) string {
		return firewallFamilyConfig(family, `provider "routeros" {}`, comment, before)
	}
	prefix := "routeros_ip_firewall_" + family
	anchor := prefix + ".tail.id"
	menu := "/ip/firewall/" + family
	chain := "tf-coverage-" + family
	checkOrder := func(first string) resource.TestCheckFunc {
		return func(s *terraform.State) error {
			var rows []map[string]any
			if e := c.Request(ctx, "GET", menu, "", url.Values{"chain": {chain}}, nil, &rows); e != nil {
				return e
			}
			if len(rows) != 2 || rows[0][".id"] != s.RootModule().Resources[prefix+"."+first].Primary.ID {
				return fmt.Errorf("unexpected chain order")
			}
			return nil
		}
	}
	steps := []resource.TestStep{{Config: cfg("first", anchor), Check: resource.ComposeTestCheckFunc(checkOrder("head"), resource.TestCheckResourceAttr("routeros_ip_firewall_addr_list.test", "address", "192.0.2.0-192.0.2.255"))}, {Config: cfg("first", anchor), PlanOnly: true}, {Config: cfg("second", anchor)}}
	for _, name := range []string{"routeros_ip_firewall_addr_list.test", prefix + ".head", prefix + ".tail"} {
		step := resource.TestStep{ResourceName: name, ImportState: true, ImportStateVerify: true}
		if name == "routeros_ip_firewall_addr_list.test" {
			step.ImportStateVerifyIgnore = []string{"address"}
		}
		steps = append(steps, step)
	}
	steps = append(steps, resource.TestStep{Config: cfg("second", `""`), Check: checkOrder("tail")}, resource.TestStep{Config: cfg("second", `""`), PlanOnly: true}, resource.TestStep{Config: cfg("second", anchor), Check: checkOrder("head")})
	steps = append(steps, resource.TestStep{Config: cfg("second", anchor), PreConfig: func() {
		var rows []map[string]any
		if e := c.Request(ctx, "GET", menu, "", url.Values{"chain": {chain}}, nil, &rows); e != nil {
			t.Fatal(e)
		}
		if len(rows) != 2 {
			t.Fatal("missing chain")
		}
		if e := c.Request(ctx, "POST", menu+"/move", "", nil, map[string]string{"numbers": rows[0][".id"].(string)}, nil); e != nil {
			t.Fatal(e)
		}
	}, Check: checkOrder("head")}, resource.TestStep{Config: cfg("second", anchor), PlanOnly: true})
	changed := strings.ReplaceAll(cfg("second", anchor), "192.0.2.0-192.0.2.255", "192.0.2.128/25")
	changed = strings.ReplaceAll(changed, "log = false", "log = true")
	changed = strings.ReplaceAll(changed, `comment = "second"`, `comment = ""`)
	changed = strings.ReplaceAll(changed, "src_address_list = routeros_ip_firewall_addr_list.test.list", `src_address_list = ""`)
	steps = append(steps, resource.TestStep{Config: changed, Check: resource.ComposeTestCheckFunc(checkOrder("head"), resource.TestCheckResourceAttr(prefix+".head", "log", "true"), resource.TestCheckResourceAttr("routeros_ip_firewall_addr_list.test", "address", "192.0.2.128/25"))}, resource.TestStep{Config: changed, PlanOnly: true})
	movedChain := strings.ReplaceAll(changed, chain, chain+"-moved")
	steps = append(steps, resource.TestStep{Config: movedChain, Check: resource.TestCheckResourceAttr(prefix+".head", "chain", chain+"-moved")}, resource.TestStep{Config: movedChain, PlanOnly: true}, resource.TestStep{Config: changed, Check: checkOrder("head")})
	switch family {
	case "nat":
		dnat := strings.ReplaceAll(changed, `action = "src-nat"`, `action = "dst-nat"`)
		dnat = strings.ReplaceAll(dnat, "192.0.2.129", "192.0.2.130")
		masquerade := strings.ReplaceAll(dnat, `action = "dst-nat"`, `action = "masquerade"`)
		masquerade = strings.ReplaceAll(masquerade, ` to_addresses = "192.0.2.130"`, "")
		steps = append(steps, resource.TestStep{Config: dnat, Check: resource.TestCheckResourceAttr(prefix+".head", "to_addresses", "192.0.2.130")}, resource.TestStep{Config: dnat, PlanOnly: true}, resource.TestStep{Config: masquerade}, resource.TestStep{Config: masquerade, PlanOnly: true}, resource.TestStep{Config: changed})
	case "mangle":
		packet := strings.ReplaceAll(changed, `action = "mark-connection"`, `action = "mark-packet"`)
		packet = strings.ReplaceAll(packet, "new_connection_mark =", "new_packet_mark =")
		packet = strings.ReplaceAll(packet, "passthrough = false", "passthrough = true")
		steps = append(steps, resource.TestStep{Config: packet, Check: resource.ComposeTestCheckFunc(resource.TestCheckResourceAttr(prefix+".head", "new_packet_mark", "tf-mark-second"), resource.TestCheckResourceAttr(prefix+".head", "passthrough", "true"))}, resource.TestStep{Config: packet, PlanOnly: true}, resource.TestStep{Config: changed})
	}
	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){"routeros": providerserver.NewProtocol6WithError(New("test")())}, Steps: steps, CheckDestroy: func(*terraform.State) error {
		for _, v := range []struct{ path, key, value string }{{menu, "chain", chain}, {menu, "chain", chain + "-moved"}, {"/ip/firewall/address-list", "list", "tf-coverage-firewall"}} {
			var rows []map[string]any
			if e := c.Request(ctx, "GET", v.path, "", url.Values{v.key: {v.value}}, nil, &rows); e != nil {
				return e
			}
			if len(rows) != 0 {
				return fmt.Errorf("firewall objects remain")
			}
		}
		return nil
	}})
}
