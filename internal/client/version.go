// SPDX-License-Identifier: MPL-2.0
package client

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Version is consumer-owned parsing, not the reference provider's global state.
// Nightly build identifiers are comparable only with other nightlies of the same base.
type Version struct {
	Major, Minor, Patch, Sequence int
	Train                         string
}

var versionPattern = regexp.MustCompile(`^([0-9]+)\.([0-9]+)(?:\.([0-9]+))?(?:(beta|rc)([0-9]+)|_ab([0-9]+))?$`)

func ParseVersion(raw string) (Version, error) {
	text := strings.TrimSpace(raw)
	if i := strings.Index(text, " ("); i >= 0 {
		text = text[:i]
	}
	m := versionPattern.FindStringSubmatch(text)
	if m == nil {
		return Version{}, fmt.Errorf("unsupported RouterOS version syntax")
	}
	v := Version{Train: "stable"}
	for i, dest := range map[int]*int{1: &v.Major, 2: &v.Minor, 3: &v.Patch} {
		if m[i] != "" {
			n, e := strconv.Atoi(m[i])
			if e != nil {
				return Version{}, fmt.Errorf("invalid RouterOS version number")
			}
			*dest = n
		}
	}
	if m[4] != "" {
		v.Train = m[4]
		n, e := strconv.Atoi(m[5])
		if e != nil {
			return Version{}, fmt.Errorf("invalid prerelease number")
		}
		v.Sequence = n
	}
	if m[6] != "" {
		v.Train = "nightly"
		n, e := strconv.Atoi(m[6])
		if e != nil {
			return Version{}, fmt.Errorf("invalid nightly number")
		}
		v.Sequence = n
	}
	return v, nil
}
func (v Version) Compare(other Version) (int, error) {
	if v.Train == "nightly" || other.Train == "nightly" {
		if v.Train != other.Train || v.Major != other.Major || v.Minor != other.Minor || v.Patch != other.Patch {
			return 0, fmt.Errorf("nightly and release versions are not orderable across trains/bases")
		}
	}
	rank := map[string]int{"beta": 0, "rc": 1, "stable": 2, "nightly": 3}
	a := []int{v.Major, v.Minor, v.Patch, rank[v.Train], v.Sequence}
	b := []int{other.Major, other.Minor, other.Patch, rank[other.Train], other.Sequence}
	for i := range a {
		if a[i] < b[i] {
			return -1, nil
		}
		if a[i] > b[i] {
			return 1, nil
		}
	}
	return 0, nil
}

// CheckRuntimeVersion gates the implemented RouterOS 7 REST family, not acceptance
// compatibility. Only 7.24.5/base/x86_64 has live evidence; other minors/trains warn.
func CheckRuntimeVersion(raw string) (reviewed bool, err error) {
	v, err := ParseVersion(raw)
	if err != nil {
		return false, err
	}
	if v.Major != 7 {
		return false, fmt.Errorf("only the RouterOS 7 REST family is implemented")
	}
	return v.Minor == 24 && v.Patch == 5 && v.Train == "stable", nil
}
