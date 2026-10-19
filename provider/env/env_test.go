package env_test

import (
	"context"
	"errors"
	"testing"

	"github.com/JsonLee12138/confy"
	"github.com/JsonLee12138/confy/core"
	"github.com/JsonLee12138/confy/provider"
	"github.com/JsonLee12138/confy/provider/env"
)

type envSettings struct {
	Server struct {
		Address string `mapstructure:"address"`
		Port    int    `mapstructure:"port"`
	} `mapstructure:"server"`
	Password string `mapstructure:"password"`
	Ignored  string `mapstructure:"-"`
	Missing  string `mapstructure:"missing"`
}

// TestBindEnvOnlyNestedStruct checks nested tagged keys and excludes ignored or missing ENV fields.
func TestBindEnvOnlyNestedStruct(t *testing.T) {
	t.Setenv("CONFYENV_SERVER_ADDRESS", "localhost")
	t.Setenv("CONFYENV_SERVER_PORT", "8848")
	t.Setenv("CONFYENV_PASSWORD", "test-env-secret")
	t.Setenv("CONFYENV_IGNORED", "ignored")
	var got envSettings
	if err := env.BindEnv(&got, "CONFYENV"); err != nil {
		t.Fatal(err)
	}
	if got.Server.Address != "localhost" || got.Server.Port != 8848 || got.Password != "test-env-secret" || got.Ignored != "" || got.Missing != "" {
		t.Fatal("ENV struct binding mismatch")
	}
}

// TestEnvProviderWorksWithoutFiles checks Config binding with no local source
// and verifies repeated Load calls return independent Viper instances.
func TestEnvProviderWorksWithoutFiles(t *testing.T) {
	t.Setenv("CONFYENV_SERVER_PORT", "9848")
	p := env.New(envSettings{}, "CONFYENV")
	var _ provider.Provider = p
	if p.Type() != provider.ProviderEnv {
		t.Fatal("incorrect source type")
	}
	cfg, err := confy.New(confy.WithProvider(p))
	if err != nil {
		t.Fatal(err)
	}
	defer cfg.Close()
	var target envSettings
	if err = cfg.Bind(&target); err != nil {
		t.Fatal(err)
	}
	if target.Server.Port != 9848 {
		t.Fatal("ENV provider did not bind known struct keys")
	}
	first, err := p.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	second, err := p.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	first.Set("server.port", 1)
	if second.GetInt("server.port") != 9848 {
		t.Fatal("Load shared mutable configuration")
	}
}

// TestBindEnvFailurePreservesTargetAndHidesValues requires failed decoding to preserve
// the old target and return a sentinel without including ENV secrets.
func TestBindEnvFailurePreservesTargetAndHidesValues(t *testing.T) {
	t.Setenv("CONFYENV_SERVER_PORT", "invalid-test-secret")
	original := envSettings{Password: "original"}
	original.Server.Port = 123
	got := original
	err := env.BindEnv(&got, "CONFYENV")
	if !errors.Is(err, core.ErrEnvDecode) {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != original {
		t.Fatal("failed ENV binding mutated target")
	}
	if err.Error() != core.ErrEnvDecode.Error() {
		t.Fatal("decode error exposed environment data")
	}
}

// TestEnvProviderInvalidSchemaTargetAndCancellation checks invalid inputs and canceled loading.
func TestEnvProviderInvalidSchemaTargetAndCancellation(t *testing.T) {
	for _, schema := range []any{nil, 12, map[string]string{}} {
		if _, err := env.New(schema, "").Load(context.Background()); !errors.Is(err, core.ErrEnvSchema) {
			t.Fatal(err)
		}
	}
	for _, target := range []any{nil, envSettings{}, (*envSettings)(nil), new(int)} {
		if err := env.BindEnv(target, ""); !errors.Is(err, core.ErrEnvTarget) {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := env.New(envSettings{}, "").Load(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

// TestBindEnvSquashAndNestedPointers checks flattened value structs and allocated nested pointers.
func TestBindEnvSquashAndNestedPointers(t *testing.T) {
	type Auth struct {
		Username string `mapstructure:"username"`
	}
	type Schema struct {
		Auth   `mapstructure:",squash"`
		Nested *struct {
			Enabled bool `mapstructure:"enabled"`
		} `mapstructure:"nested"`
	}
	t.Setenv("CONFYENV_USERNAME", "reader")
	t.Setenv("CONFYENV_NESTED_ENABLED", "true")
	var got Schema
	if err := env.BindEnv(&got, "CONFYENV"); err != nil {
		t.Fatal(err)
	}
	if got.Username != "reader" || got.Nested == nil || !got.Nested.Enabled {
		t.Fatal("pointer/squash binding failed")
	}
}

// TestEnvProviderRejectsRecursiveSchema prevents unbounded traversal of self-referential schemas.
func TestEnvProviderRejectsRecursiveSchema(t *testing.T) {
	type Recursive struct{ Next *Recursive }
	if _, err := env.New(Recursive{}, "").Load(context.Background()); !errors.Is(err, core.ErrEnvSchema) {
		t.Fatal(err)
	}
}

// TestBindEnvUnprefixedValuesAndEmptyZeroValues checks direct keys, slice conversion,
// and replacement of preexisting values when ENV is empty.
func TestBindEnvUnprefixedValuesAndEmptyZeroValues(t *testing.T) {
	type Settings struct {
		Enabled bool     `mapstructure:"confy_env_enabled"`
		Names   []string `mapstructure:"confy_env_names"`
		Empty   string   `mapstructure:"confy_env_empty"`
	}
	t.Setenv("CONFY_ENV_ENABLED", "true")
	t.Setenv("CONFY_ENV_NAMES", "one,two")
	t.Setenv("CONFY_ENV_EMPTY", "")
	target := Settings{Empty: "old-value"}
	if err := env.BindEnv(&target, ""); err != nil {
		t.Fatal(err)
	}
	if !target.Enabled || len(target.Names) != 2 || target.Names[1] != "two" || target.Empty != "" {
		t.Fatal("pure ENV values were not decoded")
	}
}

// TestEnvProviderRejectsUnsupportedSquashAndRemain rejects pointer squash and dynamic remainder maps.
func TestEnvProviderRejectsUnsupportedSquashAndRemain(t *testing.T) {
	type Auth struct{ Username string }
	for _, schema := range []any{
		struct {
			*Auth `mapstructure:",squash"`
		}{},
		struct {
			Rest map[string]any `mapstructure:",remain"`
		}{},
	} {
		if _, err := env.New(schema, "CONFYENV").Load(context.Background()); !errors.Is(err, core.ErrEnvSchema) {
			t.Fatal(err)
		}
	}
}
