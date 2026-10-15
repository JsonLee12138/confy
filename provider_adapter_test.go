package confy

import (
	"errors"
	"github.com/JsonLee12138/confy/core"
	"testing"
)

type typedProvider struct {
	staticProvider
	kind core.ProviderType
}

func (p *typedProvider) Type() core.ProviderType { return p.kind }
func TestProviderAdapterDispatch(t *testing.T) {
	for _, kind := range []core.ProviderType{core.ProviderNacos, core.ProviderCustom} {
		t.Run(string(kind), func(t *testing.T) {
			p := &typedProvider{kind: kind, staticProvider: staticProvider{value: valueConfig(map[string]any{"name": "source"})}}
			cfg, err := New(WithProvider(p))
			if err != nil {
				t.Fatal(err)
			}
			defer cfg.Close()
			var first, second reloadTarget
			if err = cfg.Bind(&first); err != nil || first.Name != "source" {
				t.Fatal("initial bind failed", err)
			}
			if err = cfg.Bind(&second); !errors.Is(err, ErrBoundTarget) {
				t.Fatal("different target was accepted", err)
			}
			encrypted := &typedProvider{kind: kind, staticProvider: staticProvider{value: valueConfig(map[string]any{"secret": "enc:AES_GCM:unconfigured"})}}
			other, err := New(WithProvider(encrypted))
			if err != nil {
				t.Fatal(err)
			}
			defer other.Close()
			if err = other.Bind(&reloadTarget{}); !errors.Is(err, ErrCandidate) {
				t.Fatal("unconfigured encrypted field was accepted", err)
			}
		})
	}
}

func TestDeclaredLocalProviderUsesLocalBindingRules(t *testing.T) {
	p := &typedProvider{kind: core.ProviderLocal, staticProvider: staticProvider{value: valueConfig(map[string]any{"secret": "enc:AES_GCM:unconfigured"})}}
	cfg, err := New(WithProvider(p))
	if err != nil {
		t.Fatal(err)
	}
	defer cfg.Close()
	var first, second reloadTarget
	if err = cfg.Bind(&first); err != nil {
		t.Fatal(err)
	}
	if err = cfg.Bind(&second); err != nil {
		t.Fatal(err)
	}
	if first.Secret != "enc:AES_GCM:unconfigured" {
		t.Fatal("local compatibility changed")
	}
}
