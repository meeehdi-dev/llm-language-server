package provider

import "strings"

// truncate extracts a window of lines around position `index` in `text`.
// Returns (promptBeforeCursor, suffixAfterCursor) expanded outward line by
// line up to `ctx` total characters.
func truncate(text string, index int, ctx int) (string, string) {
	promptLines := strings.Split(text[:index], "\n")
	suffixLines := strings.Split(text[index:], "\n")

	// Start with current line on each side
	prompt := promptLines[len(promptLines)-1]
	suffix := suffixLines[len(suffixLines)-1]

	// Rough estimate of characters so far (quarter of actual to bias expansion)
	ctxSize := len(prompt+suffix) / 4

	promptLinesCount := 1
	suffixLinesCount := 1

	// Alternating expansion: suffix line, then prompt line, repeat
	ctxInc := true
	for ctxInc {
		ctxInc = false

		if suffixLinesCount < len(suffixLines) {
			suffix_line := suffixLines[suffixLinesCount]
			if ctxSize+len(suffix_line) < ctx {
				suffix = suffix + "\n" + suffix_line
				suffixLinesCount++
				ctxInc = true
			}
		}

		if promptLinesCount < len(promptLines) {
			promptLine := promptLines[len(promptLines)-promptLinesCount-1]
			if ctxSize+len(promptLine) < ctx {
				prompt = promptLine + "\n" + prompt
				promptLinesCount++
				ctxInc = true
			}
		}
	}

	return prompt, suffix
}
