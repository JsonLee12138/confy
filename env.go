package confy

import (
	"os"
	"strings"

	"github.com/spf13/viper"
)

// applyEnvOverrides checks all config keys and overrides with environment
// variables if they exist. Config key "server.port" maps to "SERVER_PORT",
// or "MYAPP_SERVER_PORT" with prefix "MYAPP".
func applyEnvOverrides(v *viper.Viper, envPrefix string) {
	replacer := strings.NewReplacer(".", "_")

	for _, key := range v.AllKeys() {
		envKey := strings.ToUpper(replacer.Replace(key))
		if envPrefix != "" {
			envKey = envPrefix + "_" + envKey
		}

		if envValue := os.Getenv(envKey); envValue != "" {
			v.Set(key, envValue)
		}
	}
}
