package confy

import (
	"bufio"
	"bytes"
	"os"
	"strings"
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
