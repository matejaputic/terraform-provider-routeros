// SPDX-License-Identifier: MPL-2.0
package provider

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/matejaputic/terraform-provider-routeros/internal/catalog"
	"github.com/matejaputic/terraform-provider-routeros/internal/client"
)

// Maintained collection lifecycle. Only explicitly registered policy/schema pairs
// can instantiate this implementation; API inventory never registers resources.
type collectionResource struct {
	policy catalog.Collection
	schema schema.Schema
	client *client.Client
}

var _ resource.ResourceWithImportState = (*collectionResource)(nil)
var _ resource.ResourceWithValidateConfig = (*collectionResource)(nil)

func newCollection(policy catalog.Collection, s schema.Schema) resource.Resource {
	for _, f := range policy.Fields {
		if f.ForceNew {
			a := s.Attributes[f.Name].(schema.StringAttribute)
			a.PlanModifiers = []planmodifier.String{stringplanmodifier.RequiresReplace()}
			s.Attributes[f.Name] = a
		}
	}
	return &collectionResource{policy: policy, schema: s}
}
func (r *collectionResource) Metadata(_ context.Context, q resource.MetadataRequest, s *resource.MetadataResponse) {
	s.TypeName = q.ProviderTypeName + "_" + r.policy.Name
}
func (r *collectionResource) Schema(_ context.Context, _ resource.SchemaRequest, s *resource.SchemaResponse) {
	s.Schema = r.schema
}
func (r *collectionResource) Configure(_ context.Context, q resource.ConfigureRequest, s *resource.ConfigureResponse) {
	if q.ProviderData == nil {
		return
	}
	c, ok := q.ProviderData.(*client.Client)
	if !ok || c == nil {
		s.Diagnostics.AddError("Invalid provider data", "Expected configured RouterOS REST client.")
		return
	}
	r.client = c
}
func (r *collectionResource) attributeTypes() map[string]attr.Type {
	result := map[string]attr.Type{}
	for _, f := range r.policy.Fields {
		switch f.Type {
		case "string":
			result[f.Name] = types.StringType
		case "boolean":
			result[f.Name] = types.BoolType
		case "integer":
			result[f.Name] = types.Int64Type
		case "array":
			result[f.Name] = types.ListType{ElemType: types.StringType}
		}
	}
	return result
}
func (r *collectionResource) object(values map[string]attr.Value) types.Object {
	return types.ObjectValueMust(r.attributeTypes(), values)
}
func nullField(f catalog.Field) attr.Value {
	switch f.Type {
	case "string":
		return types.StringNull()
	case "boolean":
		return types.BoolNull()
	case "integer":
		return types.Int64Null()
	case "array":
		return types.ListNull(types.StringType)
	}
	panic("unsupported reviewed type")
}
func (r *collectionResource) ValidateConfig(ctx context.Context, q resource.ValidateConfigRequest, s *resource.ValidateConfigResponse) {
	var m types.Object
	s.Diagnostics.Append(q.Config.Get(ctx, &m)...)
	if s.Diagnostics.HasError() || m.IsUnknown() {
		return
	}
	if e := r.validate(ctx, m.Attributes(), false); e != nil {
		s.Diagnostics.AddError("Invalid resource configuration", e.Error())
		return
	}
	if e := r.validateFirewallIntent(m.Attributes()); e != nil {
		s.Diagnostics.AddError("Invalid firewall configuration", e.Error())
	}
}
func (r *collectionResource) validate(ctx context.Context, values map[string]attr.Value, apply bool) error {
	if r.policy.Name == "ip_dns_record" {
		if err := validateDNSAddressRecord(values); err != nil {
			return err
		}
	}
	for _, f := range r.policy.Fields {
		v := values[f.Name]
		if v == nil {
			return fmt.Errorf("missing reviewed attribute %s", f.Name)
		}
		if v.IsUnknown() {
			if apply && f.Mode != "computed" {
				return fmt.Errorf("attribute %s must be known for mutation", f.Name)
			}
			continue
		}
		if v.IsNull() {
			if !apply && f.Sensitive && f.PreserveSecretOnOmission {
				continue
			}
			if f.Mode == "required" {
				return fmt.Errorf("attribute %s is required", f.Name)
			}
			continue
		}
		if f.Mode == "computed" {
			continue
		}
		if e := validateReviewedField(f, v); e != nil {
			return e
		}
		switch x := v.(type) {
		case types.String:
			text := x.ValueString()
			if f.Mode == "required" && strings.TrimSpace(text) == "" {
				return fmt.Errorf("attribute %s must be nonempty", f.Name)
			}
			if f.Name != "comment" && strings.ContainsAny(text, "\r\n\x00") {
				return fmt.Errorf("invalid control character in %s", f.Name)
			}
			if r.policy.Name == "ip_firewall_addr_list" && f.Name == "address" {
				if _, e := firewallAddressBounds(text); e != nil {
					return e
				}
			}
			if r.orderedFirewall() && f.Name == "action" && !reviewedFirewallAction(r.policy.Name, text) {
				return fmt.Errorf("unsupported action for reviewed firewall family")
			}
			if r.orderedFirewall() && f.Name == "place_before" && text != "" && !itemID.MatchString(text) {
				return fmt.Errorf("place_before requires *HEX or empty string")
			}
			if r.policy.Name == "ip_dhcp_relay" {
				if err := validateDHCPRelayString(f.Name, text); err != nil {
					return err
				}
			}
			if f.Name == "protocol_mode" && !oneOf(text, "none", "stp", "rstp", "mstp") {
				return fmt.Errorf("protocol_mode must be none, stp, rstp or mstp")
			}
			if (r.policy.Name == "ip_dhcp_server_network" && f.Name == "address") || (r.policy.Name == "ip_route" && f.Name == "dst_address") {
				p, e := netip.ParsePrefix(text)
				if e != nil || !p.Addr().Is4() || p != p.Masked() {
					return fmt.Errorf("%s requires a canonical IPv4 CIDR", f.Name)
				}
			}
			if (r.policy.Name == "ip_dhcp_server_network" && f.Name == "gateway") || (r.policy.Name == "ip_dhcp_server_lease" && f.Name == "address") {
				p, e := netip.ParseAddr(text)
				if e != nil || !p.Is4() {
					return fmt.Errorf("%s requires an IPv4 address", f.Name)
				}
			}
			if r.policy.Name == "ip_dhcp_server_lease" && f.Name == "mac_address" {
				mac, e := net.ParseMAC(text)
				if e != nil || len(mac) != 6 {
					return fmt.Errorf("mac_address requires a 48-bit Ethernet MAC")
				}
			}
			if f.Name == "edge" && !oneOf(text, "auto", "yes", "no", "yes-discover", "no-discover") {
				return fmt.Errorf("invalid bridge edge mode")
			}
		case types.Int64:
			n := x.ValueInt64()
			if r.policy.Name == "interface_bridge" && f.Name == "pvid" {
				enabled := values["vlan_filtering"].(types.Bool)
				if enabled.IsNull() || !enabled.IsUnknown() && !enabled.ValueBool() {
					return fmt.Errorf("bridge pvid requires explicit vlan_filtering = true; RouterOS hides it otherwise")
				}
			}
			if (f.Name == "vlan_id" || f.Name == "pvid") && (n < 1 || n > 4094) {
				return fmt.Errorf("%s must be 1 to 4094", f.Name)
			}
			if f.Name == "distance" && (n < 1 || n > 255) {
				return fmt.Errorf("distance must be 1 to 255")
			}
			if (f.Name == "scope" || f.Name == "target_scope") && (n < 0 || n > 255) {
				return fmt.Errorf("scope must be 0 to 255")
			}
			if f.Name == "netmask" && (n < 0 || n > 32) {
				return fmt.Errorf("netmask must be 0 to 32")
			}
			if (r.policy.Name == "ip_dhcp_client_option" || r.policy.Name == "ip_dhcp_server_option") && f.Name == "code" && (n < 1 || n > 254) {
				return fmt.Errorf("DHCP option code must be 1 to 254")
			}
			if f.Name == "mtu" && (n < 68 || n > 65535) {
				return fmt.Errorf("mtu must be 68 to 65535")
			}
		case types.List:
			if !apply {
				unknown := false
				for _, e := range x.Elements() {
					if e.IsUnknown() {
						unknown = true
					}
				}
				if unknown {
					continue
				}
			}
			var entries []string
			if d := x.ElementsAs(ctx, &entries, false); d.HasError() {
				return fmt.Errorf("ranges must contain known non-null strings")
			}
			if e := validateCSV(f, entries); e != nil {
				return e
			}
			if r.policy.Name == "ip_dhcp_server_network" && f.Name == "dns_server" && len(entries) > 0 {
				flag := values["dns_none"].(types.Bool)
				if !flag.IsNull() && !flag.IsUnknown() && flag.ValueBool() {
					return fmt.Errorf("dns_none = true cannot be combined with nonempty dns_server; RouterOS clears the servers")
				}
			}
		}
	}
	if e := r.validateBatchAResource(values, apply); e != nil {
		return e
	}
	return r.validateFirewallFields(values, apply)
}
func validateCSV(f catalog.Field, entries []string) error {
	if f.Name == "ranges" {
		return validateRanges(entries)
	}
	if f.Name != "dns_server" {
		return fmt.Errorf("unimplemented CSV field")
	}
	seen := map[netip.Addr]bool{}
	for _, entry := range entries {
		a, e := netip.ParseAddr(entry)
		if e != nil || !a.Is4() || seen[a] {
			return fmt.Errorf("DNS servers must be distinct IPv4 addresses")
		}
		seen[a] = true
	}
	return nil
}
func oneOf(value string, values ...string) bool {
	for _, v := range values {
		if value == v {
			return true
		}
	}
	return false
}

type ipRange struct{ start, end netip.Addr }

func parseRange(raw string) (ipRange, error) {
	parts := strings.Split(raw, "-")
	if len(parts) > 2 || strings.TrimSpace(raw) != raw {
		return ipRange{}, fmt.Errorf("invalid IPv4 pool range")
	}
	a, e := netip.ParseAddr(parts[0])
	if e != nil || !a.Is4() {
		return ipRange{}, fmt.Errorf("pool ranges require IPv4 addresses")
	}
	b := a
	if len(parts) == 2 {
		b, e = netip.ParseAddr(parts[1])
		if e != nil || !b.Is4() || a.Compare(b) > 0 {
			return ipRange{}, fmt.Errorf("invalid IPv4 pool range bounds")
		}
	}
	return ipRange{a, b}, nil
}
func validateRanges(entries []string) error {
	if len(entries) == 0 {
		return fmt.Errorf("ranges must contain at least one IPv4 address/range")
	}
	ranges := []ipRange{}
	for _, entry := range entries {
		next, e := parseRange(entry)
		if e != nil {
			return e
		}
		for _, old := range ranges {
			if next.start.Compare(old.end) <= 0 && old.start.Compare(next.end) <= 0 {
				return fmt.Errorf("pool ranges must not overlap")
			}
		}
		ranges = append(ranges, next)
	}
	return nil
}
func (r *collectionResource) payload(ctx context.Context, values map[string]attr.Value) (map[string]any, error) {
	if e := r.validate(ctx, values, true); e != nil {
		return nil, e
	}
	if e := r.validateFirewallIntent(values); e != nil {
		return nil, e
	}
	result := map[string]any{}
	for _, f := range r.policy.Fields {
		v := values[f.Name]
		if f.Mode == "computed" || f.Codec == "placement" || v.IsNull() || v.IsUnknown() {
			continue
		}
		switch x := v.(type) {
		case types.String:
			result[f.Wire] = x.ValueString()
			if r.policy.Name == "ip_dhcp_server_lease" && f.Name == "mac_address" {
				mac, _ := net.ParseMAC(x.ValueString())
				result[f.Wire] = strings.ToUpper(mac.String())
			}
		case types.Bool:
			if x.ValueBool() {
				result[f.Wire] = "yes"
			} else {
				result[f.Wire] = "no"
			}
		case types.Int64:
			result[f.Wire] = strconv.FormatInt(x.ValueInt64(), 10)
		case types.List:
			var entries []string
			if d := x.ElementsAs(ctx, &entries, false); d.HasError() {
				return nil, fmt.Errorf("invalid list mutation")
			}
			result[f.Wire] = strings.Join(entries, ",")
		}
	}
	return result, nil
}
func strictWireBool(raw any) (types.Bool, error) {
	if raw == nil {
		return types.BoolNull(), fmt.Errorf("null RouterOS boolean")
	}
	if text, ok := raw.(string); ok && !oneOf(text, "true", "false", "yes", "no") {
		return types.BoolNull(), fmt.Errorf("invalid RouterOS boolean token")
	}
	return wireBool(raw)
}
func decodeField(ctx context.Context, f catalog.Field, raw any) (attr.Value, error) {
	if f.Codec == "auto-decimal" {
		if x, ok := raw.(string); ok && oneOf(x, "auto", "unspecified") {
			return types.Int64Null(), nil
		}
	}
	switch f.Type {
	case "string":
		x, ok := raw.(string)
		if !ok {
			return nil, fmt.Errorf("invalid string field %s", f.Name)
		}
		return types.StringValue(x), nil
	case "boolean":
		return strictWireBool(raw)
	case "integer":
		var n int64
		switch x := raw.(type) {
		case string:
			v, e := strconv.ParseInt(x, 10, 64)
			if e != nil {
				return nil, fmt.Errorf("invalid decimal field %s", f.Name)
			}
			n = v
		case float64:
			if math.IsNaN(x) || math.IsInf(x, 0) || x != math.Trunc(x) || x >= float64(math.MaxInt64) || x < float64(math.MinInt64) {
				return nil, fmt.Errorf("invalid integer field %s", f.Name)
			}
			n = int64(x)
		default:
			return nil, fmt.Errorf("invalid integer type %s", f.Name)
		}
		return types.Int64Value(n), nil
	case "array":
		x, ok := raw.(string)
		if !ok {
			return nil, fmt.Errorf("invalid CSV field %s", f.Name)
		}
		entries := strings.Split(x, ",")
		for i := range entries {
			entries[i] = strings.TrimSpace(entries[i])
		}
		if x == "" && f.Name == "dns_server" {
			entries = []string{}
		}
		if e := validateCSV(f, entries); e != nil {
			return nil, e
		}
		v, d := types.ListValueFrom(ctx, types.StringType, entries)
		if d.HasError() {
			return nil, fmt.Errorf("invalid list read")
		}
		return v, nil
	}
	return nil, fmt.Errorf("unsupported reviewed field")
}

// Preserve configured list order/spelling only for equivalent numeric ranges.
// RouterOS may sort ranges or serialize a single IP as a degenerate range.
func preserveRanges(old, observed types.List) types.List {
	if old.IsNull() || old.IsUnknown() || len(old.Elements()) != len(observed.Elements()) {
		return observed
	}
	seen := map[ipRange]int{}
	for _, v := range old.Elements() {
		s, ok := v.(types.String)
		if !ok || s.IsNull() || s.IsUnknown() {
			return observed
		}
		p, e := parseRange(s.ValueString())
		if e != nil {
			return observed
		}
		seen[p]++
	}
	for _, v := range observed.Elements() {
		p, e := parseRange(v.(types.String).ValueString())
		if e != nil || seen[p] == 0 {
			return observed
		}
		seen[p]--
	}
	return old
}
func (r *collectionResource) refresh(ctx context.Context, values map[string]attr.Value) (map[string]attr.Value, bool, error) {
	id, ok := values["id"].(types.String)
	if !ok || !validID(id) {
		return nil, false, fmt.Errorf("invalid resource ID")
	}
	var rows []map[string]any
	e := r.client.Request(ctx, http.MethodGet, r.policy.Path, "", url.Values{".id": {id.ValueString()}}, nil, &rows)
	if e != nil {
		var se *client.StatusError
		if errors.As(e, &se) && se.Code == 404 {
			return nil, false, nil
		}
		return nil, false, e
	}
	if len(rows) == 0 {
		return nil, false, nil
	}
	if len(rows) != 1 || wireString(rows[0][".id"]) != id.ValueString() {
		return nil, false, fmt.Errorf("read did not return exactly the requested ID")
	}
	// Importing built-in/dynamic collections could otherwise delete hardware or
	// system-owned records through ordinary CRUD. These objects are not managed.
	ownershipFlags := []string{"builtin", "dynamic"}
	if r.batchACollection() {
		ownershipFlags = append(ownershipFlags, "default")
	}
	if r.policy.Name == "ip_dhcp_client_option" {
		// RouterOS marks its default DHCP client options with this distinct flag.
		ownershipFlags = append(ownershipFlags, "default")
	}
	for _, key := range ownershipFlags {
		if v, exists := rows[0][key]; exists {
			b, e := strictWireBool(v)
			if e != nil {
				return nil, false, e
			}
			if b.ValueBool() {
				return nil, false, fmt.Errorf("system-owned or dynamic objects are not manageable")
			}
		}
	}
	if r.policy.Name == "ip_dns_record" {
		// Refuse imported regex/firewall-list actions absent from this contract.
		for _, key := range []string{"regexp", "address-list"} {
			if raw, present := rows[0][key]; present {
				text, ok := raw.(string)
				if !ok || text != "" {
					return nil, false, fmt.Errorf("DNS record has an unmanaged action or matcher")
				}
			}
		}
	}
	if r.policy.Name == "ip_route" || r.policy.Name == "ipv6_route" {
		flag, e := strictWireBool(rows[0]["static"])
		if e != nil || !flag.ValueBool() {
			return nil, false, fmt.Errorf("only explicitly static routes are manageable")
		}
	}
	if e := r.guardBatchARead(values, rows[0]); e != nil {
		return nil, false, e
	}
	result := map[string]attr.Value{}
	for _, f := range r.policy.Fields {
		if f.Name == "id" {
			result[f.Name] = id
			continue
		}
		if r.orderedFirewall() && f.Name == "place_before" {
			chain, ok := rows[0]["chain"].(string)
			if !ok {
				return nil, false, fmt.Errorf("missing filter chain")
			}
			next, e := r.nextFilterRule(ctx, id.ValueString(), chain)
			if e != nil {
				return nil, false, e
			}
			result[f.Name] = types.StringValue(next)
			continue
		}
		raw, exists := rows[0][f.Wire]
		if !exists {
			if f.Sensitive && f.PreserveSecretOnOmission {
				previous := values[f.Name]
				if previous == nil || previous.IsUnknown() {
					previous = nullField(f)
				}
				result[f.Name] = previous
				continue
			}
			if f.ReadDefault != nil {
				v, e := decodeField(ctx, f, *f.ReadDefault)
				if e != nil {
					return nil, false, e
				}
				result[f.Name] = v
				continue
			}
			if f.ConditionalRead == "vlan_filtering" {
				if flag, ok := rows[0]["vlan-filtering"]; ok {
					b, e := strictWireBool(flag)
					if e != nil {
						return nil, false, e
					}
					if !b.ValueBool() {
						result[f.Name] = nullField(f)
						continue
					}
				}
			}
			old := values[f.Name]
			if f.Mode == "required" || f.Mode != "computed" && old != nil && !old.IsNull() && !old.IsUnknown() {
				return nil, false, fmt.Errorf("missing configured wire field %s", f.Name)
			}
			result[f.Name] = nullField(f)
			continue
		}
		v, e := decodeField(ctx, f, raw)
		if e != nil {
			return nil, false, e
		}
		if f.Sensitive && f.PreserveSecretOnOmission && (wireString(raw) == "*****" || wireString(raw) == "**hidden**") {
			v = values[f.Name]
			if v == nil || v.IsUnknown() {
				v = nullField(f)
			}
		}
		v = preserveReviewedSpelling(f, values[f.Name], v)
		if r.policy.Name == "ip_firewall_addr_list" && f.Name == "address" {
			if old, ok := values[f.Name].(types.String); ok && !old.IsNull() && !old.IsUnknown() {
				a, e := firewallAddressBounds(old.ValueString())
				b, err := firewallAddressBounds(v.(types.String).ValueString())
				if e == nil && err == nil && a == b {
					v = old
				}
			}
		}
		if r.policy.Name == "ip_dhcp_server_lease" && f.Name == "mac_address" {
			if old, ok := values[f.Name].(types.String); ok && !old.IsNull() && !old.IsUnknown() {
				a, e := net.ParseMAC(old.ValueString())
				b, err := net.ParseMAC(v.(types.String).ValueString())
				if e == nil && err == nil && a.String() == b.String() {
					v = old
				}
			}
		}
		if f.Type == "array" && f.Name == "ranges" {
			if previous, ok := values[f.Name].(types.List); ok {
				v = preserveRanges(previous, v.(types.List))
			}
		}
		result[f.Name] = v
	}
	if e := r.validate(ctx, result, false); e != nil {
		return nil, false, fmt.Errorf("invalid RouterOS resource response: %w", e)
	}
	return result, true, nil
}
func (r *collectionResource) ready(d *diag.Diagnostics) bool {
	if r.client != nil && r.policy.RequiredPackage != "" && !oneOf(r.policy.RequiredPackage, r.client.Packages...) {
		d.AddError("Required RouterOS package unavailable", "This reviewed resource requires the enabled matching-version "+r.policy.RequiredPackage+" package.")
		return false
	}
	if r.client == nil {
		d.AddError("Unconfigured resource", "RouterOS client is unavailable.")
		return false
	}
	return true
}
func (r *collectionResource) Create(ctx context.Context, q resource.CreateRequest, s *resource.CreateResponse) {
	if !r.ready(&s.Diagnostics) {
		return
	}
	var plan, config types.Object
	s.Diagnostics.Append(q.Plan.Get(ctx, &plan)...)
	s.Diagnostics.Append(q.Config.Get(ctx, &config)...)
	if s.Diagnostics.HasError() {
		return
	}
	if plan.IsNull() || plan.IsUnknown() {
		s.Diagnostics.AddError("Invalid create plan", "Expected a known object plan.")
		return
	}
	body, e := r.payload(ctx, config.Attributes())
	if e != nil {
		s.Diagnostics.AddError("Invalid resource configuration", e.Error())
		return
	}
	if e := r.positionFilter(ctx, "", config.Attributes(), false); e != nil {
		s.Diagnostics.AddError("Invalid filter ordering", e.Error())
		return
	}
	var created map[string]any
	if e = r.client.Request(ctx, http.MethodPut, r.policy.Path, "", nil, body, &created); e != nil {
		s.Diagnostics.AddError("RouterOS create failed", e.Error())
		return
	}
	id, ok := created[".id"].(string)
	if !ok || !itemID.MatchString(id) {
		s.Diagnostics.AddError("Invalid create response", "RouterOS omitted its internal ID.")
		return
	}
	values := plan.Attributes()
	values["id"] = types.StringValue(id)
	partial := map[string]attr.Value{}
	for _, f := range r.policy.Fields {
		v := values[f.Name]
		if v.IsUnknown() {
			v = nullField(f)
		}
		partial[f.Name] = v
	}
	s.Diagnostics.Append(s.State.Set(ctx, r.object(partial))...)
	if e := r.positionFilter(ctx, id, config.Attributes(), true); e != nil {
		s.Diagnostics.AddError("RouterOS create ordering failed", e.Error())
		return
	}
	next, found, e := r.refresh(ctx, values)
	if e != nil || !found {
		if e == nil {
			e = fmt.Errorf("created object disappeared")
		}
		s.Diagnostics.AddError("RouterOS create refresh failed", e.Error())
		return
	}
	s.Diagnostics.Append(s.State.Set(ctx, r.object(next))...)
}
func (r *collectionResource) Read(ctx context.Context, q resource.ReadRequest, s *resource.ReadResponse) {
	if !r.ready(&s.Diagnostics) {
		return
	}
	var old types.Object
	s.Diagnostics.Append(q.State.Get(ctx, &old)...)
	if s.Diagnostics.HasError() {
		return
	}
	next, found, e := r.refresh(ctx, old.Attributes())
	if e != nil {
		s.Diagnostics.AddError("RouterOS read failed", e.Error())
		return
	}
	if !found {
		s.State.RemoveResource(ctx)
		return
	}
	s.Diagnostics.Append(s.State.Set(ctx, r.object(next))...)
}
func (r *collectionResource) Update(ctx context.Context, q resource.UpdateRequest, s *resource.UpdateResponse) {
	if r.policy.ReplacementOnly {
		s.Diagnostics.AddError("Replacement required", "This resource has no supported item PATCH; change requires replacement.")
		return
	}
	if !r.ready(&s.Diagnostics) {
		return
	}
	var plan, config, old types.Object
	s.Diagnostics.Append(q.Plan.Get(ctx, &plan)...)
	s.Diagnostics.Append(q.Config.Get(ctx, &config)...)
	s.Diagnostics.Append(q.State.Get(ctx, &old)...)
	if s.Diagnostics.HasError() {
		return
	}
	values := plan.Attributes()
	id, ok := old.Attributes()["id"].(types.String)
	if !ok || !validID(id) || values == nil {
		s.Diagnostics.AddError("Invalid resource ID", "Expected RouterOS .id.")
		return
	}
	values["id"] = id
	body, e := r.payload(ctx, config.Attributes())
	if e != nil {
		s.Diagnostics.AddError("Invalid resource configuration", e.Error())
		return
	}
	// Recheck ownership even when Terraform runs with refresh disabled.
	if _, found, err := r.refresh(ctx, old.Attributes()); err != nil || !found {
		if err == nil {
			err = fmt.Errorf("object disappeared; refresh and replan")
		}
		s.Diagnostics.AddError("RouterOS update ownership check failed", err.Error())
		return
	}
	if e := r.positionFilter(ctx, id.ValueString(), config.Attributes(), false); e != nil {
		s.Diagnostics.AddError("Invalid filter ordering", e.Error())
		return
	}
	// Replacement-owned fields never need PATCH, including a future observed
	// create-only name contract. Keep unchanged names out of update payloads.
	for _, field := range r.policy.Fields {
		if field.ForceNew {
			delete(body, field.Wire)
		}
	}
	if e = r.client.Request(ctx, http.MethodPatch, r.policy.Path, id.ValueString(), nil, body, nil); e != nil {
		s.Diagnostics.AddError("RouterOS update failed", e.Error())
		return
	}
	if e := r.positionFilter(ctx, id.ValueString(), config.Attributes(), true); e != nil {
		s.Diagnostics.AddError("RouterOS update ordering failed", e.Error())
		return
	}
	next, found, e := r.refresh(ctx, values)
	if e != nil || !found {
		if e == nil {
			e = fmt.Errorf("updated object disappeared")
		}
		s.Diagnostics.AddError("RouterOS update refresh failed", e.Error())
		return
	}
	s.Diagnostics.Append(s.State.Set(ctx, r.object(next))...)
}
func (r *collectionResource) Delete(ctx context.Context, q resource.DeleteRequest, s *resource.DeleteResponse) {
	if !r.ready(&s.Diagnostics) {
		return
	}
	var old types.Object
	s.Diagnostics.Append(q.State.Get(ctx, &old)...)
	if s.Diagnostics.HasError() {
		return
	}
	id, ok := old.Attributes()["id"].(types.String)
	if !ok || !validID(id) {
		s.Diagnostics.AddError("Invalid resource ID", "Expected RouterOS .id.")
		return
	}
	// Recheck ownership before deleting an imported object. Absence is idempotent;
	// malformed/auth failures must never authorize deletion.
	_, found, e := r.refresh(ctx, old.Attributes())
	if e != nil {
		s.Diagnostics.AddError("RouterOS delete ownership check failed", e.Error())
		return
	}
	if !found {
		return
	}
	e = r.client.Request(ctx, http.MethodDelete, r.policy.Path, id.ValueString(), nil, nil, nil)
	var se *client.StatusError
	if e != nil && !(errors.As(e, &se) && se.Code == 404) {
		s.Diagnostics.AddError("RouterOS delete failed", e.Error())
	}
}
func (r *collectionResource) ImportState(ctx context.Context, q resource.ImportStateRequest, s *resource.ImportStateResponse) {
	if !itemID.MatchString(q.ID) {
		s.Diagnostics.AddError("Invalid import ID", "Use RouterOS .id (* followed by hexadecimal digits).")
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), q, s)
}
