// SPDX-License-Identifier: MPL-2.0
package client

import (
	"fmt"
	"regexp"
	"sort"
)

var packageName = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

// ValidatePackages checks actually installed/enabled packages, not schema flavor.
// All enabled package versions must match the discovered system version. The
// current IP resource needs only routeros; extras do not authorize new resources.
func ValidatePackages(rawVersion string, rows []map[string]any) ([]string, error) {
	version, err := ParseVersion(rawVersion)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	enabled := map[string]bool{}
	for _, row := range rows {
		name, ok := row["name"].(string)
		if !ok || !packageName.MatchString(name) || seen[name] {
			return nil, fmt.Errorf("invalid or duplicate RouterOS package metadata")
		}
		seen[name] = true
		disabled := false
		switch x := row["disabled"].(type) {
		case nil:
		case bool:
			disabled = x
		case string:
			switch x {
			case "true", "yes":
				disabled = true
			case "false", "no":
			default:
				return nil, fmt.Errorf("invalid package disabled flag")
			}
		default:
			return nil, fmt.Errorf("invalid package disabled type")
		}
		if disabled {
			continue
		}
		text, ok := row["version"].(string)
		if !ok {
			return nil, fmt.Errorf("missing RouterOS package version")
		}
		actual, e := ParseVersion(text)
		if e != nil || actual != version {
			return nil, fmt.Errorf("enabled RouterOS package version disagrees with system version")
		}
		enabled[name] = true
	}
	if !enabled["routeros"] {
		return nil, fmt.Errorf("enabled routeros base package is required")
	}
	result := []string{}
	for name := range enabled {
		result = append(result, name)
	}
	sort.Strings(result)
	return result, nil
}
