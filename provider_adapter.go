package confy

import "github.com/JsonLee12138/confy/core"

// bindingAdapter selects source-specific rules for the shared binding pipeline.
// Load/Watch are always dispatched through the core Provider contracts.
type bindingAdapter struct {
	singleTarget    bool
	strictDecoding  bool
	validateUpdates bool
	deduplicate     bool
}

func adapterFor(provider core.Provider) bindingAdapter {
	switch provider.Type() {
	case core.ProviderLocal:
		// Preserve local explicit validation and historical multi-target behavior.
		return bindingAdapter{}
	case core.ProviderNacos:
		return bindingAdapter{singleTarget: true, strictDecoding: true, validateUpdates: true, deduplicate: true}
	default:
		// Custom implementations use conservative candidate validation without
		// being identified as Nacos.
		return bindingAdapter{singleTarget: true, strictDecoding: true, validateUpdates: true, deduplicate: true}
	}
}
