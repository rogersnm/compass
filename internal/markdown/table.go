package markdown

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/rogersnm/compass/internal/model"
)

var (
	headerRowStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	cellStyle      = lipgloss.NewStyle()
)

func RenderProjectTable(projects []model.Project) string {
	if len(projects) == 0 {
		return "No projects found."
	}
	rows := make([][]string, len(projects))
	for i, p := range projects {
		rows[i] = []string{p.ID, p.Name, p.CreatedAt.Format("2006-01-02")}
	}
	return renderTable([]string{"ID", "Name", "Created"}, rows)
}

func RenderDocumentTable(docs []model.Document) string {
	if len(docs) == 0 {
		return "No documents found."
	}
	rows := make([][]string, len(docs))
	for i, d := range docs {
		rows[i] = []string{d.ID, d.Title, d.Project, d.CreatedAt.Format("2006-01-02")}
	}
	return renderTable([]string{"ID", "Title", "Project", "Created"}, rows)
}

func RenderTaskTable(tasks []model.Task, allTasks map[string]*model.Task) string {
	if len(tasks) == 0 {
		return "No tasks found."
	}
	rows := make([][]string, len(tasks))
	for i, t := range tasks {
		status := RenderStatus(string(t.Status), t.IsBlocked(allTasks))
		rows[i] = []string{t.ID, t.Title, model.FormatPriority(t.Priority), status, t.Project}
	}
	return renderTable([]string{"ID", "Title", "Pri", "Status", "Project"}, rows)
}

func renderTable(headers []string, rows [][]string) string {
	t := table.New().
		Headers(headers...).
		Rows(rows...).
		BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("8"))).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return headerRowStyle
			}
			return cellStyle
		})
	return t.Render()
}
