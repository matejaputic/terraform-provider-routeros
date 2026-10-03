// SPDX-License-Identifier: MPL-2.0
package catalog

import (
	_ "embed"
	"encoding/json"
)

//go:embed collections.json
var collectionsJSON []byte

type Field struct {
	Name            string  `json:"name"`
	Wire            string  `json:"wire"`
	Type            string  `json:"type"`
	Mode            string  `json:"mode"`
	ForceNew        bool    `json:"force_new"`
	Codec           string  `json:"codec"`
	ReadDefault     *string `json:"read_default"`
	ConditionalRead string  `json:"conditional_read"`
}
type Collection struct {
	Name   string  `json:"resource_name"`
	Path   string  `json:"wire_path"`
	Fields []Field `json:"attributes"`
}

// Collections returns a fresh policy copy: runtime instances cannot mutate the
// reviewed registry shared by other provider aliases.
func Collections() []Collection {
	var result []Collection
	if err := json.Unmarshal(collectionsJSON, &result); err != nil {
		panic("invalid embedded reviewed catalog")
	}
	return result
}
