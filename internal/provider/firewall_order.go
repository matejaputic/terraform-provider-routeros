// SPDX-License-Identifier: MPL-2.0
package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"net/http"
)

// Fixed reviewed command bindings, never arbitrary catalog-derived actions.
func (r *collectionResource) orderedFirewall() bool {
	paths := map[string]string{"ip_firewall_filter": "/ip/firewall/filter", "ip_firewall_nat": "/ip/firewall/nat", "ip_firewall_mangle": "/ip/firewall/mangle", "ip_firewall_raw": "/ip/firewall/raw"}
	p, ok := paths[r.policy.Name]
	return ok && p == r.policy.Path
}
func (r *collectionResource) filterOrder(ctx context.Context, chain string) ([]string, error) {
	if !r.orderedFirewall() {
		return nil, fmt.Errorf("unregistered ordered lifecycle")
	}
	var rows []map[string]any
	if e := r.client.Request(ctx, http.MethodGet, r.policy.Path, "", nil, nil, &rows); e != nil {
		return nil, e
	}
	ids := []string{}
	seen := map[string]bool{}
	for _, row := range rows {
		id, ok := row[".id"].(string)
		if !ok || !itemID.MatchString(id) || seen[id] {
			return nil, fmt.Errorf("invalid/duplicate ordered rule ID")
		}
		seen[id] = true
		c, ok := row["chain"].(string)
		if !ok || c == "" {
			return nil, fmt.Errorf("missing ordered rule chain")
		}
		if c != chain {
			continue
		}
		if raw, ok := row["dynamic"]; ok {
			flag, e := strictWireBool(raw)
			if e != nil {
				return nil, e
			}
			if flag.ValueBool() {
				return nil, fmt.Errorf("dynamic rules in the managed chain are not supported")
			}
		}
		ids = append(ids, id)
	}
	return ids, nil
}
func (r *collectionResource) nextFilterRule(ctx context.Context, id, chain string) (string, error) {
	ids, e := r.filterOrder(ctx, chain)
	if e != nil {
		return "", e
	}
	for i, v := range ids {
		if v == id {
			if i+1 < len(ids) {
				return ids[i+1], nil
			}
			return "", nil
		}
	}
	return "", fmt.Errorf("ordered rule absent from collection snapshot")
}
func (r *collectionResource) positionFilter(ctx context.Context, id string, values map[string]attr.Value, move bool) error {
	if !r.orderedFirewall() {
		return nil
	}
	before := values["place_before"].(types.String)
	if before.IsNull() {
		return nil
	}
	if before.IsUnknown() {
		return fmt.Errorf("ordering anchor must be known")
	}
	anchor := before.ValueString()
	if anchor != "" && !itemID.MatchString(anchor) {
		return fmt.Errorf("place_before requires an internal *HEX ID or empty string for chain end")
	}
	if id != "" && id == anchor {
		return fmt.Errorf("a rule cannot be its own ordering anchor")
	}
	chain := values["chain"].(types.String).ValueString()
	ids, e := r.filterOrder(ctx, chain)
	if e != nil {
		return e
	}
	found := anchor == ""
	subject := id == "" || !move // Before PATCH, the subject may still belong to its old chain.
	for _, v := range ids {
		if v == anchor {
			found = true
		}
		if v == id {
			subject = true
		}
	}
	if !found || !subject {
		return fmt.Errorf("ordering subject/anchor must exist in the same static chain")
	}
	if !move {
		return nil
	}
	next, e := r.nextFilterRule(ctx, id, chain)
	if e != nil {
		return e
	}
	if next == anchor {
		return nil
	}
	body := map[string]string{"numbers": id}
	if anchor != "" {
		body["destination"] = anchor
	}
	if e := r.client.Request(ctx, http.MethodPost, r.policy.Path+"/move", "", nil, body, nil); e != nil {
		return e
	}
	next, e = r.nextFilterRule(ctx, id, chain)
	if e != nil {
		return e
	}
	if next != anchor {
		return fmt.Errorf("move did not establish requested chain-relative position")
	}
	return nil
}
