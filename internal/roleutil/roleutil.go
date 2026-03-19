package roleutil

import (
	"regexp"
	"strings"
	"unicode"
)

// DisplayNameToName converts "Forum Moderator" to "forumModerator" (camelCase).
func DisplayNameToName(displayName string) string {
	if displayName == "" {
		return ""
	}
	words := splitWords(displayName)
	if len(words) == 0 {
		return ""
	}
	for i, w := range words {
		w = strings.ToLower(w)
		if i > 0 && len(w) > 0 {
			runes := []rune(w)
			runes[0] = unicode.ToUpper(runes[0])
			words[i] = string(runes)
		} else {
			words[i] = w
		}
	}
	return strings.Join(words, "")
}

// DisplayNameToSlug converts "Forum Moderator" to "forum-moderator" (kebab-case).
func DisplayNameToSlug(displayName string) string {
	if displayName == "" {
		return ""
	}
	words := splitWords(displayName)
	if len(words) == 0 {
		return ""
	}
	for i, w := range words {
		words[i] = strings.ToLower(w)
	}
	return strings.Join(words, "-")
}

var wordSplitRegex = regexp.MustCompile(`[\s\p{Z}+]+`)

func splitWords(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parts := wordSplitRegex.Split(s, -1)
	var words []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			words = append(words, p)
		}
	}
	return words
}
