package id

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var validPattern = regexp.MustCompile(`^(PROJ|DOC|TASK)-[23456789ABCDEFGHJKMNPQRSTUVWXYZ]{5}$`)

func TestNew_Format(t *testing.T) {
	for _, et := range allTypes {
		id, err := New(et)
		require.NoError(t, err)
		assert.Regexp(t, validPattern, id)
		assert.True(t, len(id) > 0)
	}
}

func TestNew_AllEntityTypes(t *testing.T) {
	tests := []struct {
		et     EntityType
		prefix string
	}{
		{Project, "PROJ-"},
		{Document, "DOC-"},
		{Task, "TASK-"},
	}
	for _, tt := range tests {
		id, err := New(tt.et)
		require.NoError(t, err)
		assert.Contains(t, id, tt.prefix)
	}
}

func TestNew_Uniqueness(t *testing.T) {
	for _, et := range allTypes {
		seen := make(map[string]bool)
		for i := 0; i < 1000; i++ {
			id, err := New(et)
			require.NoError(t, err)
			assert.False(t, seen[id], "collision: %s", id)
			seen[id] = true
		}
	}
}

func TestParse_Valid(t *testing.T) {
	for _, et := range allTypes {
		id, err := New(et)
		require.NoError(t, err)

		gotType, gotHash, err := Parse(id)
		require.NoError(t, err)
		assert.Equal(t, et, gotType)
		assert.Len(t, gotHash, hashLen)
	}
}

func TestParse_InvalidFormat(t *testing.T) {
	tests := []string{
		"",
		"PROJ",
		"PROJ-",
		"PROJ-AB",     // too short
		"PROJ-00000",  // 0 not in charset
		"PROJ-1ABCD",  // 1 not in charset
		"PROJ-OABCD",  // O not in charset
		"PROJ-IABCD",  // I not in charset
		"PROJ-LABCD",  // L not in charset
		"XXX-ABCDE",
		"proj-ABCDE",
		"PROJ-ABCDEF", // too long
		"EPIC-ABCDE",  // no longer valid
	}
	for _, id := range tests {
		_, _, err := Parse(id)
		assert.Error(t, err, "expected error for %q", id)
	}
}

func TestTypeOf_AllTypes(t *testing.T) {
	for _, et := range allTypes {
		id, err := New(et)
		require.NoError(t, err)

		got, err := TypeOf(id)
		require.NoError(t, err)
		assert.Equal(t, et, got)
	}
}
