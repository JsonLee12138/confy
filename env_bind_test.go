package confy_test

import (
	"errors"
	"testing"

	"github.com/JsonLee12138/confy"
	"github.com/JsonLee12138/confy/core"
	"github.com/JsonLee12138/confy/provider"
	"github.com/JsonLee12138/confy/provider/env"
)

// TestBindEnvPublicEntry checks the root ENV API, Provider type aliases, safe errors,
// and target preservation when decoding fails.
func TestBindEnvPublicEntry(t *testing.T) {
	t.Setenv("CONFY_PUBLIC_PORT", "8848")
	var target struct {
		Port int `mapstructure:"port"`
	}
	if err := confy.BindEnv(&target, "CONFY_PUBLIC"); err != nil {
		t.Fatal(err)
	}
	if target.Port != 8848 {
		t.Fatalf("port = %d", target.Port)
	}
	if env.New(target, "CONFY_PUBLIC").Type() != confy.ProviderEnv || confy.ProviderEnv != provider.ProviderEnv {
		t.Fatal("ENV provider type differs across packages")
	}
	t.Setenv("CONFY_PUBLIC_PORT", "invalid")
	if err := confy.BindEnv(&target, "CONFY_PUBLIC"); !errors.Is(err, confy.ErrEnvDecode) || !errors.Is(err, core.ErrEnvDecode) {
		t.Fatalf("decode error = %v", err)
	}
	if target.Port != 8848 {
		t.Fatal("failed binding changed target")
	}
	if err := confy.BindEnv(nil, ""); !errors.Is(err, confy.ErrEnvTarget) {
		t.Fatalf("target error = %v", err)
	}
	if err := confy.BindEnv(&struct {
		Rest map[string]any `mapstructure:",remain"`
	}{}, ""); !errors.Is(err, confy.ErrEnvSchema) {
		t.Fatalf("schema error = %v", err)
	}
}

// TestPublicErrorsAliasCore requires root sentinel errors to share identity with core errors.
func TestPublicErrorsAliasCore(t *testing.T) {
	for _, pair := range [][2]error{
		{confy.ErrInvalidOptions, core.ErrInvalidOptions},
		{confy.ErrInvalidProvider, core.ErrInvalidProvider},
		{confy.ErrWatchUnsupported, core.ErrWatchUnsupported},
		{confy.ErrClosed, core.ErrClosed},
		{confy.ErrBoundTarget, core.ErrBoundTarget},
		{confy.ErrCandidate, core.ErrCandidate},
		{confy.ErrNilConfig, core.ErrNilConfig},
		{confy.ErrNotLoaded, core.ErrNotLoaded},
		{confy.ErrNilTarget, core.ErrNilTarget},
		{confy.ErrNoConfigFiles, core.ErrNoConfigFiles},
		{confy.ErrNoSnapshot, core.ErrNoSnapshot},
		{confy.ErrNilSnapshot, core.ErrNilSnapshot},
		{confy.ErrEmptyExportPath, core.ErrEmptyExportPath},
		{confy.ErrMustBeStruct, core.ErrMustBeStruct},
		{confy.ErrCycleDetected, core.ErrCycleDetected},
	} {
		if pair[0] != pair[1] {
			t.Fatalf("%v does not alias core", pair[0])
		}
	}
}
