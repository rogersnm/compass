package id

import (
	"crypto/rand"
	"fmt"
	"strings"
	"unicode"
)

const charset = "23456789ABCDEFGHJKMNPQRSTUVWXYZ"
const hashLen = 5

type EntityType string

const (
	Project  EntityType = "project"
	Document EntityType = "document"
	Task     EntityType = "task"
)

// GenerateKey produces a project key from a name: strip non-alpha, uppercase, first 4 chars.
// Returns error if fewer than 2 alpha chars remain.
func GenerateKey(name string) (string, error) {
	var buf []rune
	for _, r := range name {
		if unicode.IsLetter(r) {
			buf = append(buf, unicode.ToUpper(r))
		}
		if len(buf) == 4 {
			break
		}
	}
	if len(buf) < 2 {
		return "", fmt.Errorf("cannot auto-generate key from %q: need at least 2 alpha characters (use --key)", name)
	}
	return string(buf), nil
}

// ValidateKey checks that a key is 2-5 uppercase alphanumeric chars with no dashes.
func ValidateKey(key string) error {
	if len(key) < 2 || len(key) > 5 {
		return fmt.Errorf("invalid key %q: must be 2-5 characters", key)
	}
	for _, r := range key {
		if !unicode.IsUpper(r) && !unicode.IsDigit(r) {
			return fmt.Errorf("invalid key %q: must be uppercase alphanumeric (no dashes)", key)
		}
	}
	return nil
}

func randomHash() (string, error) {
	b := make([]byte, hashLen)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating id: %w", err)
	}
	for i := range b {
		b[i] = charset[int(b[i])%len(charset)]
	}
	return string(b), nil
}

// NewTaskID returns a task ID like "AUTH-TABCDE".
func NewTaskID(projectKey string) (string, error) {
	if err := ValidateKey(projectKey); err != nil {
		return "", err
	}
	h, err := randomHash()
	if err != nil {
		return "", err
	}
	return projectKey + "-T" + h, nil
}

// NewDocID returns a document ID like "AUTH-DABCDE".
func NewDocID(projectKey string) (string, error) {
	if err := ValidateKey(projectKey); err != nil {
		return "", err
	}
	h, err := randomHash()
	if err != nil {
		return "", err
	}
	return projectKey + "-D" + h, nil
}

// Parse parses an ID into (projectKey, entityType, hash, error).
// No dash = project key. With dash: suffix must be 6 chars (T/D + 5 hash chars).
func Parse(id string) (string, EntityType, string, error) {
	idx := strings.LastIndex(id, "-")
	if idx < 0 {
		// Bare key = project
		if err := ValidateKey(id); err != nil {
			return "", "", "", fmt.Errorf("invalid id %q: %w", id, err)
		}
		return id, Project, "", nil
	}

	key := id[:idx]
	suffix := id[idx+1:]

	if err := ValidateKey(key); err != nil {
		return "", "", "", fmt.Errorf("invalid id %q: bad key: %w", id, err)
	}

	if len(suffix) != hashLen+1 {
		return "", "", "", fmt.Errorf("invalid id %q: suffix must be %d chars (type indicator + %d hash)", id, hashLen+1, hashLen)
	}

	typeChar := suffix[0]
	hash := suffix[1:]

	var t EntityType
	switch typeChar {
	case 'T':
		t = Task
	case 'D':
		t = Document
	default:
		return "", "", "", fmt.Errorf("invalid id %q: unknown type indicator %q", id, string(typeChar))
	}

	for _, c := range hash {
		if !strings.ContainsRune(charset, c) {
			return "", "", "", fmt.Errorf("invalid id %q: invalid character %q in hash", id, c)
		}
	}

	return key, t, hash, nil
}

// TypeOf returns the entity type for an ID.
func TypeOf(id string) (EntityType, error) {
	_, t, _, err := Parse(id)
	return t, err
}

// ProjectKeyFrom extracts the project key from any ID (task, doc, or bare project key).
func ProjectKeyFrom(id string) (string, error) {
	key, _, _, err := Parse(id)
	return key, err
}
