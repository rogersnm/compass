package markdown

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

var (
	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	labelStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	openStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	inProgStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	closedSty   = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	blockedSty  = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
)

func termWidth() int {
	if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 0 {
		return w
	}
	return 80
}

func RenderMarkdown(content string) (string, error) {
	// glamour's default styles add a 2-char left margin to the document block;
	// subtract it so rendered output fits the terminal without overflow.
	r, err := glamour.NewTermRenderer(glamour.WithAutoStyle(), glamour.WithWordWrap(termWidth()-2))
	if err != nil {
		return "", fmt.Errorf("creating renderer: %w", err)
	}
	out, err := r.Render(content)
	if err != nil {
		return "", fmt.Errorf("rendering markdown: %w", err)
	}
	return out, nil
}

func StatusStyle(status string) lipgloss.Style {
	switch status {
	case "closed":
		return closedSty
	case "in_progress":
		return inProgStyle
	default:
		return openStyle
	}
}

func RenderField(label, value string) string {
	return labelStyle.Render(label+":") + " " + value
}

func RenderStatus(status string, blocked bool) string {
	s := StatusStyle(status).Render(status)
	if blocked {
		s += " " + blockedSty.Render("(blocked)")
	}
	return s
}

func RenderEntityHeader(title string, fields []string) string {
	var sb strings.Builder
	sb.WriteString(headerStyle.Render(title))
	sb.WriteString("\n")
	for _, f := range fields {
		sb.WriteString("  " + f + "\n")
	}
	return sb.String()
}
