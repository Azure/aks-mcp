package utils

import "strings"

// ReplaceSpacesWithUnderscores converts spaces to underscores
// to create a valid tool name that follows the [a-z0-9_-] pattern
func ReplaceSpacesWithUnderscores(s string) string {
	return strings.ReplaceAll(s, " ", "_")
}
