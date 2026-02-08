package markdown

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testMeta struct {
	ID        string    `yaml:"id"`
	Title     string    `yaml:"title"`
	Status    string    `yaml:"status,omitempty"`
	DependsOn []string  `yaml:"depends_on,omitempty"`
	CreatedAt time.Time `yaml:"created_at"`
}

func TestParse_AllFields(t *testing.T) {
	input := `---
id: TASK-ABCDE
title: "Test Task"
status: open
depends_on:
  - TASK-11111
  - TASK-22222
created_at: 2026-01-01T00:00:00Z
---

This is the body.
`
	meta, body, err := Parse[testMeta](strings.NewReader(input))
	require.NoError(t, err)
	assert.Equal(t, "TASK-ABCDE", meta.ID)
	assert.Equal(t, "Test Task", meta.Title)
	assert.Equal(t, "open", meta.Status)
	assert.Equal(t, []string{"TASK-11111", "TASK-22222"}, meta.DependsOn)
	assert.Equal(t, "This is the body.", body)
}

func TestParse_EmptyBody(t *testing.T) {
	input := `---
id: DOC-ABCDE
title: "Empty Doc"
created_at: 2026-01-01T00:00:00Z
---
`
	meta, body, err := Parse[testMeta](strings.NewReader(input))
	require.NoError(t, err)
	assert.Equal(t, "DOC-ABCDE", meta.ID)
	assert.Equal(t, "", body)
}

func TestParse_NoFrontmatter(t *testing.T) {
	input := "Just some plain markdown."
	meta, body, err := Parse[testMeta](strings.NewReader(input))
	// adrg/frontmatter returns empty struct when no frontmatter found
	require.NoError(t, err)
	assert.Equal(t, "", meta.ID)
	assert.Equal(t, "Just some plain markdown.", body)
}

func TestParse_MalformedYAML(t *testing.T) {
	input := "---\n{{invalid yaml\n---\n"
	_, _, err := Parse[testMeta](strings.NewReader(input))
	assert.Error(t, err)
}

func TestMarshal_RoundTrip(t *testing.T) {
	original := testMeta{
		ID:        "TASK-ABCDE",
		Title:     "Round Trip",
		Status:    "open",
		DependsOn: []string{"TASK-11111"},
		CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	body := "Some body content."

	data, err := Marshal(original, body)
	require.NoError(t, err)

	parsed, parsedBody, err := Parse[testMeta](strings.NewReader(string(data)))
	require.NoError(t, err)
	assert.Equal(t, original.ID, parsed.ID)
	assert.Equal(t, original.Title, parsed.Title)
	assert.Equal(t, original.Status, parsed.Status)
	assert.Equal(t, original.DependsOn, parsed.DependsOn)
	assert.Equal(t, body, parsedBody)
}

func TestMarshal_EmptyBody(t *testing.T) {
	meta := testMeta{ID: "DOC-ABCDE", Title: "No Body"}
	data, err := Marshal(meta, "")
	require.NoError(t, err)

	parsed, body, err := Parse[testMeta](strings.NewReader(string(data)))
	require.NoError(t, err)
	assert.Equal(t, meta.ID, parsed.ID)
	assert.Equal(t, "", body)
}

func TestMarshal_PreservesBody(t *testing.T) {
	body := "Line 1\n\n```go\nfunc main() {}\n```\n\n**Bold** and *italic*"
	meta := testMeta{ID: "DOC-ABCDE", Title: "Code"}
	data, err := Marshal(meta, body)
	require.NoError(t, err)

	_, parsedBody, err := Parse[testMeta](strings.NewReader(string(data)))
	require.NoError(t, err)
	assert.Equal(t, body, parsedBody)
}
