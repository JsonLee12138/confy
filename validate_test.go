package confy

import (
	"fmt"
	"testing"
)

type ValidConfig struct {
	Name string `mapstructure:"name" required:"true"`
	Port int    `mapstructure:"port" required:"true"`
}

func (v ValidConfig) Validate() error {
	if v.Port < 1 || v.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	return nil
}

type InvalidPortConfig struct {
	Name string `mapstructure:"name" required:"true"`
	Port int    `mapstructure:"port" required:"true"`
}

func (v InvalidPortConfig) Validate() error {
	return fmt.Errorf("invalid port")
}

func TestValidate_WithValidator(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "config.yaml", "name: test\nport: 8080")

	cfg, err := New(WithPath(dir))
	if err != nil {
		t.Fatal(err)
	}

	var appCfg ValidConfig
	cfg.Bind(&appCfg)

	if err := cfg.Validate(); err != nil {
		t.Errorf("expected valid, got error: %v", err)
	}
}

func TestValidate_FailsValidator(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "config.yaml", "name: test\nport: 99999")

	cfg, err := New(WithPath(dir))
	if err != nil {
		t.Fatal(err)
	}

	var appCfg InvalidPortConfig
	cfg.Bind(&appCfg)

	err = cfg.Validate()
	if err == nil {
		t.Error("expected validation error")
	}
}

func TestValidateType_Required(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "config.yaml", "name: test")

	cfg, err := New(WithPath(dir))
	if err != nil {
		t.Fatal(err)
	}

	var appCfg ValidConfig
	cfg.Bind(&appCfg)

	err = cfg.ValidateType(&appCfg)
	if err == nil {
		t.Error("expected error for missing required port")
	}
}

func TestValidateType_Valid(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "config.yaml", "name: test\nport: 8080")

	cfg, err := New(WithPath(dir))
	if err != nil {
		t.Fatal(err)
	}

	var appCfg ValidConfig
	cfg.Bind(&appCfg)

	if err := cfg.ValidateType(&appCfg); err != nil {
		t.Errorf("expected valid, got error: %v", err)
	}
}
