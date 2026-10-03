// SPDX-License-Identifier: MPL-2.0
// Keep the existing Go entry point; the snapshot adapter uses Python's standard
// library and does not emit Framework code (only the official generators do).
package main

import (
	"fmt"
	"os"
	"os/exec"
)

func main() {
	args := append([]string{"tools/schema/normalize/normalize.py"}, os.Args[1:]...)
	cmd := exec.Command("python3", args...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "normalization failed")
		os.Exit(1)
	}
}
