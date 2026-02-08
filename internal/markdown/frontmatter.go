package markdown

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/adrg/frontmatter"
	"gopkg.in/yaml.v3"
)

// Parse reads YAML frontmatter and body from r into T.
func Parse[T any](r io.Reader) (T, string, error) {
	var meta T
	body, err := frontmatter.Parse(r, &meta)
	if err != nil {
		return meta, "", fmt.Errorf("parsing frontmatter: %w", err)
	}
	return meta, strings.TrimSpace(string(body)), nil
}

// Marshal serializes meta as YAML frontmatter followed by body.
func Marshal[T any](meta T, body string) ([]byte, error) {
	yamlBytes, err := yaml.Marshal(meta)
	if err != nil {
		return nil, fmt.Errorf("marshaling frontmatter: %w", err)
	}

	var buf bytes.Buffer
	buf.WriteString("---\n")
	buf.Write(yamlBytes)
	buf.WriteString("---\n")
	if body != "" {
		buf.WriteString("\n")
		buf.WriteString(body)
		if !strings.HasSuffix(body, "\n") {
			buf.WriteString("\n")
		}
	}
	return buf.Bytes(), nil
}
