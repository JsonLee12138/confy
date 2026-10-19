package local

import (
	"context"
	"sync"

	"github.com/JsonLee12138/confy/provider"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// Watcher stops delivery; Viper does not expose shutdown of its file watcher.
type Watcher struct {
	startOnce sync.Once
	once      sync.Once
	cancel    context.CancelFunc
}

// Viper owns the native watcher. Its private notification instance prevents
// asynchronous native writes racing with reads of the shared configuration.
func (w *Watcher) start(
	ctx context.Context,
	p *Provider,
	onChange func(provider.Change),
	onError func(error),
) {
	w.startOnce.Do(func() {
		ctx, cancel := context.WithCancel(ctx)
		w.cancel = cancel

		v := viper.New()
		v.SetConfigFile(p.v.ConfigFileUsed())
		v.OnConfigChange(func(e fsnotify.Event) {
			if ctx.Err() != nil {
				return
			}
			updated, err := p.Load(ctx)
			if err != nil {
				if onError != nil {
					onError(err)
				}
				return
			}
			if onChange == nil {
				return
			}
			onChange(provider.Change{
				Config: updated,
				Event: provider.Event{
					Name: e.Name,
					Op:   provider.EventOp(e.Op),
				},
			})
		})
		v.WatchConfig()
	})
}

// Close is idempotent and does not wait for an in-flight callback.
func (w *Watcher) Close() error {
	w.once.Do(func() {
		if w.cancel != nil {
			w.cancel()
		}
	})
	return nil
}
