// SPDX-License-Identifier: MPL-2.0
package provider

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/matejaputic/terraform-provider-routeros/internal/catalog"
	"github.com/matejaputic/terraform-provider-routeros/internal/client"
)

func TestAccAdditionalSettingsCHR(t *testing.T) {
	if os.Getenv("TF_ACC") != "1" {
		t.Skip("live acceptance requires TF_ACC=1")
	}
	if os.Getenv("ROS_TEST_DISPOSABLE") != "1" || os.Getenv("ROS_HOSTURL") != "http://127.0.0.1:18780" {
		t.Fatal("explicit owned disposable CHR only")
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
		t.Fatal("unexpected CHR lane")
	}
	var packages []map[string]any
	if err := c.Request(context.Background(), "GET", "/system/package", "", nil, nil, &packages); err != nil {
		t.Fatal(err)
	}
	enabled := map[string]bool{}
	for _, p := range packages {
		if fmt.Sprint(p["disabled"]) == "false" {
			enabled[fmt.Sprint(p["name"])] = true
		}
	}
	policies := []catalog.Collection{}
	for _, p := range catalog.AdditionalSettings() {
		if p.Name == "system_led_settings" {
			t.Log("unavailable: physical LED controls; CHR reports feature unsupported on board")
			continue
		}
		if p.RequiredPackage != "" && !enabled[p.RequiredPackage] {
			t.Logf("unavailable on this target: %s requires %s", p.Name, p.RequiredPackage)
			continue
		}
		policies = append(policies, p)
	}
	if len(policies) == 0 {
		t.Fatal("no available reviewed settings")
	}
	t.Logf("available settings constructors: %d", len(policies))
	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){"routeros": providerserver.NewProtocol6WithError(New("test")())}, Steps: additionalSettingsSteps(t, c, policies, `provider "routeros" {}`), CheckDestroy: additionalSettingsDestroyRetains(c, policies)})
}
