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

func TestAccDNSRecordCHR(t *testing.T) {
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
	config := func(kind, address, comment string) string {
		return fmt.Sprintf(`provider "routeros" {}
resource "routeros_ip_dns_record" "test" {
 name = "tf-coverage-record.invalid"
 type = %q
 address = %q
 comment = %q
 disabled = true
 match_subdomain = %t
}`, kind, address, comment, kind == "A" && comment == "")
	}
	first := config("A", "192.0.2.1", "first")
	changed := config("A", "192.0.2.2", "")
	replaced := config("AAAA", "2001:db8::1", "")
	recordID := func() string {
		t.Helper()
		var rows []map[string]any
		if err := c.Request(context.Background(), "GET", "/ip/dns/static", "", url.Values{"name": {"tf-coverage-record.invalid"}}, nil, &rows); err != nil {
			t.Fatal(err)
		}
		if len(rows) != 1 {
			t.Fatal("expected one owned DNS record")
		}
		id, ok := rows[0][".id"].(string)
		if !ok || !itemID.MatchString(id) {
			t.Fatal("invalid DNS record ID")
		}
		return id
	}
	var oldID, deletedID string
	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){"routeros": providerserver.NewProtocol6WithError(New("test")())}, Steps: []resource.TestStep{
		{Config: first, Check: resource.TestCheckResourceAttr("routeros_ip_dns_record.test", "address", "192.0.2.1")},
		{Config: first, PlanOnly: true},
		{ResourceName: "routeros_ip_dns_record.test", ImportState: true, ImportStateVerify: true},
		{Config: changed, Check: resource.ComposeTestCheckFunc(resource.TestCheckResourceAttr("routeros_ip_dns_record.test", "address", "192.0.2.2"), resource.TestCheckResourceAttr("routeros_ip_dns_record.test", "comment", ""))},
		{Config: changed, PlanOnly: true},
		{Config: changed, PreConfig: func() {
			if err := c.Request(context.Background(), "PATCH", "/ip/dns/static", recordID(), nil, map[string]string{"address": "192.0.2.99"}, nil); err != nil {
				t.Fatal(err)
			}
		}, Check: resource.TestCheckResourceAttr("routeros_ip_dns_record.test", "address", "192.0.2.2")},
		{Config: changed, PreConfig: func() {
			deletedID = recordID()
			if err := c.Request(context.Background(), "DELETE", "/ip/dns/static", deletedID, nil, nil, nil); err != nil {
				t.Fatal(err)
			}
		}, Check: func(s *terraform.State) error {
			if s.RootModule().Resources["routeros_ip_dns_record.test"].Primary.ID == deletedID {
				return fmt.Errorf("deleted DNS record not recreated")
			}
			return nil
		}},
		{Config: replaced, PreConfig: func() { oldID = recordID() }, Check: func(s *terraform.State) error {
			if s.RootModule().Resources["routeros_ip_dns_record.test"].Primary.ID == oldID {
				return fmt.Errorf("type change did not replace")
			}
			return nil
		}},
		{Config: replaced, PlanOnly: true},
		{ResourceName: "routeros_ip_dns_record.test", ImportState: true, ImportStateVerify: true},
	}, CheckDestroy: func(*terraform.State) error {
		var rows []map[string]any
		if err := c.Request(context.Background(), "GET", "/ip/dns/static", "", url.Values{"name": {"tf-coverage-record.invalid"}}, nil, &rows); err != nil {
			return err
		}
		if len(rows) != 0 {
			return fmt.Errorf("DNS record remains")
		}
		return nil
	}})
}
