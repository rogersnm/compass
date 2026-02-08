package id

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateKey_BasicName(t *testing.T) {
	key, err := GenerateKey("Authentication Service")
	require.NoError(t, err)
	assert.Equal(t, "AUTH", key)
}

func TestGenerateKey_ShortName(t *testing.T) {
	key, err := GenerateKey("API")
	require.NoError(t, err)
	assert.Equal(t, "API", key)
}

func TestGenerateKey_TwoCharName(t *testing.T) {
	key, err := GenerateKey("Go")
	require.NoError(t, err)
	assert.Equal(t, "GO", key)
}

func TestGenerateKey_StripNonAlpha(t *testing.T) {
	key, err := GenerateKey("my-cool-project 123!")
	require.NoError(t, err)
	assert.Equal(t, "MYCO", key)
}

func TestGenerateKey_TooShort(t *testing.T) {
	_, err := GenerateKey("X")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "need at least 2 alpha")
}

func TestGenerateKey_NumbersOnly(t *testing.T) {
	_, err := GenerateKey("12345")
	assert.Error(t, err)
}

func TestGenerateKey_Empty(t *testing.T) {
	_, err := GenerateKey("")
	assert.Error(t, err)
}

func TestValidateKey_Valid(t *testing.T) {
	for _, key := range []string{"AB", "AUTH", "AUTH2", "ABCDE", "A2B3C"} {
		assert.NoError(t, ValidateKey(key), "expected valid: %s", key)
	}
}

func TestValidateKey_TooShort(t *testing.T) {
	assert.Error(t, ValidateKey("A"))
}

func TestValidateKey_TooLong(t *testing.T) {
	assert.Error(t, ValidateKey("ABCDEF"))
}

func TestValidateKey_Lowercase(t *testing.T) {
	assert.Error(t, ValidateKey("auth"))
}

func TestValidateKey_HasDash(t *testing.T) {
	assert.Error(t, ValidateKey("AU-TH"))
}

var taskPattern = regexp.MustCompile(`^[A-Z0-9]{2,5}-T[23456789ABCDEFGHJKMNPQRSTUVWXYZ]{5}$`)
var docPattern = regexp.MustCompile(`^[A-Z0-9]{2,5}-D[23456789ABCDEFGHJKMNPQRSTUVWXYZ]{5}$`)

func TestNewTaskID_Format(t *testing.T) {
	id, err := NewTaskID("AUTH")
	require.NoError(t, err)
	assert.Regexp(t, taskPattern, id)
}

func TestNewDocID_Format(t *testing.T) {
	id, err := NewDocID("AUTH")
	require.NoError(t, err)
	assert.Regexp(t, docPattern, id)
}

func TestNewTaskID_Uniqueness(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		id, err := NewTaskID("TEST")
		require.NoError(t, err)
		assert.False(t, seen[id], "collision: %s", id)
		seen[id] = true
	}
}

func TestNewDocID_Uniqueness(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		id, err := NewDocID("TEST")
		require.NoError(t, err)
		assert.False(t, seen[id], "collision: %s", id)
		seen[id] = true
	}
}

func TestNewTaskID_InvalidKey(t *testing.T) {
	_, err := NewTaskID("x")
	assert.Error(t, err)
}

func TestParse_ProjectKey(t *testing.T) {
	key, typ, hash, err := Parse("AUTH")
	require.NoError(t, err)
	assert.Equal(t, "AUTH", key)
	assert.Equal(t, Project, typ)
	assert.Equal(t, "", hash)
}

func TestParse_TaskID(t *testing.T) {
	key, typ, hash, err := Parse("AUTH-TABCDE")
	require.NoError(t, err)
	assert.Equal(t, "AUTH", key)
	assert.Equal(t, Task, typ)
	assert.Equal(t, "ABCDE", hash)
}

func TestParse_DocID(t *testing.T) {
	key, typ, hash, err := Parse("AUTH-DABCDE")
	require.NoError(t, err)
	assert.Equal(t, "AUTH", key)
	assert.Equal(t, Document, typ)
	assert.Equal(t, "ABCDE", hash)
}

func TestParse_KeyWithDigit(t *testing.T) {
	key, typ, _, err := Parse("AUTH2-T23456")
	require.NoError(t, err)
	assert.Equal(t, "AUTH2", key)
	assert.Equal(t, Task, typ)
}

func TestParse_Invalid(t *testing.T) {
	tests := []string{
		"",
		"A",               // key too short (bare)
		"a",               // lowercase
		"AUTH-",           // empty suffix
		"AUTH-T",          // suffix too short
		"AUTH-TABCDEF",    // suffix too long
		"AUTH-XABCDE",    // unknown type indicator
		"AUTH-T00000",    // 0 not in charset
		"AUTH-TABCD1",    // 1 not in charset
		"AUTH-TABCDO",    // O not in charset
		"AUTH-TABCDI",    // I not in charset
		"AUTH-TABCDL",    // L not in charset
		"auth-TABCDE",    // lowercase key
		"ABCDEF-TABCDE",  // key too long
	}
	for _, id := range tests {
		_, _, _, err := Parse(id)
		assert.Error(t, err, "expected error for %q", id)
	}
}

func TestParse_RoundTrip_Task(t *testing.T) {
	id, err := NewTaskID("AUTH")
	require.NoError(t, err)

	key, typ, hash, err := Parse(id)
	require.NoError(t, err)
	assert.Equal(t, "AUTH", key)
	assert.Equal(t, Task, typ)
	assert.Len(t, hash, hashLen)
}

func TestParse_RoundTrip_Doc(t *testing.T) {
	id, err := NewDocID("AUTH")
	require.NoError(t, err)

	key, typ, hash, err := Parse(id)
	require.NoError(t, err)
	assert.Equal(t, "AUTH", key)
	assert.Equal(t, Document, typ)
	assert.Len(t, hash, hashLen)
}

func TestTypeOf(t *testing.T) {
	typ, err := TypeOf("AUTH")
	require.NoError(t, err)
	assert.Equal(t, Project, typ)

	typ, err = TypeOf("AUTH-TABCDE")
	require.NoError(t, err)
	assert.Equal(t, Task, typ)

	typ, err = TypeOf("AUTH-DABCDE")
	require.NoError(t, err)
	assert.Equal(t, Document, typ)
}

func TestProjectKeyFrom_TaskID(t *testing.T) {
	key, err := ProjectKeyFrom("AUTH-TABCDE")
	require.NoError(t, err)
	assert.Equal(t, "AUTH", key)
}

func TestProjectKeyFrom_ProjectKey(t *testing.T) {
	key, err := ProjectKeyFrom("AUTH")
	require.NoError(t, err)
	assert.Equal(t, "AUTH", key)
}

func TestProjectKeyFrom_DocID(t *testing.T) {
	key, err := ProjectKeyFrom("AUTH-DABCDE")
	require.NoError(t, err)
	assert.Equal(t, "AUTH", key)
}
