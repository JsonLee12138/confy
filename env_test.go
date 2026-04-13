package confy

import (
	"os"
	"testing"

	"github.com/spf13/viper"
)

func TestApplyEnvOverrides(t *testing.T) {
	v := viper.New()
	v.Set("server.port", 8080)
	v.Set("database.host", "localhost")

	t.Setenv("SERVER_PORT", "9090")
	t.Setenv("DATABASE_HOST", "remote-host")

	applyEnvOverrides(v, "")

	if v.GetString("server.port") != "9090" {
		t.Errorf("expected server.port '9090', got '%s'", v.GetString("server.port"))
	}
	if v.GetString("database.host") != "remote-host" {
		t.Errorf("expected database.host 'remote-host', got '%s'", v.GetString("database.host"))
	}
}

func TestApplyEnvOverrides_WithPrefix(t *testing.T) {
	v := viper.New()
	v.Set("server.port", 8080)

	t.Setenv("MYAPP_SERVER_PORT", "3000")

	applyEnvOverrides(v, "MYAPP")

	if v.GetString("server.port") != "3000" {
		t.Errorf("expected server.port '3000', got '%s'", v.GetString("server.port"))
	}
}

func TestApplyEnvOverrides_NoOverrideWhenEmpty(t *testing.T) {
	v := viper.New()
	v.Set("server.port", 8080)

	os.Unsetenv("SERVER_PORT")
	applyEnvOverrides(v, "")

	if v.GetString("server.port") != "8080" {
		t.Errorf("expected server.port '8080', got '%s'", v.GetString("server.port"))
	}
}
