// SPDX-License-Identifier: MPL-2.0
package provider

import (
	"os"
	"testing"
)

// This is a disposable-test recipe selector, not a provider support allowlist.
// Keep the explicit target names aligned with tools/chr/recipes.json. Unknown
// input must fail before any mutation rather than allowing arbitrary routers.
func expectedCHRVersion(t *testing.T) string {
	t.Helper()
	version := os.Getenv("ROS_TEST_VERSION")
	switch version {
	case "", "7.24.5":
		return "7.24.5 (stable)"
	case "7.25beta5":
		return "7.25beta5"
	default:
		t.Fatal("unsupported disposable CHR acceptance recipe")
		return ""
	}
}

func TestExpectedCHRVersion(t *testing.T) {
	for input, expected := range map[string]string{"": "7.24.5 (stable)", "7.24.5": "7.24.5 (stable)", "7.25beta5": "7.25beta5"} {
		t.Setenv("ROS_TEST_VERSION", input)
		if actual := expectedCHRVersion(t); actual != expected {
			t.Fatalf("recipe version: got %q want %q", actual, expected)
		}
	}
}
