package tui

import (
	"strings"
	"testing"
)

func TestRenderMarkdownProducesReadableTerminalText(t *testing.T) {
	input := "# Titre\n\n## À faire\n\n- Utilise `env` et **compare** le résultat.\n- Puis lance `set`.\n\n```sh\nVAR=test\nenv\n```"
	got := RenderMarkdown(input)

	for _, rawMarker := range []string{"# ", "## ", "**", "`", "```"} {
		if strings.Contains(got, rawMarker) {
			t.Fatalf("RenderMarkdown() left raw marker %q in %q", rawMarker, got)
		}
	}
	for _, want := range []string{"Titre", "À faire", "• Utilise env et compare le résultat.", "VAR=test", "env"} {
		if !strings.Contains(got, want) {
			t.Fatalf("RenderMarkdown() = %q, want %q", got, want)
		}
	}
}
