package core_test

import (
	"context"
	"testing"

	"github.com/JsonLee12138/confy"
	"github.com/JsonLee12138/confy/core"
	"github.com/JsonLee12138/confy/provider/nacos"
)

// These assertions prove both adapters implement the same canonical contract.
var _ core.Provider = (*confy.LocalProvider)(nil)
var _ core.WatchProvider = (*confy.LocalProvider)(nil)
var _ core.Provider = (*nacos.Provider)(nil)
var _ core.WatchProvider = (*nacos.Provider)(nil)

func TestCanonicalProviderContracts(t *testing.T) {
	if (&confy.LocalProvider{}).Type() != core.ProviderLocal {
		t.Fatal("incorrect local provider type")
	}
	p := nacos.New(nacos.WithDataID("contract-only"))
	if p.Type() != core.ProviderNacos {
		t.Fatal("incorrect Nacos provider type")
	}
	// Root-package aliases must remain assignable without conversions.
	var change confy.Change = core.Change{Event: core.Event{Name: "resource", Op: core.EventWrite}}
	var event confy.Event = change.Event
	if event.Op != confy.EventWrite {
		t.Fatal("event alias changed")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := p.Load(ctx); err != context.Canceled {
		t.Fatal(err)
	}
	_ = p.Close()
}
