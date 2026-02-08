package dag

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/rogersnm/compass/internal/model"
)

var (
	openStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))  // white
	inProgressStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("11")) // yellow
	closedStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("10")) // green
	blockedStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))  // red
)

func statusStyle(t *model.Task, allTasks map[string]*model.Task) lipgloss.Style {
	if t.IsBlocked(allTasks) {
		return blockedStyle
	}
	switch t.Status {
	case model.StatusClosed:
		return closedStyle
	case model.StatusInProgress:
		return inProgressStyle
	default:
		return openStyle
	}
}

// RenderASCII produces an ASCII tree visualization of the task DAG.
func RenderASCII(g *Graph) string {
	if len(g.nodes) == 0 {
		return "No tasks."
	}

	allTasks := g.nodes
	roots := g.Roots()
	if len(roots) == 0 {
		return "No root tasks (all tasks have dependencies)."
	}

	visited := make(map[string]bool)
	var sb strings.Builder

	for i, root := range roots {
		if i > 0 {
			sb.WriteString("\n")
		}
		renderNode(&sb, g, root, "", true, visited, allTasks)
	}

	return sb.String()
}

func renderNode(sb *strings.Builder, g *Graph, id, prefix string, isLast bool, visited map[string]bool, allTasks map[string]*model.Task) {
	t := g.nodes[id]
	if t == nil {
		return
	}

	connector := "├── "
	if isLast {
		connector = "└── "
	}
	if prefix == "" {
		connector = ""
	}

	style := statusStyle(t, allTasks)
	statusStr := string(t.Status)
	if t.IsBlocked(allTasks) {
		statusStr = fmt.Sprintf("%s (blocked)", t.Status)
	}

	label := style.Render(fmt.Sprintf("%s %s [%s]", t.ID, t.Title, statusStr))

	if visited[id] {
		sb.WriteString(prefix + connector + label + " (see above)\n")
		return
	}
	visited[id] = true

	sb.WriteString(prefix + connector + label + "\n")

	children := g.Dependents(id)
	sort.Strings(children)

	childPrefix := prefix
	if prefix == "" {
		childPrefix = "    "
	} else if isLast {
		childPrefix += "    "
	} else {
		childPrefix += "│   "
	}

	for i, child := range children {
		renderNode(sb, g, child, childPrefix, i == len(children)-1, visited, allTasks)
	}
}
