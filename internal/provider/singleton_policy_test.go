// SPDX-License-Identifier: MPL-2.0
package provider

import (
	"github.com/matejaputic/terraform-provider-routeros/internal/catalog"
	"testing"
)

func TestSingletonRejectsUnsupportedPolicy(t *testing.T) {
	for _, mutate := range []func(*singletonResource){
		func(r *singletonResource) { r.core.policy.Fields = r.core.policy.Fields[1:] },
		func(r *singletonResource) { r.core.policy.Fields[0].Mode = "required" },
		func(r *singletonResource) { r.core.policy.Fields[0].Wire = "id" },
		func(r *singletonResource) { r.core.policy.Fields[1].Wire = ".id" },
		func(r *singletonResource) {
			r.core.policy.Fields = append(r.core.policy.Fields, r.core.policy.Fields[1])
		},
		func(r *singletonResource) { r.core.policy.Fields[1].Type = "array" },
		func(r *singletonResource) { r.core.policy.Fields[1].ConditionalRead = "guessed" },
		func(r *singletonResource) { x := ""; r.core.policy.Fields[1].ReadDefault = &x },
		func(r *singletonResource) { r.core.policy.Fields[1].ForceNew = true },
		func(r *singletonResource) { r.core.policy.Fields[1].Mode = "optional" },
		func(r *singletonResource) { r.core.policy.Fields[1] = catalog.Field{} },
	} {
		r := singletonFixture(t, nil)
		values := singletonValues(r)
		mutate(r)
		if _, e := r.payload(values); e == nil {
			t.Fatal("unsupported singleton policy authorized mutation")
		}
	}
}
