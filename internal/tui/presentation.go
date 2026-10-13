package tui

import (
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/Loe159/lpic-daily/internal/terminal"
)

// Colors are an enhancement, never the only way a state is communicated.
type Tone string

const (
	ToneAccent  Tone = "accent"
	ToneSuccess Tone = "success"
	ToneWarning Tone = "warning"
	ToneMuted   Tone = "muted"
	ToneTitle   Tone = "title"
)

func ColorEnabled(out io.Writer) bool {
	if _, disabled := os.LookupEnv("NO_COLOR"); disabled || os.Getenv("TERM") == "dumb" {
		return false
	}
	file, ok := out.(*os.File)
	return ok && terminal.IsTerminal(file)
}

func Paint(enabled bool, tone Tone, value string) string {
	if !enabled {
		return value
	}
	code := "0"
	switch tone {
	case ToneAccent:
		code = "1;36"
	case ToneSuccess:
		code = "1;32"
	case ToneWarning:
		code = "1;33"
	case ToneMuted:
		code = "2"
	case ToneTitle:
		code = "1"
	}
	return "\x1b[" + code + "m" + value + "\x1b[0m"
}

func TerminalWidth(out io.Writer) int {
	if file, ok := out.(*os.File); ok && terminal.IsTerminal(file) {
		if w, _, err := terminal.Size(file); err == nil && w >= 30 {
			return min(int(w), 110)
		}
	}
	return 88
}

// WrapParagraph preserves every word, while making long lab instructions scanable.
func WrapParagraph(value string, width int, indent string) string {
	if width < 24 {
		width = 24
	}
	words := strings.Fields(SanitizeText(value))
	if len(words) == 0 {
		return ""
	}
	var result strings.Builder
	lineLen := 0
	for _, word := range words {
		n := utf8.RuneCountInString(word)
		if lineLen > 0 && lineLen+1+n > width {
			result.WriteByte('\n')
			result.WriteString(indent)
			lineLen = utf8.RuneCountInString(indent)
		} else if lineLen > 0 {
			result.WriteByte(' ')
			lineLen++
		}
		result.WriteString(word)
		lineLen += n
	}
	return result.String()
}

type LabPresentation struct {
	Title      string
	Objectives []string
	Minutes    int
	Mission    string
	Criteria   []string
}

// PrintLabBrief has a stable reading order: mission, success, controls.
func PrintLabBrief(out io.Writer, lab LabPresentation) {
	color := ColorEnabled(out)
	width := TerminalWidth(out)
	title := singleLine(lab.Title)
	fmt.Fprintln(out, Paint(color, ToneAccent, "LPIC  /  LAB"))
	fmt.Fprintln(out, Paint(color, ToneTitle, WrapParagraph(title, width, "")))
	meta := fmt.Sprintf("%s   ·   ~%d min", strings.Join(lab.Objectives, ", "), lab.Minutes)
	fmt.Fprintln(out, Paint(color, ToneMuted, meta))
	fmt.Fprintln(out)
	fmt.Fprintln(out, Paint(color, ToneAccent, "MISSION"))
	mission := RenderMarkdown(lab.Mission)
	for _, paragraph := range strings.Split(mission, "\n\n") {
		for _, line := range strings.Split(paragraph, "\n") {
			if strings.TrimSpace(line) != "" {
				fmt.Fprintln(out, WrapParagraph(line, width-2, ""))
			}
		}
	}
	fmt.Fprintln(out)
	fmt.Fprintln(out, Paint(color, ToneAccent, "RÉUSSITE  ·  état attendu"))
	for _, criterion := range lab.Criteria {
		text := WrapParagraph(singleLine(criterion), width-5, "    ")
		fmt.Fprintln(out, "  "+Paint(color, ToneMuted, "□")+" "+text)
	}
	fmt.Fprintln(out)
	fmt.Fprintln(out, Paint(color, ToneMuted, ":check vérifier  ·  :hint indice  ·  :reset recommencer  ·  :quit quitter"))
}

type CheckPresentation struct {
	Label      string
	Passed     bool
	Diagnostic string
}

// PrintCheckResults puts actionable failures before already successful criteria.
func PrintCheckResults(out io.Writer, checks []CheckPresentation) {
	color := ColorEnabled(out)
	total, passed := len(checks), 0
	for _, check := range checks {
		if check.Passed {
			passed++
		}
	}
	summaryTone := ToneSuccess
	if passed < total {
		summaryTone = ToneWarning
	}
	fmt.Fprintln(out)
	fmt.Fprintln(out, Paint(color, summaryTone, fmt.Sprintf("BILAN  %d/%d OK  ·  %d à corriger", passed, total, total-passed)))
	for _, check := range checks {
		if check.Passed {
			continue
		}
		fmt.Fprintln(out, Paint(color, ToneWarning, "  À CORRIGER")+"  "+WrapParagraph(singleLine(check.Label), TerminalWidth(out)-15, "             "))
		if check.Diagnostic != "" {
			fmt.Fprintln(out, Paint(color, ToneMuted, "     ↳ "+WrapParagraph(singleLine(check.Diagnostic), TerminalWidth(out)-8, "       ")))
		}
	}
	// Small checks show every success. Large batches collapse green lines to reduce noise.
	printed := 0
	for _, check := range checks {
		if !check.Passed || (passed > 4 && printed >= 2) {
			continue
		}
		fmt.Fprintln(out, Paint(color, ToneSuccess, "  OK")+"          "+WrapParagraph(singleLine(check.Label), TerminalWidth(out)-15, "              "))
		printed++
	}
	if passed > printed {
		fmt.Fprintf(out, "  %s\n", Paint(color, ToneMuted, fmt.Sprintf("+ %d autres critères OK", passed-printed)))
	}
	if passed < total {
		fmt.Fprintln(out, Paint(color, ToneMuted, "Corrige les points signalés, puis relance :check."))
	}
}

// The parent terminal receives a plain-text report over a private socket.
// Add colors only at the final display boundary, respecting NO_COLOR and pipes.
func PrintCheckOutput(out io.Writer, report string) {
	color := ColorEnabled(out)
	for _, raw := range strings.SplitAfter(SanitizeText(report), "\n") {
		line := strings.TrimSuffix(raw, "\n")
		tone := ToneMuted
		switch {
		case strings.Contains(line, "À CORRIGER"), strings.Contains(line, "à corriger"), strings.Contains(line, "interrompue"):
			tone = ToneWarning
		case strings.HasPrefix(line, "  OK"), strings.Contains(line, "BILAN  "), strings.Contains(line, "Lab réussi"):
			tone = ToneSuccess
		default:
			fmt.Fprint(out, raw)
			continue
		}
		fmt.Fprint(out, Paint(color, tone, line))
		if strings.HasSuffix(raw, "\n") {
			fmt.Fprintln(out)
		}
	}
}
