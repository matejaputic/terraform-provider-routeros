// SPDX-License-Identifier: MPL-2.0
package provider

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var routerDurationTokens = regexp.MustCompile(`([0-9]+(?:\.[0-9]+)?)(ms|[wdhms])`)

// Bounded, nonnegative durations. Equivalent readback retains configuration
// spelling; malformed/overflowing device values never overwrite prior state.
func routerDuration(text string) (time.Duration, error) {
	const maxSeconds = 3650 * 24 * 60 * 60
	total := float64(0)
	if strings.Contains(text, ":") {
		parts := strings.Split(text, ":")
		if len(parts) != 3 {
			return 0, fmt.Errorf("invalid duration")
		}
		for i, part := range parts {
			n, err := strconv.ParseFloat(part, 64)
			if err != nil || math.IsNaN(n) || math.IsInf(n, 0) || n < 0 || i > 0 && n >= 60 {
				return 0, fmt.Errorf("invalid duration")
			}
			total = total*60 + n
		}
	} else {
		tokens := routerDurationTokens.FindAllStringSubmatchIndex(text, -1)
		end := 0
		units := map[string]float64{"w": 604800, "d": 86400, "h": 3600, "m": 60, "s": 1, "ms": 0.001}
		for _, token := range tokens {
			if token[0] != end {
				return 0, fmt.Errorf("invalid duration")
			}
			n, err := strconv.ParseFloat(text[token[2]:token[3]], 64)
			if err != nil {
				return 0, fmt.Errorf("invalid duration")
			}
			total += n * units[text[token[4]:token[5]]]
			end = token[1]
		}
		if end != len(text) || len(tokens) == 0 {
			return 0, fmt.Errorf("invalid duration")
		}
	}
	if math.IsNaN(total) || math.IsInf(total, 0) || total < 0 || total > maxSeconds {
		return 0, fmt.Errorf("duration outside reviewed bounds")
	}
	milliseconds := math.Round(total * 1000)
	if math.Abs(total*1000-milliseconds) > 0.0001 {
		return 0, fmt.Errorf("duration requires millisecond precision")
	}
	return time.Duration(milliseconds) * time.Millisecond, nil
}
