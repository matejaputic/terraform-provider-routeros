// SPDX-License-Identifier: MPL-2.0
package catalog

import (
	_ "embed"
	"encoding/json"
)

//go:embed collections.json
var collectionsJSON []byte

//go:embed singletons.json
var singletonsJSON []byte

//go:embed batch-a-collections.json
var batchACollectionsJSON []byte

type Field struct {
	Name                     string   `json:"name"`
	Wire                     string   `json:"wire"`
	Type                     string   `json:"type"`
	Mode                     string   `json:"mode"`
	ForceNew                 bool     `json:"force_new"`
	Codec                    string   `json:"codec"`
	ReadDefault              *string  `json:"read_default"`
	ConditionalRead          string   `json:"conditional_read"`
	Choices                  []string `json:"choices"`
	Minimum                  *int64   `json:"minimum"`
	Maximum                  *int64   `json:"maximum"`
	Sensitive                bool     `json:"sensitive"`
	PreserveSecretOnOmission bool     `json:"preserve_secret_on_omission"`
	Constraint               string   `json:"constraint"`
}
type Collection struct {
	Name            string  `json:"resource_name"`
	Revision        string  `json:"revision"`
	Path            string  `json:"wire_path"`
	Fields          []Field `json:"attributes"`
	Lifecycle       string  `json:"lifecycle"`
	ReplacementOnly bool    `json:"replacement_only"`
	RequiredPackage string  `json:"required_package"`
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

func Singletons() []Collection {
	var result []Collection
	if err := json.Unmarshal(singletonsJSON, &result); err != nil {
		panic("invalid embedded reviewed singleton catalog")
	}
	return result
}

// Resources is the exact reviewed set consumed by independent code-spec gates.
func BatchACollections() []Collection {
	var result []Collection
	if err := json.Unmarshal(batchACollectionsJSON, &result); err != nil {
		panic("invalid embedded reviewed Batch A collection catalog")
	}
	return result
}
func Resources() []Collection {
	return append(append(Collections(), BatchACollections()...), Singletons()...)
}
