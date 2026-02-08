package id

import (
	"crypto/rand"
	"fmt"
	"strings"
)

const charset = "23456789ABCDEFGHJKMNPQRSTUVWXYZ"
const hashLen = 5

type EntityType string

const (
	Project  EntityType = "PROJ"
	Document EntityType = "DOC"
	Task     EntityType = "TASK"
)

var allTypes = []EntityType{Project, Document, Task}

func New(t EntityType) (string, error) {
	b := make([]byte, hashLen)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating id: %w", err)
	}
	for i := range b {
		b[i] = charset[int(b[i])%len(charset)]
	}
	return string(t) + "-" + string(b), nil
}

func Parse(id string) (EntityType, string, error) {
	idx := strings.LastIndex(id, "-")
	if idx < 0 {
		return "", "", fmt.Errorf("invalid id %q: missing separator", id)
	}
	prefix := id[:idx]
	hash := id[idx+1:]

	t, err := parsePrefix(prefix)
	if err != nil {
		return "", "", err
	}

	if len(hash) != hashLen {
		return "", "", fmt.Errorf("invalid id %q: hash must be %d chars", id, hashLen)
	}
	for _, c := range hash {
		if !strings.ContainsRune(charset, c) {
			return "", "", fmt.Errorf("invalid id %q: invalid character %q", id, c)
		}
	}
	return t, hash, nil
}

func TypeOf(id string) (EntityType, error) {
	t, _, err := Parse(id)
	return t, err
}

func parsePrefix(prefix string) (EntityType, error) {
	for _, t := range allTypes {
		if string(t) == prefix {
			return t, nil
		}
	}
	return "", fmt.Errorf("unknown entity prefix %q", prefix)
}
