// SPDX-FileCopyrightText: 2026 Adriano Sela Aviles (@adrianosela)
// SPDX-License-Identifier: MIT

package wildcard

import (
	"fmt"
	"regexp"
	"strings"
)

// Match returns true if a given string matches
// a template string with wildcards ('*')
func Match(template string, check string) bool {
	return regexp.MustCompile(wildcardToRegexp(template)).MatchString(check)
}

// wildcardToRegexp returns a regular expression
// pattern given a template string with wildcards
func wildcardToRegexp(template string) string {
	parts := strings.Split(template, "*")
	if len(parts) == 1 {
		// no *'s, return exact match regex pattern
		return fmt.Sprintf("^%s$", template)
	}
	var result strings.Builder
	for i, literal := range parts {
		// replace * with .*
		if i > 0 {
			result.WriteString(".*")
		}
		// quote any regex meta characters
		result.WriteString(regexp.QuoteMeta(literal))
	}
	return fmt.Sprintf("^%s$", result.String())
}
