package e2e_test

import (
	"errors"
	"testing"

	"github.com/JsonLee12138/confy"
	"github.com/JsonLee12138/confy/provider/env"
)

// TestE2EEnvProvider exercises direct and Provider-based ENV binding through public APIs,
// including precedence, explicit zeros, invalid-value recovery, and unsupported watching.
func TestE2EEnvProvider(t *testing.T) {
	t.Setenv("CONFY_E2E_NAME", "initial")
	t.Setenv("CONFY_E2E_SERVER_PORT", "9090")
	t.Setenv("CONFY_E2E_TIMEOUT", "0")
	t.Setenv("CONFY_E2E_ENABLED", "false")
	var direct e2eSettings
	if err := confy.BindEnv(&direct, "CONFY_E2E"); err != nil {
		t.Fatal(err)
	}
	if direct.Name != "initial" || direct.Server.Port != 9090 || direct.Timeout != 0 || direct.Enabled {
		t.Fatalf("direct binding = %+v", direct)
	}
	cfg, err := confy.New(confy.WithProvider(env.New(e2eSettings{}, "CONFY_E2E")))
	if err != nil {
		t.Fatal(err)
	}
	defer cfg.Close()
	var target e2eSettings
	if err := cfg.Bind(&target); err != nil {
		t.Fatal(err)
	}
	if target != direct {
		t.Fatalf("provider=%+v, direct=%+v", target, direct)
	}
	t.Setenv("CONFY_E2E_NAME", "changed")
	t.Setenv("CONFY_E2E_TIMEOUT", "")
	cfg.Set("server.port", 10001)
	if err := cfg.Bind(&target); err != nil {
		t.Fatal(err)
	}
	if target.Name != "changed" || target.Timeout != 30 || target.Server.Port != 10001 {
		t.Fatalf("rebind = %+v", target)
	}
	before := target
	t.Setenv("CONFY_E2E_TIMEOUT", "invalid")
	if err := cfg.Bind(&target); err == nil {
		t.Fatal("invalid ENV decoded")
	}
	if target != before {
		t.Fatal("invalid ENV changed target")
	}
	if err := confy.BindEnv(&direct, "CONFY_E2E"); !errors.Is(err, confy.ErrEnvDecode) {
		t.Fatalf("direct invalid ENV = %v", err)
	}
	t.Setenv("CONFY_E2E_TIMEOUT", "45")
	if err := cfg.Bind(&target); err != nil || target.Timeout != 45 {
		t.Fatalf("recovery: %+v, %v", target, err)
	}
	watch, err := confy.New(confy.WithProvider(env.New(e2eSettings{}, "CONFY_E2E")), confy.WithWatch(true))
	if err != nil {
		t.Fatal(err)
	}
	defer watch.Close()
	if err := watch.Bind(&e2eSettings{}); !errors.Is(err, confy.ErrWatchUnsupported) {
		t.Fatalf("ENV watch = %v", err)
	}
	t.Log("real process ENV: direct/provider Bind, nested keys, zero/default, reread, Set precedence, invalid value, recovery, unsupported watch passed")
}
