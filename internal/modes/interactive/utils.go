package interactive

import (
	"unicode"
	"unicode/utf8"
)

// Capitalize returns the string with its first rune uppercased.
func Capitalize(s string) string {
	if s == "" {
		return ""
	}
	r, size := utf8.DecodeRuneInString(s)
	return string(unicode.ToUpper(r)) + s[size:]
}
