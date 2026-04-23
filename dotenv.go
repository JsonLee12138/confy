package confy

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/JsonLee12138/confy/internal"
)

// loadDotEnv loads a .env file and sets environment variables.
// If the file doesn't exist, it returns nil (no error).
func loadDotEnv(path string) error {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

	envs, err := parseDotEnv(data)
	if err != nil {
		return err
	}

	for k, v := range envs {
		if existing := os.Getenv(k); existing == "" {
			os.Setenv(k, v)
		}
	}

	return nil
}

// parseDotEnv parses .env file content into key-value pairs.
// Supports: comments (#), quoted values (" and '), empty values.
func parseDotEnv(data []byte) (map[string]string, error) {
	envs := make(map[string]string)
	scanner := bufio.NewScanner(bytes.NewReader(data))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		idx := strings.IndexByte(line, '=')
		if idx < 0 {
			continue
		}

		key := strings.TrimSpace(line[:idx])
		value := strings.TrimSpace(line[idx+1:])

		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') ||
				(value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}

		envs[key] = value
	}

	return envs, scanner.Err()
}

// getDotEnvFilePaths discovers .env files based on environment mode.
// Priority order: .env → .env.local → .env.{env} → .env.{env}.local
func getDotEnvFilePaths(dir string) []string {
	env := internal.Mode()
	fileNames := []string{
		".env",
		".env.local",
		fmt.Sprintf(".env.%s", env),
		fmt.Sprintf(".env.%s.local", env),
	}

	switch env {
	case internal.DevMode:
		fileNames = append(fileNames,
			".env.dev",
			".env.dev.local",
			".env.development",
			".env.development.local",
		)
	case internal.ProMode:
		fileNames = append(fileNames,
			".env.pro",
			".env.pro.local",
			".env.prod",
			".env.prod.local",
			".env.production",
			".env.production.local",
		)
	case internal.TestMode:
		fileNames = append(fileNames,
			".env.test",
			".env.test.local",
		)
	}

	fileNames = deduplicate(fileNames)

	var files []string
	for _, name := range fileNames {
		path := filepath.Join(dir, name)
		if isDir, exists, _ := internal.Exists(path); exists && !isDir {
			files = append(files, path)
		}
	}
	return files
}
