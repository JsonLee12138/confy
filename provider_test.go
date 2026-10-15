package confy

import (
	"context"
	"errors"
	"github.com/JsonLee12138/confy/core"
	"github.com/spf13/viper"
	"testing"
)

type staticProvider struct {
	value *viper.Viper
	err   error
}

func (p *staticProvider) Load(context.Context) (*viper.Viper, error) { return p.value, p.err }
func TestProviderOptions(t *testing.T) {
	p := &staticProvider{value: viper.New()}
	p.value.Set("name", "remote")
	for _, opts := range [][]Option{
		{WithProvider(nil)}, {WithProvider(p), WithProvider(p)},
		{WithPath("missing"), WithProvider(p)}, {WithProvider(p), WithLoadAll(false)},
		{WithFileType(YAML), WithProvider(p)}, {WithProvider(p), WithFile("x")},
	} {
		if _, err := New(opts...); !errors.Is(err, ErrInvalidOptions) {
			t.Fatalf("expected options error, got %v", err)
		}
	}
	if _, err := New(WithProvider(&staticProvider{})); !errors.Is(err, ErrInvalidProvider) {
		t.Fatalf("nil result: %v", err)
	}
	cfg, err := New(WithProvider(p))
	if err != nil {
		t.Fatal(err)
	}
	defer cfg.Close()
	var target struct{ Name string }
	if err = cfg.Bind(&target); err != nil || target.Name != "remote" {
		t.Fatalf("bind: %v %+v", err, target)
	}
}

func (*staticProvider) Type() core.ProviderType { return core.ProviderCustom }
