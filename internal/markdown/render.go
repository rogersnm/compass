package markdown

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/glamour/styles"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
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

func autoStyle() ansi.StyleConfig {
	if termenv.HasDarkBackground() {
		return styles.DarkStyleConfig
	}
	return styles.LightStyleConfig
}

func RenderMarkdown(content string) (string, error) {
	style := autoStyle()
	margin := 0
	if style.Document.Margin != nil {
		margin = int(*style.Document.Margin)
	}
	r, err := glamour.NewTermRenderer(glamour.WithStyles(style), glamour.WithWordWrap(termWidth()-margin))
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
