package draft

import (
	"strings"
)

// preamblePatterns are common LLM thought-process prefixes that leak into output.
// Each pattern is checked as a case-insensitive prefix of the first non-empty line.
var preamblePatterns = []string{
	"here is",
	"here's",
	"i'll ",
	"i will ",
	"i've ",
	"i have ",
	"let me ",
	"sure,",
	"sure!",
	"okay,",
	"okay!",
	"certainly",
	"absolutely",
	"of course",
	"now i ",
	"now let me",
	"based on",
	"looking at",
	"after reviewing",
	"after analyzing",
	"having reviewed",
	"having analyzed",
}

// signoffPatterns are common LLM sign-offs appended after the actual content.
var signoffPatterns = []string{
	"let me know",
	"feel free to",
	"hope this helps",
	"is there anything",
	"would you like",
	"shall i ",
	"do you want",
	"i can also",
	"if you need",
	"if you'd like",
}

// SanitizeLLMOutput strips common LLM preamble and sign-off patterns from generated content.
// This provides a deterministic safety net beyond prompt instructions.
func SanitizeLLMOutput(content string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return content
	}

	content = stripPreamble(content)
	content = stripSignoff(content)

	return strings.TrimSpace(content)
}

// stripPreamble removes leading lines that match preamble patterns.
// Strips at most 3 lines to avoid eating actual content.
func stripPreamble(content string) string {
	lines := strings.SplitN(content, "\n", 5) // look at first few lines only
	stripped := 0

	for stripped < len(lines) && stripped < 3 {
		line := strings.TrimSpace(lines[stripped])
		if line == "" {
			stripped++
			continue
		}
		if matchesAnyPrefix(line, preamblePatterns) {
			stripped++
			continue
		}
		break
	}

	if stripped == 0 {
		return content
	}

	return strings.Join(lines[stripped:], "\n")
}

// stripSignoff removes trailing lines that match sign-off patterns.
func stripSignoff(content string) string {
	lines := strings.Split(content, "\n")

	// Work backwards from the end
	end := len(lines)
	for end > 0 {
		line := strings.TrimSpace(lines[end-1])
		if line == "" {
			end--
			continue
		}
		if matchesAnyPrefix(line, signoffPatterns) {
			end--
			continue
		}
		break
	}

	if end == len(lines) {
		return content
	}

	return strings.Join(lines[:end], "\n")
}

// matchesAnyPrefix checks if the line starts with any of the given patterns (case-insensitive).
func matchesAnyPrefix(line string, patterns []string) bool {
	lower := strings.ToLower(line)
	for _, p := range patterns {
		if strings.HasPrefix(lower, p) {
			return true
		}
	}
	return false
}
