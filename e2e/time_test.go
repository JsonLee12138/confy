package e2e_test

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/JsonLee12138/confy"
	"github.com/JsonLee12138/confy/provider/env"
)

type timeSettings struct {
	Opaque    struct{ hidden string } `mapstructure:"opaque"`
	StartedAt time.Time               `mapstructure:"started_at"`
	Optional  *time.Time              `mapstructure:"optional"`
	Missing   *time.Time              `mapstructure:"missing"`
	Delay     time.Duration           `mapstructure:"delay"`
	Names     []string                `mapstructure:"names"`
	Ports     []int                   `mapstructure:"ports"`
}

// TestE2EEnvTimeBinding checks RFC3339 time values and pointers through both ENV entry points,
// preserving duration/slice decoding and the previous target after invalid timestamps.
func TestE2EEnvTimeBinding(t *testing.T) {
	stamp := "2026-10-19T06:00:00.123456789Z"
	want, err := time.Parse(time.RFC3339Nano, stamp)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("CONFY_TIME_OPAQUE", "not-a-supported-scalar")
	t.Setenv("CONFY_TIME_STARTED_AT", stamp)
	t.Setenv("CONFY_TIME_OPTIONAL", stamp)
	t.Setenv("CONFY_TIME_MISSING", "")
	t.Setenv("CONFY_TIME_DELAY", "2s")
	t.Setenv("CONFY_TIME_NAMES", "one,two")
	t.Setenv("CONFY_TIME_PORTS", "80,443")
	cfg, err := confy.New(confy.WithProvider(env.New(timeSettings{}, "CONFY_TIME")))
	if err != nil {
		t.Fatal(err)
	}
	defer cfg.Close()
	for _, bind := range []struct {
		name string
		call func(*timeSettings) error
	}{
		{"direct", func(target *timeSettings) error { return confy.BindEnv(target, "CONFY_TIME") }},
		{"provider", func(target *timeSettings) error { return cfg.Bind(target) }},
	} {
		t.Run(bind.name, func(t *testing.T) {
			var target timeSettings
			if err := bind.call(&target); err != nil {
				t.Fatal(err)
			}
			if !target.StartedAt.Equal(want) || target.Optional == nil || !target.Optional.Equal(want) || target.Missing != nil || target.Delay != 2*time.Second || !reflect.DeepEqual(target.Names, []string{"one", "two"}) || !reflect.DeepEqual(target.Ports, []int{80, 443}) {
				t.Fatalf("binding = %+v", target)
			}
			before := target
			for _, key := range []string{"CONFY_TIME_STARTED_AT", "CONFY_TIME_OPTIONAL"} {
				t.Setenv(key, "invalid-time")
				err := bind.call(&target)
				if err == nil {
					t.Fatal("invalid time accepted")
				}
				if bind.name == "direct" && !errors.Is(err, confy.ErrEnvDecode) {
					t.Fatalf("direct error = %v", err)
				}
				if !reflect.DeepEqual(before, target) {
					t.Fatal("invalid time changed target")
				}
				t.Setenv(key, stamp)
			}
		})
	}
}
