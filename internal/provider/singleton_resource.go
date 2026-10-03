// SPDX-License-Identifier: MPL-2.0
package provider

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/matejaputic/terraform-provider-routeros/internal/catalog"
)

// singletonResource manages existing global settings, never collection records.
// It is intentionally not registered yet: each policy/schema pair must first
// pass the reviewed singleton adapter and both official generators. This helper
// does not authorize exposure of any entry in the Batch A planning manifest.
// Reference: pinned resource_actions.go SystemResourceCreateUpdate/Read/Delete.
// Create/update POST a fixed /set action; destroy only relinquishes management.
type singletonResource struct{ core *collectionResource }

var _ resource.ResourceWithImportState = (*singletonResource)(nil)
var _ resource.ResourceWithValidateConfig = (*singletonResource)(nil)

func newSingleton(policy catalog.Collection, s schema.Schema) resource.Resource {
	return &singletonResource{core: &collectionResource{policy: policy, schema: s}}
}
func (r *singletonResource) Metadata(ctx context.Context, q resource.MetadataRequest, s *resource.MetadataResponse) {
	r.core.Metadata(ctx, q, s)
}
func (r *singletonResource) Schema(ctx context.Context, q resource.SchemaRequest, s *resource.SchemaResponse) {
	r.core.Schema(ctx, q, s)
}
func (r *singletonResource) Configure(ctx context.Context, q resource.ConfigureRequest, s *resource.ConfigureResponse) {
	r.core.Configure(ctx, q, s)
}
func (r *singletonResource) identity() string {
	return strings.ReplaceAll(strings.TrimPrefix(r.core.policy.Path, "/"), "/", ".")
}

func (r *singletonResource) validate(values map[string]attr.Value, mutation bool) error {
	if r.identity() == "" {
		return fmt.Errorf("missing singleton identity")
	}
	seen := map[string]bool{}
	idCount := 0
	for _, f := range r.core.policy.Fields {
		if seen[f.Name] || f.Name == "" || f.Wire == "" {
			return fmt.Errorf("invalid singleton field identity")
		}
		seen[f.Name] = true
		if f.Name == "id" {
			idCount++
			if f.Wire != ".id" || f.Type != "string" || f.Mode != "computed" {
				return fmt.Errorf("invalid singleton ID policy")
			}
		} else if f.Wire == ".id" {
			return fmt.Errorf("singleton ID alias is not supported")
		}
		if f.ForceNew || f.ReadDefault != nil || f.ConditionalRead != "" || !oneOf(f.Type, "string", "boolean", "integer") || !oneOf(f.Mode, "computed", "required", "computed_optional") {
			return fmt.Errorf("unsupported singleton field policy: %s", f.Name)
		}
		v, ok := values[f.Name]
		if !ok || v == nil {
			return fmt.Errorf("missing singleton attribute %s", f.Name)
		}
		if v.IsUnknown() {
			if mutation && f.Mode != "computed" {
				return fmt.Errorf("unknown singleton attribute %s", f.Name)
			}
			continue
		}
		if v.IsNull() {
			if f.Mode == "required" {
				return fmt.Errorf("required singleton attribute %s", f.Name)
			}
			continue
		}
		switch f.Type {
		case "string":
			x, ok := v.(types.String)
			if !ok {
				return fmt.Errorf("invalid singleton string %s", f.Name)
			}
			text := x.ValueString()
			if strings.ContainsRune(text, 0) || (f.Name != "note" && strings.ContainsAny(text, "\r\n")) {
				return fmt.Errorf("invalid control character in %s", f.Name)
			}
			if f.Mode == "required" && f.Name != "note" && strings.TrimSpace(text) == "" {
				return fmt.Errorf("empty required singleton attribute %s", f.Name)
			}
		case "boolean":
			if _, ok := v.(types.Bool); !ok {
				return fmt.Errorf("invalid singleton boolean %s", f.Name)
			}
		case "integer":
			if _, ok := v.(types.Int64); !ok {
				return fmt.Errorf("invalid singleton integer %s", f.Name)
			}
		}
	}
	if idCount != 1 {
		return fmt.Errorf("exactly one computed singleton ID required")
	}
	return nil
}
func (r *singletonResource) ValidateConfig(ctx context.Context, q resource.ValidateConfigRequest, s *resource.ValidateConfigResponse) {
	var value types.Object
	s.Diagnostics.Append(q.Config.Get(ctx, &value)...)
	if s.Diagnostics.HasError() || value.IsUnknown() || value.IsNull() {
		return
	}
	if e := r.validate(value.Attributes(), false); e != nil {
		s.Diagnostics.AddError("Invalid singleton configuration", e.Error())
	}
}
func (r *singletonResource) payload(values map[string]attr.Value) (map[string]any, error) {
	if e := r.validate(values, true); e != nil {
		return nil, e
	}
	body := map[string]any{}
	for _, f := range r.core.policy.Fields {
		v := values[f.Name]
		if f.Mode == "computed" || v.IsNull() {
			continue
		}
		switch x := v.(type) {
		case types.String:
			body[f.Wire] = x.ValueString()
		case types.Bool:
			if x.ValueBool() {
				body[f.Wire] = "yes"
			} else {
				body[f.Wire] = "no"
			}
		case types.Int64:
			body[f.Wire] = strconv.FormatInt(x.ValueInt64(), 10)
		default:
			return nil, fmt.Errorf("unsupported singleton mutation type")
		}
	}
	return body, nil
}
func (r *singletonResource) refresh(ctx context.Context, old map[string]attr.Value) (map[string]attr.Value, error) {
	id, ok := old["id"].(types.String)
	if !ok || id.IsNull() || id.IsUnknown() || id.ValueString() != r.identity() {
		return nil, fmt.Errorf("invalid singleton identity")
	}
	var observed map[string]any
	if e := r.core.client.Request(ctx, http.MethodGet, r.core.policy.Path, "", nil, nil, &observed); e != nil {
		return nil, e
	}
	if len(observed) == 0 {
		return nil, fmt.Errorf("empty singleton settings response")
	}
	next := map[string]attr.Value{}
	for _, f := range r.core.policy.Fields {
		if f.Name == "id" {
			next[f.Name] = id
			continue
		}
		raw, exists := observed[f.Wire]
		if !exists {
			previous := old[f.Name]
			if f.Mode == "required" || f.Mode != "computed" && previous != nil && !previous.IsNull() && !previous.IsUnknown() {
				return nil, fmt.Errorf("missing configured singleton field %s", f.Name)
			}
			next[f.Name] = nullField(f)
			continue
		}
		v, e := decodeField(ctx, f, raw)
		if e != nil {
			return nil, e
		}
		next[f.Name] = v
	}
	if e := r.validate(next, false); e != nil {
		return nil, e
	}
	return next, nil
}
func (r *singletonResource) Create(ctx context.Context, q resource.CreateRequest, s *resource.CreateResponse) {
	if !r.core.ready(&s.Diagnostics) {
		return
	}
	var config, plan types.Object
	s.Diagnostics.Append(q.Config.Get(ctx, &config)...)
	s.Diagnostics.Append(q.Plan.Get(ctx, &plan)...)
	if s.Diagnostics.HasError() {
		return
	}
	if config.IsNull() || config.IsUnknown() || plan.IsNull() || plan.IsUnknown() {
		s.Diagnostics.AddError("Invalid singleton plan", "Expected known configuration and plan objects.")
		return
	}
	body, e := r.payload(config.Attributes())
	if e != nil {
		s.Diagnostics.AddError("Invalid singleton configuration", e.Error())
		return
	}
	// Observe before mutation. Existing settings have a deterministic ID, so retain
	// recovery state even if /set times out or the post-write read is malformed.
	baseline := map[string]attr.Value{}
	for _, f := range r.core.policy.Fields {
		baseline[f.Name] = nullField(f)
	}
	baseline["id"] = types.StringValue(r.identity())
	baseline, e = r.refresh(ctx, baseline)
	if e != nil {
		s.Diagnostics.AddError("RouterOS singleton preflight failed", e.Error())
		return
	}
	s.Diagnostics.Append(s.State.Set(ctx, r.core.object(baseline))...)
	if s.Diagnostics.HasError() {
		return
	}
	if len(body) > 0 {
		if e = r.core.client.Request(ctx, http.MethodPost, r.core.policy.Path+"/set", "", nil, body, nil); e != nil {
			s.Diagnostics.AddError("RouterOS singleton create failed", e.Error())
			return
		}
	}
	expected := plan.Attributes()
	expected["id"] = types.StringValue(r.identity())
	next, e := r.refresh(ctx, expected)
	if e != nil {
		s.Diagnostics.AddError("RouterOS singleton create refresh failed", e.Error())
		return
	}
	s.Diagnostics.Append(s.State.Set(ctx, r.core.object(next))...)
}
func (r *singletonResource) Read(ctx context.Context, q resource.ReadRequest, s *resource.ReadResponse) {
	if !r.core.ready(&s.Diagnostics) {
		return
	}
	var old types.Object
	s.Diagnostics.Append(q.State.Get(ctx, &old)...)
	if s.Diagnostics.HasError() {
		return
	}
	next, e := r.refresh(ctx, old.Attributes())
	if e != nil {
		s.Diagnostics.AddError("RouterOS singleton read failed", e.Error())
		return
	}
	s.Diagnostics.Append(s.State.Set(ctx, r.core.object(next))...)
}
func (r *singletonResource) Update(ctx context.Context, q resource.UpdateRequest, s *resource.UpdateResponse) {
	if !r.core.ready(&s.Diagnostics) {
		return
	}
	var old, config, plan types.Object
	s.Diagnostics.Append(q.State.Get(ctx, &old)...)
	s.Diagnostics.Append(q.Config.Get(ctx, &config)...)
	s.Diagnostics.Append(q.Plan.Get(ctx, &plan)...)
	if s.Diagnostics.HasError() {
		return
	}
	if old.IsNull() || old.IsUnknown() || config.IsNull() || config.IsUnknown() || plan.IsNull() || plan.IsUnknown() {
		s.Diagnostics.AddError("Invalid singleton plan", "Expected known state, configuration and plan objects.")
		return
	}
	body, e := r.payload(config.Attributes())
	if e != nil {
		s.Diagnostics.AddError("Invalid singleton configuration", e.Error())
		return
	}
	if _, e = r.refresh(ctx, old.Attributes()); e != nil {
		s.Diagnostics.AddError("RouterOS singleton preflight failed", e.Error())
		return
	}
	if len(body) > 0 {
		if e = r.core.client.Request(ctx, http.MethodPost, r.core.policy.Path+"/set", "", nil, body, nil); e != nil {
			s.Diagnostics.AddError("RouterOS singleton update failed", e.Error())
			return
		}
	}
	expected := plan.Attributes()
	expected["id"] = old.Attributes()["id"]
	next, e := r.refresh(ctx, expected)
	if e != nil {
		s.Diagnostics.AddError("RouterOS singleton update refresh failed", e.Error())
		return
	}
	s.Diagnostics.Append(s.State.Set(ctx, r.core.object(next))...)
}
func (r *singletonResource) Delete(ctx context.Context, _ resource.DeleteRequest, s *resource.DeleteResponse) {
	// No connection, DELETE, reset or stale-state replay on destroy, including when
	// the device is offline. Settings belong to the device, not to Terraform.
	s.State.RemoveResource(ctx)
	s.Diagnostics.AddWarning("Singleton settings retained", "Destroy stops managing these RouterOS settings; it does not reset them.")
}
func (r *singletonResource) ImportState(ctx context.Context, q resource.ImportStateRequest, s *resource.ImportStateResponse) {
	if q.ID != r.identity() {
		s.Diagnostics.AddError("Invalid singleton import ID", "Use the documented path-derived singleton identity.")
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), q, s)
}
