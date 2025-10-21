package repositorycache

import (
	"strings"
	"unicode"
)

// toSnake converts the provided string to snake_case using ASCII-aware rules.
// We keep this implementation local so we can aggressively strip punctuation
// (e.g. pointers, generic suffixes) that can show up in reflected type names;
// leaving those characters in the cache namespace would break our prefix-based
// invalidation strategy and produce keys Redis/Memcache reject.
func toSnake(s string) string {
	if s == "" {
		return ""
	}

	runes := []rune(s)
	var b strings.Builder
	b.Grow(len(runes) + len(runes)/2)

	lastUnderscore := false

	for i := 0; i < len(runes); i++ {
		r := runes[i]

		switch {
		case unicode.IsUpper(r):
			if b.Len() > 0 {
				prev := runes[i-1]
				nextLower := i+1 < len(runes) && unicode.IsLower(runes[i+1])
				if (unicode.IsLower(prev) || unicode.IsDigit(prev) || nextLower) && !lastUnderscore {
					b.WriteByte('_')
					lastUnderscore = true
				}
			}
			b.WriteRune(unicode.ToLower(r))
			lastUnderscore = false

		case unicode.IsLower(r):
			b.WriteRune(r)
			lastUnderscore = false

		case unicode.IsDigit(r):
			if b.Len() > 0 {
				prev := runes[i-1]
				if !unicode.IsDigit(prev) && prev != '_' && !lastUnderscore {
					b.WriteByte('_')
				}
			}
			b.WriteRune(r)
			lastUnderscore = false

		case r == '_':
			if !lastUnderscore && b.Len() > 0 {
				b.WriteByte('_')
				lastUnderscore = true
			}

		case r == '-' || unicode.IsSpace(r):
			if !lastUnderscore && b.Len() > 0 {
				b.WriteByte('_')
				lastUnderscore = true
			}

		default:
			if !lastUnderscore && b.Len() > 0 {
				b.WriteByte('_')
				lastUnderscore = true
			}
		}
	}

	return strings.Trim(b.String(), "_")
}
