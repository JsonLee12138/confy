package confy

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/JsonLee12138/confy/internal"
)

// getConfigFilePaths discovers config files based on environment mode.
// Priority order: config.yaml → config.local.yaml → config.{env}.yaml → config.{env}.local.yaml
func getConfigFilePaths(opts *configOptions) []string {
	env := internal.Mode()
	fileNames := []string{
		opts.fileName,
		fmt.Sprintf("%s.local", opts.fileName),
		fmt.Sprintf("%s.%s", opts.fileName, env),
		fmt.Sprintf("%s.%s.local", opts.fileName, env),
	}

	switch env {
	case internal.DevMode:
		fileNames = append(fileNames,
			fmt.Sprintf("%s.dev", opts.fileName),
			fmt.Sprintf("%s.dev.local", opts.fileName),
			fmt.Sprintf("%s.development", opts.fileName),
			fmt.Sprintf("%s.development.local", opts.fileName),
		)
	case internal.ProMode:
		fileNames = append(fileNames,
			fmt.Sprintf("%s.pro", opts.fileName),
			fmt.Sprintf("%s.pro.local", opts.fileName),
			fmt.Sprintf("%s.prod", opts.fileName),
			fmt.Sprintf("%s.prod.local", opts.fileName),
			fmt.Sprintf("%s.production", opts.fileName),
			fmt.Sprintf("%s.production.local", opts.fileName),
		)
	case internal.TestMode:
		fileNames = append(fileNames,
			fmt.Sprintf("%s.test", opts.fileName),
			fmt.Sprintf("%s.test.local", opts.fileName),
		)
	}

	// Deduplicate file names while preserving order.
	fileNames = deduplicate(fileNames)

	var configFiles []string
	for _, fileName := range fileNames {
		for _, ext := range supportedExtensions(opts.fileType) {
			file := filepath.Join(opts.basePath, fileName+"."+ext)
			if isDir, exists, _ := internal.Exists(file); exists && !isDir {
				configFiles = append(configFiles, file)
				break
			}
		}
	}

	return configFiles
}

// getAllConfigFilePaths discovers all config files in the directory (LoadAll mode).
func getAllConfigFilePaths(opts *configOptions) []string {
	baseNames := getConfigBaseNames(opts.basePath, opts.fileType)
	if len(baseNames) == 0 {
		return nil
	}

	sort.Strings(baseNames)
	baseNames = moveConfigFirst(baseNames)

	seen := make(map[string]struct{})
	var configFiles []string
	for _, baseName := range baseNames {
		tempOpts := *opts
		tempOpts.fileName = baseName
		tempOpts.loadAll = false
		for _, path := range getConfigFilePaths(&tempOpts) {
			if _, exists := seen[path]; exists {
				continue
			}
			seen[path] = struct{}{}
			configFiles = append(configFiles, path)
		}
	}

	return configFiles
}

func getConfigBaseNames(basePath string, fileType FileType) []string {
	entries, err := os.ReadDir(basePath)
	if err != nil {
		return nil
	}

	seen := make(map[string]struct{})
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		ext := filepath.Ext(name)
		if !isSupportedExt(ext) {
			continue
		}
		base := strings.TrimSuffix(name, ext)
		base = stripConfigSuffix(base)
		if base == "" {
			continue
		}
		seen[base] = struct{}{}
	}

	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	return names
}

func stripConfigSuffix(name string) string {
	name = strings.TrimSuffix(name, ".local")
	return trimEnvSuffix(name)
}

func trimEnvSuffix(name string) string {
	suffixes := []string{".dev", ".development", ".pro", ".prod", ".production", ".test"}
	for _, s := range suffixes {
		if strings.HasSuffix(name, s) {
			return strings.TrimSuffix(name, s)
		}
	}
	return name
}

func moveConfigFirst(names []string) []string {
	idx := -1
	for i, name := range names {
		if name == "config" {
			idx = i
			break
		}
	}
	if idx <= 0 {
		return names
	}
	out := make([]string, 0, len(names))
	out = append(out, "config")
	out = append(out, names[:idx]...)
	out = append(out, names[idx+1:]...)
	return out
}

func supportedExtensions(defaultType FileType) []string {
	switch defaultType {
	case JSON:
		return []string{"json", "yaml", "toml"}
	case TOML:
		return []string{"toml", "yaml", "json"}
	default:
		return []string{"yaml", "json", "toml"}
	}
}

func isSupportedExt(ext string) bool {
	ext = strings.TrimPrefix(ext, ".")
	return ext == "yaml" || ext == "yml" || ext == "json" || ext == "toml"
}

func deduplicate(ss []string) []string {
	seen := make(map[string]struct{}, len(ss))
	out := ss[:0]
	for _, s := range ss {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}
