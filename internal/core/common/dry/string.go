package dry

import "strings"

// RemoveIndentation removes the minimum common leading spaces/tabs from each line in a multi-line string.
// It preserves extra indentation on some lines, matching the 'mixed indentation' test expectation.
//
// Example:
//
//	input := `
//	    line 1
//	    line 2
//	      line 3
//	`
//	result := RemoveIndentation(input)
//	// result: "line 1\nline 2\n  line 3"
func RemoveIndentation(s string) string {
	lines := strings.Split(s, "\n")
	if len(lines) == 0 {
		return s
	}

	// Remove empty lines from the beginning and end
	start := 0
	end := len(lines)
	for start < len(lines) && strings.TrimSpace(lines[start]) == "" {
		start++
	}
	for end > start && strings.TrimSpace(lines[end-1]) == "" {
		end--
	}
	if start >= end {
		return ""
	}
	lines = lines[start:end]

	// Find the minimum indentation of all non-empty lines
	minIndent := -1
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		indent := 0
		for _, c := range line {
			if c != ' ' && c != '\t' {
				break
			}
			indent++
		}
		if minIndent == -1 || indent < minIndent {
			minIndent = indent
		}
	}

	// Remove minIndent leading whitespace from each line
	result := make([]string, len(lines))
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			result[i] = ""
			continue
		}
		removed := 0
		j := 0
		for k, c := range line {
			if removed >= minIndent {
				j = k
				break
			}
			if c != ' ' && c != '\t' {
				j = k
				break
			}
			removed++
		}
		if removed < minIndent {
			j = len(line)
		}
		result[i] = line[j:]
	}
	return strings.Join(result, "\n")
}
