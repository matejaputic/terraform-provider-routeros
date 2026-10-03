// SPDX-License-Identifier: MPL-2.0
package provider

import (
	"context"
	"errors"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/matejaputic/terraform-provider-routeros/internal/catalog"
	"github.com/matejaputic/terraform-provider-routeros/internal/client"
	"github.com/matejaputic/terraform-provider-routeros/internal/generated"
	"net"
	"net/http"
	"net/url"
	"regexp"
)

var _ resource.ResourceWithConfigure = (*ipAddressResource)(nil)
var _ resource.ResourceWithImportState = (*ipAddressResource)(nil)

type ipAddressResource struct{ client *client.Client }

var itemID = regexp.MustCompile(`^\*[0-9A-Fa-f]+$`)

func validID(value types.String) bool {
	return !value.IsNull() && !value.IsUnknown() && itemID.MatchString(value.ValueString())
}

func NewIPAddressResource() resource.Resource { return &ipAddressResource{} }
func (r *ipAddressResource) Metadata(_ context.Context, q resource.MetadataRequest, s *resource.MetadataResponse) {
	s.TypeName = q.ProviderTypeName + "_ip_address"
}
func (r *ipAddressResource) Schema(ctx context.Context, _ resource.SchemaRequest, s *resource.SchemaResponse) {
	s.Schema = generated.IpAddressResourceSchema(ctx)
	for name, v := range map[string]validator.String{"address": ipValidator{cidr: true}, "network": ipValidator{}, "interface": nonemptyValidator{}} {
		a := s.Schema.Attributes[name].(schema.StringAttribute)
		a.Validators = []validator.String{v}
		s.Schema.Attributes[name] = a
	}
}
func (r *ipAddressResource) Configure(_ context.Context, q resource.ConfigureRequest, s *resource.ConfigureResponse) {
	if q.ProviderData == nil {
		return
	}
	c, ok := q.ProviderData.(*client.Client)
	if !ok {
		s.Diagnostics.AddError("Invalid provider client", "Expected RouterOS REST client.")
		return
	}
	r.client = c
}
func validate(m generated.IpAddressModel) error {
	if m.Address.IsUnknown() || m.Address.IsNull() {
		return fmt.Errorf("address must be known")
	}
	if _, _, e := net.ParseCIDR(m.Address.ValueString()); e != nil && net.ParseIP(m.Address.ValueString()) == nil {
		return fmt.Errorf("address must be an IP address or CIDR")
	}
	if m.Interface.IsUnknown() || m.Interface.IsNull() || m.Interface.ValueString() == "" {
		return fmt.Errorf("interface must be a nonempty string")
	}
	if !m.Network.IsNull() && !m.Network.IsUnknown() && net.ParseIP(m.Network.ValueString()) == nil {
		return fmt.Errorf("network must be an IP address")
	}
	return nil
}
func payload(m generated.IpAddressModel) map[string]string {
	address := m.Address.ValueString()
	// A bare address means a host prefix. Explicitly send it so PATCH cannot
	// inherit an old RouterOS netmask and contradict the configured value.
	if ip := net.ParseIP(address); ip != nil {
		bits := 128
		if ip.To4() != nil {
			bits = 32
		}
		address = fmt.Sprintf("%s/%d", ip.String(), bits)
	}
	p := map[string]string{"address": address, "interface": m.Interface.ValueString()}
	for n, v := range map[string]types.String{"comment": m.Comment, "network": m.Network} {
		if !v.IsNull() && !v.IsUnknown() {
			p[catalog.IPAddressWireNames[n]] = v.ValueString()
		}
	}
	if !m.Disabled.IsNull() && !m.Disabled.IsUnknown() {
		if m.Disabled.ValueBool() {
			p["disabled"] = "yes"
		} else {
			p["disabled"] = "no"
		}
	}
	return p
}
func wireString(v any) string {
	if v == nil {
		return ""
	}
	switch x := v.(type) {
	case string:
		return x
	case bool:
		if x {
			return "true"
		}
		return "false"
	}
	return ""
}
func wireBool(v any) (types.Bool, error) {
	if v != nil {
		switch v.(type) {
		case string, bool:
		default:
			return types.BoolNull(), fmt.Errorf("invalid RouterOS boolean type")
		}
	}
	switch wireString(v) {
	case "true", "yes":
		return types.BoolValue(true), nil
	case "false", "no", "":
		return types.BoolValue(false), nil
	default:
		return types.BoolNull(), fmt.Errorf("invalid RouterOS boolean")
	}
}
func (r *ipAddressResource) refresh(ctx context.Context, m *generated.IpAddressModel) (bool, error) {
	if r.client == nil {
		return false, fmt.Errorf("RouterOS client is not configured")
	}
	if !validID(m.Id) {
		return false, fmt.Errorf("invalid RouterOS resource ID")
	}
	var rows []map[string]any
	e := r.client.Request(ctx, http.MethodGet, catalog.IPAddressPath, "", url.Values{catalog.IDKey: []string{m.Id.ValueString()}}, nil, &rows)
	var status *client.StatusError
	if errors.As(e, &status) && status.Code == 404 {
		return false, nil
	}
	if e != nil {
		return false, e
	}
	if len(rows) == 0 {
		return false, nil
	}
	if len(rows) != 1 || wireString(rows[0][catalog.IDKey]) != m.Id.ValueString() {
		return false, fmt.Errorf("RouterOS returned ambiguous or mismatched ID")
	}
	row := rows[0]
	for _, key := range []string{"address", "interface", "actual-interface", "comment", "network", "vrf"} {
		if v, ok := row[key]; ok {
			if _, ok := v.(string); !ok {
				return false, fmt.Errorf("invalid RouterOS string field type")
			}
		}
	}
	address := wireString(row["address"])
	// RouterOS adds /32 (or /128) to a configured bare host address. Retain
	// the configured representation only when it is semantically identical;
	// real address/prefix drift must still be visible.
	if prior := net.ParseIP(m.Address.ValueString()); prior != nil {
		if ip, network, err := net.ParseCIDR(address); err == nil {
			ones, bits := network.Mask.Size()
			if ones == bits && prior.Equal(ip) {
				address = m.Address.ValueString()
			}
		}
	}
	next := *m
	next.Address = types.StringValue(address)
	next.Interface = types.StringValue(wireString(row["interface"]))
	next.ActualInterface = types.StringValue(wireString(row["actual-interface"]))
	next.Comment = types.StringValue(wireString(row["comment"]))
	next.Network = types.StringValue(wireString(row["network"]))
	next.Vrf = types.StringValue(wireString(row["vrf"]))
	required := next
	required.Network = types.StringNull()
	if e := validate(required); e != nil {
		return false, fmt.Errorf("invalid RouterOS IP-address response")
	}
	if next.Network.ValueString() != "" && net.ParseIP(next.Network.ValueString()) == nil {
		return false, fmt.Errorf("invalid RouterOS network response")
	}
	for key, dest := range map[string]*types.Bool{"disabled": &next.Disabled, "dynamic": &next.Dynamic, "invalid": &next.Invalid, "slave": &next.Slave} {
		b, e := wireBool(row[key])
		if e != nil {
			return false, e
		}
		*dest = b
	}
	*m = next
	return true, nil
}
func (r *ipAddressResource) Create(ctx context.Context, q resource.CreateRequest, s *resource.CreateResponse) {
	var m, config generated.IpAddressModel
	s.Diagnostics.Append(q.Plan.Get(ctx, &m)...)
	s.Diagnostics.Append(q.Config.Get(ctx, &config)...)
	if s.Diagnostics.HasError() {
		return
	}
	if e := validate(m); e != nil {
		s.Diagnostics.AddError("Invalid IP address", e.Error())
		return
	}
	if r.client == nil {
		s.Diagnostics.AddError("Unconfigured client", "Configure the provider first.")
		return
	}
	var row map[string]any
	if e := r.client.Request(ctx, http.MethodPut, catalog.IPAddressPath, "", nil, payload(config), &row); e != nil {
		s.Diagnostics.AddError("Create failed", e.Error())
		return
	}
	id := wireString(row[catalog.IDKey])
	if !itemID.MatchString(id) {
		s.Diagnostics.AddError("Create failed", "RouterOS did not return a unique ID.")
		return
	}
	m.Id = types.StringValue(id)
	// Persist a fully known partial state before the read. Terraform cannot store
	// unknown computed plan values even when an operation reports an error.
	recovery := recoveryModel(m)
	s.Diagnostics.Append(s.State.Set(ctx, &recovery)...)
	found, e := r.refresh(ctx, &m)
	if e != nil {
		s.Diagnostics.AddError("Read after create failed", e.Error())
		return
	}
	if !found {
		s.Diagnostics.AddError("Read after create failed", "Created IP address was not found.")
		return
	}
	s.Diagnostics.Append(s.State.Set(ctx, &m)...)
}
func (r *ipAddressResource) Read(ctx context.Context, q resource.ReadRequest, s *resource.ReadResponse) {
	var m generated.IpAddressModel
	s.Diagnostics.Append(q.State.Get(ctx, &m)...)
	if s.Diagnostics.HasError() {
		return
	}
	found, e := r.refresh(ctx, &m)
	if e != nil {
		s.Diagnostics.AddError("Read failed", e.Error())
		return
	}
	if !found {
		s.State.RemoveResource(ctx)
		return
	}
	s.Diagnostics.Append(s.State.Set(ctx, &m)...)
}
func (r *ipAddressResource) Update(ctx context.Context, q resource.UpdateRequest, s *resource.UpdateResponse) {
	var m, old, config generated.IpAddressModel
	s.Diagnostics.Append(q.Plan.Get(ctx, &m)...)
	s.Diagnostics.Append(q.Config.Get(ctx, &config)...)
	s.Diagnostics.Append(q.State.Get(ctx, &old)...)
	if s.Diagnostics.HasError() {
		return
	}
	m.Id = old.Id
	if !validID(m.Id) {
		s.Diagnostics.AddError("Invalid resource ID", "Expected RouterOS .id.")
		return
	}
	if e := validate(m); e != nil {
		s.Diagnostics.AddError("Invalid IP address", e.Error())
		return
	}
	if r.client == nil {
		s.Diagnostics.AddError("Unconfigured client", "Configure the provider first.")
		return
	}
	if e := r.client.Request(ctx, http.MethodPatch, catalog.IPAddressPath, m.Id.ValueString(), nil, payload(config), nil); e != nil {
		s.Diagnostics.AddError("Update failed", e.Error())
		return
	}
	found, e := r.refresh(ctx, &m)
	if e != nil {
		s.Diagnostics.AddError("Read after update failed", e.Error())
		return
	}
	if !found {
		s.Diagnostics.AddError("Read after update failed", "IP address was not found.")
		return
	}
	s.Diagnostics.Append(s.State.Set(ctx, &m)...)
}
func (r *ipAddressResource) Delete(ctx context.Context, q resource.DeleteRequest, s *resource.DeleteResponse) {
	var m generated.IpAddressModel
	s.Diagnostics.Append(q.State.Get(ctx, &m)...)
	if s.Diagnostics.HasError() {
		return
	}
	if r.client == nil {
		s.Diagnostics.AddError("Unconfigured client", "Configure the provider first.")
		return
	}
	if !validID(m.Id) {
		s.Diagnostics.AddError("Invalid resource ID", "Expected RouterOS .id.")
		return
	}
	e := r.client.Request(ctx, http.MethodDelete, catalog.IPAddressPath, m.Id.ValueString(), nil, nil, nil)
	var status *client.StatusError
	if errors.As(e, &status) && status.Code == 404 {
		return
	}
	if e != nil {
		s.Diagnostics.AddError("Delete failed", e.Error())
	}
}
func (r *ipAddressResource) ImportState(ctx context.Context, q resource.ImportStateRequest, s *resource.ImportStateResponse) {
	if !itemID.MatchString(q.ID) {
		s.Diagnostics.AddError("Invalid import ID", "Expected RouterOS .id (for example *2).")
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), q, s)
}
