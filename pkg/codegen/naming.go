// Package codegen provides runtime support for generated code
package codegen

import (
	"strings"
	"unicode"
)

// GoKeywords maps Go keywords to safe variable names
var GoKeywords = map[string]string{
	"type":      "typeVal",
	"range":     "rangeVal",
	"map":       "mapVal",
	"chan":      "chanVal",
	"interface": "interfaceVal",
	"func":      "funcVal",
	"var":       "varVal",
	"const":     "constVal",
	"struct":    "structVal",
	"package":   "packageVal",
	"import":    "importVal",
	"return":    "returnVal",
	"if":        "ifVal",
	"else":      "elseVal",
	"for":       "forVal",
	"switch":    "switchVal",
	"case":      "caseVal",
	"default":   "defaultVal",
	"break":     "breakVal",
	"continue":  "continueVal",
	"goto":      "gotoVal",
	"defer":     "deferVal",
	"go":        "goVal",
	"select":    "selectVal",
	"fallthrough": "fallthroughVal",
}

// ToValidIdentifier converts any string to a valid Go identifier
// - Replaces hyphens and spaces with underscores
// - Converts to camelCase
// - Escapes Go keywords
func ToValidIdentifier(s string) string {
	if s == "" {
		return ""
	}

	// Replace hyphens and spaces with underscores
	s = strings.ReplaceAll(s, "-", "_")
	s = strings.ReplaceAll(s, " ", "_")

	// Convert to camelCase
	s = toCamelCase(s)

	// Check for Go keywords
	if replacement, ok := GoKeywords[s]; ok {
		return replacement
	}

	return s
}

// toCamelCase converts snake_case or PascalCase to camelCase
// First word is lowercase, subsequent words are Title case
// Handles: customer_number → customerNumber, CustomerNumber → customerNumber
func toCamelCase(s string) string {
	// If no underscores but has camelCase/PascalCase, convert to snake_case first
	if !strings.Contains(s, "_") && hasCamelCase(s) {
		s = camelToSnake(s)
	}

	parts := strings.Split(s, "_")
	for i, p := range parts {
		if p == "" {
			continue
		}
		if i == 0 {
			// First word: lowercase
			parts[i] = strings.ToLower(p)
		} else {
			// Subsequent words: Title case
			parts[i] = strings.Title(strings.ToLower(p))
		}
	}
	return strings.Join(parts, "")
}

// ToPascalCase converts snake_case or camelCase to PascalCase
// Handles: customer_number → CustomerNumber, customerNumber → CustomerNumber
func ToPascalCase(s string) string {
	if s == "" {
		return ""
	}

	// Replace hyphens with underscores
	s = strings.ReplaceAll(s, "-", "_")

	// If no underscores but has camelCase (e.g., "customerNumber"), convert to snake_case first
	if !strings.Contains(s, "_") && hasCamelCase(s) {
		s = camelToSnake(s)
	}

	parts := strings.Split(s, "_")
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.Title(strings.ToLower(p))
	}
	return strings.Join(parts, "")
}

// hasCamelCase checks if string contains uppercase letters (indicating camelCase/PascalCase)
func hasCamelCase(s string) bool {
	for _, r := range s {
		if unicode.IsUpper(r) {
			return true
		}
	}
	return false
}

// camelToSnake converts camelCase/PascalCase to snake_case
// e.g., customerNumber → customer_number, APIKey → api_key
func camelToSnake(s string) string {
	var result strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) {
			// Add underscore before uppercase letter (except at start)
			if i > 0 {
				result.WriteRune('_')
			}
			result.WriteRune(unicode.ToLower(r))
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// IsValidIdentifier checks if a string is a valid Go identifier
func IsValidIdentifier(s string) bool {
	if s == "" {
		return false
	}

	// Check first character
	r := rune(s[0])
	if !unicode.IsLetter(r) && r != '_' {
		return false
	}

	// Check remaining characters
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
			return false
		}
	}

	// Check for keywords
	if _, ok := GoKeywords[s]; ok {
		return false
	}

	return true
}
