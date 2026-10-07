package tui

import (
	"strings"
	"unicode/utf8"
)

// RenderMarkdown converts the small Markdown subset used by LPIC Daily into
// readable plain-terminal text. It deliberately avoids ANSI styling so the
// result stays legible in every terminal and in redirected output.
func RenderMarkdown(markdown string) string {
	markdown = strings.ReplaceAll(markdown, "\r\n", "\n")
	lines := strings.Split(markdown, "\n")

	var out strings.Builder
	inCode := false
	for _, raw := range lines {
		trimmed := strings.TrimSpace(raw)

		if strings.HasPrefix(trimmed, "```") {
			inCode = !inCode
			continue
		}
		if inCode {
			if trimmed == "" {
				out.WriteByte('\n')
				continue
			}
			out.WriteString("    ")
			out.WriteString(raw)
			out.WriteByte('\n')
			continue
		}

		if trimmed == "" {
			out.WriteByte('\n')
			continue
		}

		if level, title, ok := markdownHeading(trimmed); ok {
			title = renderInlineMarkdown(title)
			out.WriteString(title)
			out.WriteByte('\n')
			underline := "-"
			if level == 1 {
				underline = "="
			}
			out.WriteString(strings.Repeat(underline, max(3, utf8.RuneCountInString(title))))
			out.WriteByte('\n')
			continue
		}

		switch {
		case strings.HasPrefix(trimmed, "- "):
			out.WriteString("• ")
			out.WriteString(renderInlineMarkdown(strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))))
		case strings.HasPrefix(trimmed, "* "):
			out.WriteString("• ")
			out.WriteString(renderInlineMarkdown(strings.TrimSpace(strings.TrimPrefix(trimmed, "* "))))
		case strings.HasPrefix(trimmed, "> "):
			out.WriteString("│ ")
			out.WriteString(renderInlineMarkdown(strings.TrimSpace(strings.TrimPrefix(trimmed, "> "))))
		default:
			out.WriteString(renderInlineMarkdown(raw))
		}
		out.WriteByte('\n')
	}

	return strings.TrimSpace(out.String())
}

func markdownHeading(line string) (int, string, bool) {
	for level := 6; level >= 1; level-- {
		prefix := strings.Repeat("#", level) + " "
		if strings.HasPrefix(line, prefix) {
			return level, strings.TrimSpace(strings.TrimPrefix(line, prefix)), true
		}
	}
	return 0, "", false
}

func renderInlineMarkdown(line string) string {
	line = strings.ReplaceAll(line, "**", "")
	line = strings.ReplaceAll(line, "__", "")
	line = strings.ReplaceAll(line, "`", "")
	return line
}
