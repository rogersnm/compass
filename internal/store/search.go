package store

import (
	"strings"
)

type SearchResult struct {
	Type    string
	ID      string
	Title   string
	Snippet string
}

func (s *Store) Search(query, projectID string) ([]SearchResult, error) {
	q := strings.ToLower(query)
	var results []SearchResult

	projects, err := s.ListProjects()
	if err != nil {
		return nil, err
	}
	for _, p := range projects {
		if projectID != "" && p.ID != projectID {
			continue
		}
		if matchesQuery(q, p.Name) {
			results = append(results, SearchResult{Type: "project", ID: p.ID, Title: p.Name})
		}
		// Also search project body
		if _, body, err := s.GetProject(p.ID); err == nil && matchesQuery(q, body) {
			if !hasResult(results, p.ID) {
				results = append(results, SearchResult{
					Type: "project", ID: p.ID, Title: p.Name,
					Snippet: snippet(body, q),
				})
			}
		}
	}

	filterProject := projectID
	docs, err := s.ListDocuments(filterProject)
	if err != nil {
		return nil, err
	}
	for _, d := range docs {
		if matchesQuery(q, d.Title) {
			results = append(results, SearchResult{Type: "document", ID: d.ID, Title: d.Title})
		}
		if _, body, err := s.GetDocument(d.ID); err == nil && matchesQuery(q, body) {
			if !hasResult(results, d.ID) {
				results = append(results, SearchResult{
					Type: "document", ID: d.ID, Title: d.Title,
					Snippet: snippet(body, q),
				})
			}
		}
	}

	epics, err := s.ListEpics(filterProject)
	if err != nil {
		return nil, err
	}
	for _, e := range epics {
		if matchesQuery(q, e.Title) {
			results = append(results, SearchResult{Type: "epic", ID: e.ID, Title: e.Title})
		}
		if _, body, err := s.GetEpic(e.ID); err == nil && matchesQuery(q, body) {
			if !hasResult(results, e.ID) {
				results = append(results, SearchResult{
					Type: "epic", ID: e.ID, Title: e.Title,
					Snippet: snippet(body, q),
				})
			}
		}
	}

	tasks, err := s.ListTasks(TaskFilter{ProjectID: filterProject})
	if err != nil {
		return nil, err
	}
	for _, t := range tasks {
		if matchesQuery(q, t.Title) {
			results = append(results, SearchResult{Type: "task", ID: t.ID, Title: t.Title})
		}
		if _, body, err := s.GetTask(t.ID); err == nil && matchesQuery(q, body) {
			if !hasResult(results, t.ID) {
				results = append(results, SearchResult{
					Type: "task", ID: t.ID, Title: t.Title,
					Snippet: snippet(body, q),
				})
			}
		}
	}

	return results, nil
}

func matchesQuery(q, text string) bool {
	return strings.Contains(strings.ToLower(text), q)
}

func hasResult(results []SearchResult, id string) bool {
	for _, r := range results {
		if r.ID == id {
			return true
		}
	}
	return false
}

func snippet(body, query string) string {
	lower := strings.ToLower(body)
	idx := strings.Index(lower, query)
	if idx < 0 {
		return ""
	}
	start := idx - 40
	if start < 0 {
		start = 0
	}
	end := idx + len(query) + 40
	if end > len(body) {
		end = len(body)
	}
	s := body[start:end]
	if start > 0 {
		s = "..." + s
	}
	if end < len(body) {
		s = s + "..."
	}
	return strings.ReplaceAll(s, "\n", " ")
}
