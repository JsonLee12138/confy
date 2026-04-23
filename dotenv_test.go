package confy

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/JsonLee12138/confy/internal"
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

func TestGetDotEnvFilePaths_DevMode(t *testing.T) {
	dir := t.TempDir()
	internal.ResetMode()
	t.Setenv("GO_ENV_MODE", "development")

	writeDotEnvFile(t, dir, ".env", "BASE=1")
	writeDotEnvFile(t, dir, ".env.local", "LOCAL=1")
	writeDotEnvFile(t, dir, ".env.development", "DEV=1")
	writeDotEnvFile(t, dir, ".env.development.local", "DEV_LOCAL=1")

	paths := getDotEnvFilePaths(dir)
	if len(paths) != 4 {
		t.Fatalf("expected 4 paths, got %d: %v", len(paths), paths)
	}

	expected := []string{".env", ".env.local", ".env.development", ".env.development.local"}
	for i, name := range expected {
		if filepath.Base(paths[i]) != name {
			t.Errorf("expected paths[%d] = %s, got %s", i, name, filepath.Base(paths[i]))
		}
	}
}

func TestGetDotEnvFilePaths_ProdMode(t *testing.T) {
	dir := t.TempDir()
	internal.ResetMode()
	t.Setenv("GO_ENV_MODE", "production")

	writeDotEnvFile(t, dir, ".env", "BASE=1")
	writeDotEnvFile(t, dir, ".env.production", "PROD=1")
	writeDotEnvFile(t, dir, ".env.production.local", "PROD_LOCAL=1")

	paths := getDotEnvFilePaths(dir)
	if len(paths) != 3 {
		t.Fatalf("expected 3 paths, got %d: %v", len(paths), paths)
	}

	expected := []string{".env", ".env.production", ".env.production.local"}
	for i, name := range expected {
		if filepath.Base(paths[i]) != name {
			t.Errorf("expected paths[%d] = %s, got %s", i, name, filepath.Base(paths[i]))
		}
	}
}

func TestGetDotEnvFilePaths_TestMode(t *testing.T) {
	dir := t.TempDir()
	internal.ResetMode()
	t.Setenv("GO_ENV_MODE", "test")

	writeDotEnvFile(t, dir, ".env", "BASE=1")
	writeDotEnvFile(t, dir, ".env.test", "TEST=1")
	writeDotEnvFile(t, dir, ".env.test.local", "TEST_LOCAL=1")

	paths := getDotEnvFilePaths(dir)
	if len(paths) != 3 {
		t.Fatalf("expected 3 paths, got %d: %v", len(paths), paths)
	}

	expected := []string{".env", ".env.test", ".env.test.local"}
	for i, name := range expected {
		if filepath.Base(paths[i]) != name {
			t.Errorf("expected paths[%d] = %s, got %s", i, name, filepath.Base(paths[i]))
		}
	}
}

func TestGetDotEnvFilePaths_SkipsMissingFiles(t *testing.T) {
	dir := t.TempDir()
	internal.ResetMode()
	t.Setenv("GO_ENV_MODE", "production")

	// Only create .env and .env.production, skip .env.local and .env.production.local
	writeDotEnvFile(t, dir, ".env", "BASE=1")
	writeDotEnvFile(t, dir, ".env.production", "PROD=1")

	paths := getDotEnvFilePaths(dir)
	if len(paths) != 2 {
		t.Fatalf("expected 2 paths, got %d: %v", len(paths), paths)
	}
}

func TestGetDotEnvFilePaths_NoFiles(t *testing.T) {
	dir := t.TempDir()
	internal.ResetMode()
	t.Setenv("GO_ENV_MODE", "development")

	paths := getDotEnvFilePaths(dir)
	if len(paths) != 0 {
		t.Errorf("expected 0 paths, got %d", len(paths))
	}
}

func TestGetDotEnvFilePaths_DevAlias(t *testing.T) {
	dir := t.TempDir()
	internal.ResetMode()
	t.Setenv("GO_ENV_MODE", "dev")

	// .env.dev should be discovered when GO_ENV_MODE=dev
	writeDotEnvFile(t, dir, ".env", "BASE=1")
	writeDotEnvFile(t, dir, ".env.dev", "DEV=1")

	paths := getDotEnvFilePaths(dir)
	if len(paths) != 2 {
		t.Fatalf("expected 2 paths, got %d: %v", len(paths), paths)
	}

	if filepath.Base(paths[1]) != ".env.dev" {
		t.Errorf("expected .env.dev, got %s", filepath.Base(paths[1]))
	}
}

func TestGetDotEnvFilePaths_ProdAlias(t *testing.T) {
	dir := t.TempDir()
	internal.ResetMode()
	t.Setenv("GO_ENV_MODE", "prod")

	writeDotEnvFile(t, dir, ".env", "BASE=1")
	writeDotEnvFile(t, dir, ".env.prod", "PROD=1")

	paths := getDotEnvFilePaths(dir)
	if len(paths) != 2 {
		t.Fatalf("expected 2 paths, got %d: %v", len(paths), paths)
	}

	if filepath.Base(paths[1]) != ".env.prod" {
		t.Errorf("expected .env.prod, got %s", filepath.Base(paths[1]))
	}
}

func writeDotEnvFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
