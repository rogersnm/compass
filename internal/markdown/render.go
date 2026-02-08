package markdown

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

var (
	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	labelStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	openStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	inProgStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	closedSty   = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	blockedSty  = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
)

func RenderMarkdown(content string) (string, error) {
	r, err := glamour.NewTermRenderer(glamour.WithAutoStyle())
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
