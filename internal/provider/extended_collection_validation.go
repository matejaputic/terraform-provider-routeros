// SPDX-License-Identifier: MPL-2.0
package provider

import (
	"encoding/base64"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/matejaputic/terraform-provider-routeros/internal/catalog"
	"golang.org/x/crypto/ssh"
)

func (r *collectionResource) extendedCollection() bool {
	return strings.HasPrefix(r.policy.Revision, "extended-collections-curated-")
}
func validateReviewedField(f catalog.Field, v attr.Value) error {
	if v == nil || v.IsNull() || v.IsUnknown() {
		return nil
	}
	if x, ok := v.(types.Int64); ok {
		if f.Minimum != nil && x.ValueInt64() < *f.Minimum || f.Maximum != nil && x.ValueInt64() > *f.Maximum {
			return fmt.Errorf("%s is outside reviewed bounds", f.Name)
		}
	}
	x, ok := v.(types.String)
	if !ok {
		return nil
	}
	text := x.ValueString()
	if len(f.Choices) > 0 && !oneOf(text, f.Choices...) {
		return fmt.Errorf("unsupported value for %s", f.Name)
	}
	if f.Codec == "csv-string-set" {
		if text == "" && f.Mode != "required" {
			return nil
		}
		seen := map[string]bool{}
		for _, entry := range strings.Split(text, ",") {
			if entry == "" || strings.TrimSpace(entry) != entry || seen[entry] {
				return fmt.Errorf("%s requires unique nonempty comma-separated entries", f.Name)
			}
			seen[entry] = true
		}
	}
	if text == "" && f.Mode != "required" {
		return nil
	}
	fail := func() error { return fmt.Errorf("invalid reviewed format for %s", f.Name) }
	switch f.Constraint {
	case "user_policy":
		if _, err := userPolicySet(text); err != nil {
			return fail()
		}
	case "url":
		u, err := url.Parse(text)
		if err != nil || !oneOf(u.Scheme, "https", "http") || u.Hostname() == "" || u.Fragment != "" {
			return fail()
		}
	case "ip", "ip4", "ip6":
		ip, err := netip.ParseAddr(text)
		if err != nil || ip.Zone() != "" || ip.String() != text || ip.Is4In6() || f.Constraint == "ip4" && !ip.Is4() || f.Constraint == "ip6" && !ip.Is6() {
			return fail()
		}
	case "prefix", "prefix4", "prefix6":
		p, err := netip.ParsePrefix(text)
		if err != nil || p != p.Masked() || p.String() != text || f.Constraint == "prefix4" && !p.Addr().Is4() || f.Constraint == "prefix6" && !p.Addr().Is6() {
			return fail()
		}
	case "address6":
		p, err := netip.ParsePrefix(text)
		if err != nil || !p.Addr().Is6() || p.Addr().Is4In6() || p.String() != text {
			return fail()
		}
	case "ip6_or_prefix":
		ip, err := netip.ParseAddr(text)
		if err == nil {
			if !ip.Is6() || ip.Is4In6() || ip.Zone() != "" || ip.String() != text {
				return fail()
			}
		} else {
			p, err := netip.ParsePrefix(text)
			if err != nil || !p.Addr().Is6() || p != p.Masked() || p.String() != text {
				return fail()
			}
		}
	case "mac":
		mac, err := net.ParseMAC(text)
		if err != nil || len(mac) != 6 {
			return fail()
		}
	case "vlan_ids":
		if _, err := vlanIDSet(text); err != nil {
			return fail()
		}
	case "rate_pair", "priority_pair":
		parts := strings.Split(text, "/")
		if len(parts) != 2 {
			return fail()
		}
		for _, part := range parts {
			n, err := strconv.ParseInt(part, 10, 64)
			if err != nil || n < 0 || strconv.FormatInt(n, 10) != part || f.Constraint == "priority_pair" && (n < 1 || n > 8) {
				return fail()
			}
		}
	case "duration":
		if _, err := routerDuration(text); err != nil {
			return fail()
		}
	}
	return nil
}
func userPolicySet(text string) (map[string]bool, error) {
	result := map[string]bool{}
	seen := map[string]bool{}
	for _, entry := range strings.Split(text, ",") {
		permission := strings.TrimPrefix(entry, "!")
		if seen[permission] || !oneOf(permission, "local", "telnet", "ssh", "ftp", "reboot", "read", "write", "policy", "test", "winbox", "password", "web", "sniff", "sensitive", "api", "romon", "rest-api") {
			return nil, fmt.Errorf("invalid or contradictory user permission")
		}
		seen[permission] = true
		if !strings.HasPrefix(entry, "!") {
			result[permission] = true
		}
	}
	return result, nil
}
func vlanIDSet(text string) (map[int]bool, error) {
	result := map[int]bool{}
	for _, entry := range strings.Split(text, ",") {
		parts := strings.Split(entry, "-")
		if len(parts) > 2 {
			return nil, fmt.Errorf("invalid VLAN range")
		}
		start, err := strconv.Atoi(parts[0])
		if err != nil || start < 1 || start > 4094 {
			return nil, fmt.Errorf("invalid VLAN ID")
		}
		end := start
		if len(parts) == 2 {
			end, err = strconv.Atoi(parts[1])
			if err != nil || end < start || end > 4094 {
				return nil, fmt.Errorf("invalid VLAN range")
			}
		}
		for id := start; id <= end; id++ {
			if result[id] {
				return nil, fmt.Errorf("overlapping VLAN ranges")
			}
			result[id] = true
		}
	}
	return result, nil
}
func preserveReviewedSpelling(f catalog.Field, old, next attr.Value) attr.Value {
	if f.Constraint == "user_policy" {
		if v, ok := next.(types.String); ok && !v.IsNull() && !v.IsUnknown() {
			if permissions, err := userPolicySet(v.ValueString()); err == nil {
				entries := []string{}
				for p := range permissions {
					entries = append(entries, p)
				}
				sort.Strings(entries)
				next = types.StringValue(strings.Join(entries, ","))
			}
		}
	}
	a, ok := old.(types.String)
	if !ok || a.IsNull() || a.IsUnknown() {
		return next
	}
	b, ok := next.(types.String)
	if !ok || b.IsNull() || b.IsUnknown() {
		return next
	}
	if f.Constraint == "user_policy" {
		left, e := userPolicySet(a.ValueString())
		right, err := userPolicySet(b.ValueString())
		if e == nil && err == nil && len(left) == len(right) {
			same := true
			for permission := range left {
				if !right[permission] {
					same = false
				}
			}
			if same {
				return a
			}
		}
	}
	if f.Codec == "csv-string-set" && equivalentAddressList(a.ValueString(), b.ValueString()) {
		return a
	}
	if f.Codec == "routeros-duration" {
		left, e := routerDuration(a.ValueString())
		right, err := routerDuration(b.ValueString())
		if e == nil && err == nil && left == right {
			return a
		}
	}
	if f.Constraint == "mac" {
		left, e := net.ParseMAC(a.ValueString())
		right, err := net.ParseMAC(b.ValueString())
		if e == nil && err == nil && left.String() == right.String() {
			return a
		}
	}
	if f.Constraint == "vlan_ids" {
		left, e := vlanIDSet(a.ValueString())
		right, err := vlanIDSet(b.ValueString())
		if e == nil && err == nil && len(left) == len(right) {
			equal := true
			for id := range left {
				if !right[id] {
					equal = false
				}
			}
			if equal {
				return a
			}
		}
	}
	return next
}
func (r *collectionResource) validateExtendedResource(values map[string]attr.Value, apply bool) error {
	if !r.extendedCollection() {
		return nil
	}
	text := func(name string) string { v, _ := values[name].(types.String); return v.ValueString() }
	switch r.policy.Name {
	case "system_user":
		if strings.EqualFold(text("name"), "admin") || r.client != nil && r.client.IsAuthenticatedUser(text("name")) {
			return fmt.Errorf("the authentication administrator cannot be managed")
		}
	case "system_user_group":
		if oneOf(text("name"), "full", "read", "write") {
			return fmt.Errorf("built-in user groups cannot be managed")
		}
	case "system_user_sshkeys":
		if strings.EqualFold(text("user"), "admin") || r.client != nil && r.client.IsAuthenticatedUser(text("user")) {
			return fmt.Errorf("the authentication administrator's SSH keys cannot be managed")
		}
		key, ok := values["key"].(types.String)
		if ok && !key.IsNull() && !key.IsUnknown() {
			if _, err := reviewedSSHKey(key.ValueString()); err != nil {
				return err
			}
		}
	case "interface_wireguard_peer":
		key, ok := values["public_key"].(types.String)
		if ok && !key.IsNull() && !key.IsUnknown() {
			bytes, err := base64.StdEncoding.DecodeString(key.ValueString())
			if err != nil || len(bytes) != 32 {
				return fmt.Errorf("invalid WireGuard public key")
			}
		}
	case "ipv6_pool":
		p, err := netip.ParsePrefix(text("prefix"))
		length, ok := values["prefix_length"].(types.Int64)
		if err == nil && ok && !length.IsNull() && !length.IsUnknown() && length.ValueInt64() < int64(p.Bits()) {
			return fmt.Errorf("prefix_length cannot be shorter than the pool prefix")
		}
	}
	return nil
}
func reviewedSSHKey(text string) (ssh.PublicKey, error) {
	fields := strings.Fields(text)
	if len(fields) < 2 || !oneOf(fields[0], "ssh-ed25519", "ssh-rsa", "ecdsa-sha2-nistp256") {
		return nil, fmt.Errorf("unsupported SSH public key format")
	}
	key, _, options, rest, err := ssh.ParseAuthorizedKey([]byte(text))
	if err != nil || len(options) != 0 || len(strings.TrimSpace(string(rest))) != 0 {
		return nil, fmt.Errorf("invalid SSH public key")
	}
	return key, nil
}
func (r *collectionResource) guardExtendedResourceRead(old map[string]attr.Value, row map[string]any) error {
	if !r.extendedCollection() {
		return nil
	}
	if r.policy.Name == "tool_netwatch" {
		for _, field := range []string{"up-script", "down-script", "test-script"} {
			if raw, present := row[field]; present {
				value, ok := raw.(string)
				if !ok || value != "" {
					return fmt.Errorf("Netwatch with unmanaged callbacks cannot be managed")
				}
			}
		}
	}
	if r.policy.Name == "system_user_sshkeys" {
		fingerprint, ok := row["fingerprint"].(string)
		if !ok || !strings.HasPrefix(fingerprint, "SHA256:") {
			return fmt.Errorf("missing or unsupported SSH key fingerprint")
		}
		hash, err := base64.RawStdEncoding.DecodeString(strings.TrimRight(strings.TrimPrefix(fingerprint, "SHA256:"), "="))
		if err != nil || len(hash) != 32 {
			return fmt.Errorf("invalid SSH key fingerprint")
		}
		if previous, ok := old["key"].(types.String); ok && !previous.IsNull() && !previous.IsUnknown() {
			key, err := reviewedSSHKey(previous.ValueString())
			if err != nil || ssh.FingerprintSHA256(key) != strings.TrimRight(fingerprint, "=") {
				return fmt.Errorf("SSH key fingerprint changed; refusing adoption of an unrelated key")
			}
		}
	}
	return nil
}
