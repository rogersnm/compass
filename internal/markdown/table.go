package markdown

import (
	"sort"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/rogersnm/compass/internal/model"
)

var (
	headerRowStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	cellStyle      = lipgloss.NewStyle()
)

// ProjectRow pairs a project with its store name for multi-store display.
type ProjectRow struct {
	Project   model.Project
	StoreName string
}

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

func RenderProjectTableWithStores(projectRows []ProjectRow) string {
	if len(projectRows) == 0 {
		return "No projects found."
	}
	sort.Slice(projectRows, func(i, j int) bool {
		return projectRows[i].Project.ID < projectRows[j].Project.ID
	})
	rows := make([][]string, len(projectRows))
	for i, r := range projectRows {
		rows[i] = []string{r.Project.ID, r.Project.Name, r.StoreName, r.Project.CreatedAt.Format("2006-01-02")}
	}
	return renderTable([]string{"ID", "Name", "Store", "Created"}, rows)
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
	sort.Slice(tasks, func(i, j int) bool {
		bi := tasks[i].IsBlocked(allTasks)
		bj := tasks[j].IsBlocked(allTasks)
		if bi != bj {
			return !bi // unblocked first
		}
		return tasks[i].CreatedAt.Before(tasks[j].CreatedAt)
	})
	rows := make([][]string, len(tasks))
	for i, t := range tasks {
		var status string
		if t.Type == model.TypeEpic {
			status = "N/A"
		} else {
			status = RenderStatus(string(t.Status), t.IsBlocked(allTasks))
		}
		rows[i] = []string{t.ID, t.Title, string(t.Type), model.FormatPriority(t.Priority), status, t.Project}
	}
	return renderTable([]string{"ID", "Title", "Type", "Pri", "Status", "Project"}, rows)
}

func RenderStoreTable(rows [][]string) string {
	if len(rows) == 0 {
		return "No stores configured."
	}
	return renderTable([]string{"Store", "Default"}, rows)
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
