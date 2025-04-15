package openai

import "strings"

// SplitMsg splits a message into parts that are under lengthLimit characters.
// Splits by newlines or spaces if possible.
// Returns slice with at least one element.
func SplitMsg(msg string, lengthLimit int) []string {
	if lengthLimit <= 0 {
		return []string{msg}
	}

	if len(msg) <= lengthLimit {
		return []string{msg}
	}

	parts := make([]string, 0, len(msg)/lengthLimit+1)
	remaining := msg

	for len(remaining) > lengthLimit {
		splitIndex := findSplitIndex(remaining[:lengthLimit])
		parts = append(parts, remaining[:splitIndex])
		remaining = strings.TrimLeft(remaining[splitIndex:], " \n")
	}

	if len(remaining) > 0 {
		parts = append(parts, remaining)
	}

	return parts
}

func findSplitIndex(text string) int {
	if splitIndex := strings.LastIndex(text, "\n"); splitIndex != -1 {
		return splitIndex
	}
	if splitIndex := strings.LastIndex(text, " "); splitIndex != -1 {
		return splitIndex
	}
	return len(text)
}
