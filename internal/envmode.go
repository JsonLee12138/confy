package internal

import (
	"os"
	"strings"
	"sync"
)

const EnvModeKey = "GO_ENV_MODE"

type EnvMode string

const (
	DevMode  EnvMode = "development"
	ProMode  EnvMode = "production"
	TestMode EnvMode = "test"
)

var (
	currentEnv EnvMode
	modeOnce   sync.Once
)

func ParseEnv(env string) EnvMode {
	normalized := strings.ToLower(strings.TrimSpace(env))
	switch normalized {
	case "development", "dev", "":
		return DevMode
	case "production", "prod", "pro":
		return ProMode
	case "test", "testing":
		return TestMode
	default:
		return DevMode
	}
}

func Mode() EnvMode {
	if currentEnv == "" {
		modeOnce.Do(func() {
			currentEnv = ParseEnv(os.Getenv(EnvModeKey))
			if currentEnv == "" {
				currentEnv = DevMode
			}
		})
	}
	return currentEnv
}

func SetMode(mode EnvMode) {
	os.Setenv(EnvModeKey, string(mode))
}
