package confy

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseDotEnv(t *testing.T) {
	content := `DB_HOST=localhost
DB_PORT=5432
# comment line
EMPTY_VALUE=
QUOTED="hello world"
SINGLE_QUOTED='single'
`
	envs, err := parseDotEnv([]byte(content))
	if err != nil {
		t.Fatal(err)
	}

	tests := map[string]string{
		"DB_HOST":       "localhost",
		"DB_PORT":       "5432",
		"EMPTY_VALUE":   "",
		"QUOTED":        "hello world",
		"SINGLE_QUOTED": "single",
	}

	for key, expected := range tests {
		if envs[key] != expected {
			t.Errorf("expected %s='%s', got '%s'", key, expected, envs[key])
		}
	}
}

func TestLoadDotEnv(t *testing.T) {
	dir := t.TempDir()
	envFile := filepath.Join(dir, ".env")
	os.WriteFile(envFile, []byte("TEST_KEY=test_value\n"), 0644)

	old := os.Getenv("TEST_KEY")
	defer os.Setenv("TEST_KEY", old)

	err := loadDotEnv(envFile)
	if err != nil {
		t.Fatal(err)
	}

	if os.Getenv("TEST_KEY") != "test_value" {
		t.Errorf("expected TEST_KEY='test_value', got '%s'", os.Getenv("TEST_KEY"))
	}
}

func TestLoadDotEnv_FileNotExist(t *testing.T) {
	err := loadDotEnv("/nonexistent/.env")
	if err != nil {
		t.Errorf("expected no error for missing file, got: %v", err)
	}
}

func TestParseDotEnv_IgnoresComments(t *testing.T) {
	content := `# This is a comment
KEY=value
# Another comment
`
	envs, err := parseDotEnv([]byte(content))
	if err != nil {
		t.Fatal(err)
	}
	if len(envs) != 1 {
		t.Errorf("expected 1 entry, got %d", len(envs))
	}
}
