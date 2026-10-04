// SPDX-License-Identifier: MPL-2.0
package provider

import (
	"context"
	"encoding/json"
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

// Explicit owned, disabled/unattached configuration; not inferred from schema.
// Parent references form a Terraform dependency graph. No administrator key,
// script execution, management interface, global reset or traffic assertion.
var extendedResourceLiveBodies = map[string]string{
	"ip_dhcp_client":                  `interface="tf-test" add_default_route="no" use_peer_dns=false`,
	"ip_dhcp_server_option_matcher":   `name="tf-batch-matcher" server=routeros_ip_dhcp_server.batch_parent.name address_pool=routeros_ip_pool.batch_parent.name code=60 value="tf-batch" matching_type="exact"`,
	"ip_dhcp_server_option_set":       `name="tf-batch-options" options=routeros_ip_dhcp_server_option.batch_parent.name`,
	"ip_dns_adlist":                   `url="https://invalid.invalid/tf-batch.txt"`,
	"ip_vrf":                          `name="tf-batch-vrf" interfaces="none"`,
	"ipv6_address":                    `address="2001:db8:107::1/64" interface="tf-test" advertise=false`,
	"ipv6_dhcp_client":                `interface="tf-test" request="prefix" pool_name="tf-batch-received" add_default_route=false use_peer_dns=false`,
	"ipv6_dhcp_client_option":         `name="tf-batch-v6client-option" code=100 value="0x01"`,
	"ipv6_firewall_addr_list":         `list="tf-reviewed-resourcesddresses" address="2001:db8:107::/64"`,
	"ipv6_firewall_filter":            `chain="tf-batch-unattached" action="drop"`,
	"ipv6_neighbor_discovery":         `interface="tf-test"`,
	"ipv6_route":                      `dst_address="2001:db8:108::/64" gateway="2001:db8:107::2%tf-test"`,
	"interface_6to4":                  `name="tf-batch-6to4" local_address="192.0.2.1" remote_address="192.0.2.2"`,
	"interface_bonding":               `name="tf-remaining-resourcesond" slaves="tf-port" mode="active-backup"`,
	"interface_bridge_vlan":           `bridge="tf-test" vlan_ids="107" tagged="tf-test"`,
	"interface_dot1x_client":          `interface="tf-port" identity="tf-batch-dot1x" eap_methods="eap-mschapv2"`,
	"interface_dot1x_server":          `interface="tf-port"`,
	"interface_gre6":                  `name="tf-batch-gre6" local_address="2001:db8::1" remote_address="2001:db8::2"`,
	"interface_l2tp_client":           `name="tf-batch-l2tp" connect_to="192.0.2.2" user="tf-batch-ppp" add_default_route=false`,
	"interface_macvlan":               `name="tf-batch-macvlan" interface="tf-test"`,
	"interface_ovpn_server":           `name="tf-batch-ovpn-static" user="tf-batch-ppp"`,
	"interface_pppoe_client":          `name="tf-batch-pppoe" interface="tf-test" user="tf-batch-ppp" add_default_route=false use_peer_dns=false`,
	"interface_pppoe_server":          `name="tf-batch-pppoe-static" user="tf-batch-ppp" service="tf-batch-service"`,
	"interface_sstp_client":           `name="tf-batch-sstp" connect_to="192.0.2.2" user="tf-batch-ppp" add_default_route=false`,
	"interface_veth":                  `name="tf-batch-veth" address="192.0.2.10/24" gateway="192.0.2.1"`,
	"interface_vxlan":                 `name="tf-batch-vxlan" vni=107 local_address="192.0.2.1"`,
	"interface_vxlan_vteps":           `interface=routeros_interface_vxlan.test.name remote_ip="192.0.2.2"`,
	"interface_wireguard":             `name="tf-batch-wg" listen_port=21070`,
	"interface_wireguard_peer":        `interface=routeros_interface_wireguard.test.name allowed_address="192.0.2.0/24"`,
	"ip_dns_forwarders":               `name="tf-batch-forward" dns_servers="192.0.2.53"`,
	"ip_firewall_layer7_protocol":     `name="tf-batch-layer7" regexp="^tf-batch$"`,
	"ip_hotspot":                      `name="tf-batch-hotspot" interface="tf-test" address_pool=routeros_ip_pool.batch_parent.name profile=routeros_ip_hotspot_profile.test.name`,
	"ip_hotspot_ip_binding":           `server=routeros_ip_hotspot.test.name address="192.0.2.10" type="bypassed"`,
	"ip_hotspot_profile":              `name="tf-batch-hotspot-profile" hotspot_address="192.0.2.1" login_by="http-chap" use_radius=false`,
	"ip_hotspot_user":                 `name="tf-batch-hotspot-user" server=routeros_ip_hotspot.test.name profile=routeros_ip_hotspot_user_profile.test.name`,
	"ip_hotspot_user_profile":         `name="tf-batch-hotspot-users" shared_users=1`,
	"ip_hotspot_walled_garden":        `server=routeros_ip_hotspot.test.name dst_host="invalid.invalid" action="allow"`,
	"ip_hotspot_walled_garden_ip":     `server=routeros_ip_hotspot.test.name dst_address="192.0.2.0/24" action="accept"`,
	"ip_ipsec_identity":               `peer=routeros_ip_ipsec_peer.test.name auth_method="pre-shared-key"`,
	"ip_ipsec_mode_config":            `name="tf-batch-mode" address_pool=routeros_ip_pool.batch_parent.name responder=true`,
	"ip_ipsec_peer":                   `name="tf-batch-ipsec-peer" address="192.0.2.2/32" profile=routeros_ip_ipsec_profile.test.name exchange_mode="ike2" passive=true`,
	"ip_ipsec_policy":                 `src_address="192.0.2.0/24" dst_address="198.51.100.0/24" template=true group=routeros_ip_ipsec_policy_group.test.name proposal=routeros_ip_ipsec_proposal.test.name`,
	"ip_ipsec_policy_group":           `name="tf-batch-policies"`,
	"ip_ipsec_profile":                `name="tf-batch-ipsec-profile" dh_group="modp2048" enc_algorithm="aes-128" hash_algorithm="sha256"`,
	"ip_ipsec_proposal":               `name="tf-batch-proposal" auth_algorithms="sha256" enc_algorithms="aes-128-cbc" pfs_group="modp2048"`,
	"ip_tftp":                         `req_filename="tf-batch-never-match" real_filename="tf-batch-not-present" allow=false read_only=true`,
	"ip_traffic_flow_target":          `dst_address="192.0.2.2" port=21071 version="9"`,
	"ipv6_dhcp_server":                `name="tf-batch-v6server" interface="tf-test" prefix_pool=routeros_ipv6_pool.test.name`,
	"ipv6_dhcp_server_option":         `name="tf-batch-v6-option" code=100 value="0x01"`,
	"ipv6_dhcp_server_option_sets":    `name="tf-batch-v6-options" options=routeros_ipv6_dhcp_server_option.test.name`,
	"ipv6_nd_prefix":                  `interface="tf-test" prefix="2001:db8:107::/64"`,
	"ipv6_pool":                       `name="tf-batch-v6pool" prefix="2001:db8:107::/48" prefix_length=64`,
	"ip_nat_pmp_interfaces":           `interface="tf-test" type="internal"`,
	"interface_ovpn_client":           `name="tf-batch-ovpn" connect_to="192.0.2.2" user="tf-batch-ppp" add_default_route=false`,
	"ppp_profile":                     `name="tf-batch-ppp-profile" local_address="192.0.2.1" remote_address=routeros_ip_pool.batch_parent.name`,
	"ppp_secret":                      `name="tf-batch-ppp" profile=routeros_ppp_profile.test.name service="any"`,
	"queue_simple":                    `name="tf-batch-simple" target="192.0.2.0/24" max_limit="1000000/1000000"`,
	"queue_tree":                      `name="tf-batch-tree" parent="tf-test" packet_mark="tf-batch-unattached" max_limit=1000000`,
	"queue_type":                      `name="tf-batch-queue-type" kind="pfifo" pfifo_limit=100`,
	"radius":                          `address="192.0.2.2" service="ppp"`,
	"routing_bfd_configuration":       `interfaces="tf-test" addresses="192.0.2.0/24" forbid_bfd=true`,
	"routing_bgp_connection":          `name="tf-remaining-resourcesgp-connection" instance=routeros_routing_bgp_instance.test.name remote_address="192.0.2.2/32" local_role="ibgp" remote_as=64512 connect=false listen=false`,
	"routing_bgp_evpn":                `name="tf-batch-evpn" instance=routeros_routing_bgp_instance.test.name rd="64512:107" vni=107 vrf=routeros_ip_vrf.test.name`,
	"routing_bgp_instance":            `name="tf-remaining-resourcesgp" as=64512 router_id="192.0.2.1"`,
	"routing_bgp_template":            `name="tf-batch-template" as=64512 address_families="ip"`,
	"routing_bgp_vpn":                 `name="tf-batch-vpn" instance=routeros_routing_bgp_instance.test.name route_distinguisher="64512:108" label_allocation_policy="per-vrf" vrf=routeros_ip_vrf.test.name`,
	"routing_filter_rule":             `chain="tf-batch-unattached" rule="reject"`,
	"routing_id":                      `name="tf-batch-id" router_id="192.0.2.1"`,
	"routing_igmp_proxy_interface":    `interface="tf-test" upstream=false`,
	"routing_ospf_area":               `name="tf-reviewed-resourcesrea" instance=routeros_routing_ospf_instance.test.name area_id="0.0.0.107"`,
	"routing_ospf_area_range":         `area=routeros_routing_ospf_area.test.name prefix="192.0.2.0/24"`,
	"routing_ospf_instance":           `name="tf-batch-ospf" version=2 router_id="192.0.2.1"`,
	"routing_ospf_interface_template": `area=routeros_routing_ospf_area.test.name interfaces="tf-test"`,
	"routing_rule":                    `dst_address="192.0.2.0/24" table="main" action="lookup-only-in-table"`,
	"system_logging":                  `action="memory" topics="info" prefix="tf-batch"`,
	"system_scheduler":                `name="tf-batch-scheduler" on_event=":nothing" interval="1m"`,
	"tool_graphing_interface":         `interface="tf-test" allow_address="192.0.2.0/24" store_on_disk=false`,
	"tool_graphing_queue":             `simple_queue=routeros_queue_simple.test.name allow_address="192.0.2.0/24" store_on_disk=false`,
	"tool_graphing_resource":          `allow_address="192.0.2.0/24" store_on_disk=false`,
	"tool_netwatch":                   `name="tf-batch-netwatch" host="192.0.2.2" type="simple" interval="1m"`,
	"ip_upnp_interfaces":              `interface="tf-test" type="internal"`,
	"system_user":                     `name="tf-batch-user" group=routeros_system_user_group.test.name`,
	"system_user_group":               `name="tf-batch-group" policy="read,ssh"`,
	"system_user_sshkeys":             `user=routeros_system_user.test.name`,
}

func extendedResourceLiveConfig(t *testing.T, fixture extendedResourceFixture, changed bool, withContainer bool) string {
	t.Helper()
	var b strings.Builder
	b.WriteString(`provider "routeros" {}
 resource "routeros_ip_pool" "batch_parent" {
 name="tf-batch-pool"
 ranges=["192.0.2.10-192.0.2.20"]
 }
 resource "routeros_ip_dhcp_server" "batch_parent" {
 name="tf-batch-server"
 interface="tf-test"
 address_pool=routeros_ip_pool.batch_parent.name
 disabled=true
 }
 resource "routeros_ip_dhcp_server_option" "batch_parent" {
 name="tf-batch-option"
 code=100
 value="0x01"
 }
 `)
	for _, p := range catalog.ExtendedCollections() {
		if p.RequiredPackage != "" && !withContainer {
			continue
		}
		body, exists := extendedResourceLiveBodies[p.Name]
		if p.Name == "snmp_community" {
			name, _ := json.Marshal("tf-batch-" + fixture.secret)
			body = fmt.Sprintf(`name=%s addresses="192.0.2.0/24" read_access=false write_access=false`, name)
			exists = true
		}
		if p.Name == "system_certificate_scep_server" {
			body = `ca_cert="tf-batch-ca" path="/scep/tf-batch" days_valid=7`
			exists = true
		}
		if !exists {
			t.Fatalf("missing explicit safe case: %s", p.Name)
		}
		fmt.Fprintf(&b, "\nresource %q %q {\n%s\n", "routeros_"+p.Name, "test", strings.ReplaceAll(body, " ", "\n"))
		// Body literals above contain no spaces inside quoted scalar values.
		for _, f := range p.Fields {
			if f.Name == "disabled" {
				b.WriteString("disabled=true\n")
			}
			if f.Name == "comment" && f.Mode != "computed" {
				comment := "tf-batch-first"
				if changed {
					comment = "tf-batch-updated"
				}
				fmt.Fprintf(&b, "comment=%q\n", comment)
			}
		}
		if p.Name == "interface_wireguard_peer" {
			fmt.Fprintf(&b, "public_key=%q\n", fixture.wgKey)
		}
		if p.Name == "system_user_sshkeys" {
			fmt.Fprintf(&b, "key=%q\n", fixture.key)
		}
		if p.Name == "system_user" || p.Name == "interface_dot1x_client" {
			fmt.Fprintf(&b, "password=%q\n", fixture.secret)
		}
		if p.Name == "ip_ipsec_identity" || p.Name == "radius" {
			fmt.Fprintf(&b, "secret=%q\n", fixture.secret)
		}
		b.WriteString("}\n")
	}
	return b.String()
}
func TestAccExtendedCollectionsCHR(t *testing.T) {
	if os.Getenv("TF_ACC") != "1" {
		t.Skip("live CHR requires TF_ACC=1")
	}
	if os.Getenv("ROS_TEST_DISPOSABLE") != "1" || os.Getenv("ROS_HOSTURL") != "http://127.0.0.1:18780" {
		t.Fatal("only explicit disposable CHR allowed")
	}
	c, err := client.New(client.Config{HostURL: os.Getenv("ROS_HOSTURL"), Username: os.Getenv("ROS_USERNAME"), Password: os.Getenv("ROS_PASSWORD"), Timeout: 59 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	ctx := context.Background()
	var system map[string]any
	if err = c.Request(ctx, "GET", "/system/resource", "", nil, nil, &system); err != nil {
		t.Fatal(err)
	}
	if system["version"] != expectedCHRVersion(t) || system["architecture-name"] != "x86_64" {
		t.Fatal("wrong live lane")
	}
	var packages []map[string]any
	if err = c.Request(ctx, "GET", "/system/package", "", nil, nil, &packages); err != nil {
		t.Fatal(err)
	}
	withContainer := false
	for _, p := range packages {
		if p["name"] == "container" && p["disabled"] == "false" {
			withContainer = true
		}
	}
	// Only the owned CA prerequisite is signed. It is never trusted for management
	// TLS or assigned to any service; signing is not arbitrary script execution.
	var cert map[string]any
	if err = c.Request(ctx, "PUT", "/certificate", "", nil, map[string]any{"name": "tf-batch-ca", "common-name": "tf-batch-ca.invalid", "key-usage": "key-cert-sign,crl-sign", "days-valid": "1"}, &cert); err != nil {
		t.Fatal("owned CA prerequisite create failed", err)
	}
	var signed any
	if err = c.Request(ctx, "POST", "/certificate/sign", "", nil, map[string]any{"number": cert[".id"], "name": "tf-batch-ca"}, &signed); err != nil {
		t.Fatal("owned CA prerequisite sign failed", err)
	}
	defer func() {
		if id, ok := cert[".id"].(string); ok {
			_ = c.Request(ctx, "DELETE", "/certificate", id, nil, nil, nil)
		}
	}()
	fixture := newExtendedResourceFixture(t)
	first := extendedResourceLiveConfig(t, fixture, false, withContainer)
	updatedFixture := fixture
	updatedFixture.key = newExtendedResourceFixture(t).key
	changed := extendedResourceLiveConfig(t, updatedFixture, true, withContainer)
	steps := []resource.TestStep{{Config: first}, {Config: first, PlanOnly: true}, {Config: changed}, {Config: changed, PlanOnly: true}}
	for _, p := range catalog.ExtendedCollections() {
		if p.RequiredPackage != "" && !withContainer {
			t.Logf("%s: unavailable on base lane; requires container", p.Name)
			continue
		}
		ignore := []string{}
		for _, f := range p.Fields {
			if f.PreserveSecretOnOmission {
				ignore = append(ignore, f.Name)
			}
		}
		steps = append(steps, resource.TestStep{ResourceName: "routeros_" + p.Name + ".test", ImportState: true, ImportStateVerify: true, ImportStateVerifyIgnore: ignore})
	}
	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){"routeros": providerserver.NewProtocol6WithError(New("test")())}, Steps: steps, CheckDestroy: func(state *terraform.State) error {
		for _, rs := range state.RootModule().Resources {
			if rs.Primary == nil {
				continue
			}
			for _, p := range catalog.ExtendedCollections() {
				if rs.Type == "routeros_"+p.Name {
					var rows []map[string]any
					if err := c.Request(ctx, "GET", p.Path, "", url.Values{".id": {rs.Primary.ID}, ".proplist": {".id"}}, nil, &rows); err != nil {
						return err
					}
					if len(rows) != 0 {
						return fmt.Errorf("owned live object remains: %s", p.Name)
					}
				}
			}
		}
		return nil
	}})
}
