package nacos

import (
	"context"
	"reflect"
	"sync"
	"time"

	"github.com/JsonLee12138/confy/provider"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
)

type watcher struct {
	ctx       context.Context
	provider  *Provider
	changes   chan struct{}
	done      chan struct{}
	once      sync.Once
	retry     bool // accessed only by the delivery goroutine
	finished  chan struct{}
	cancelErr error
}

// Watch starts one serial delivery loop; repeated calls retain the active
// subscription and its original context and callbacks.
func (p *Provider) Watch(ctx context.Context, onChange func(provider.Change), onError func(error)) (provider.Watcher, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := p.initializeLocked(); err != nil {
		return nil, err
	}
	if p.watcher != nil {
		select {
		case <-p.watcher.done:
		default:
			return p.watcher, nil
		}
	}
	w := &watcher{ctx: ctx, provider: p, changes: make(chan struct{}, 1), done: make(chan struct{}), finished: make(chan struct{})}
	param := vo.ConfigParam{DataId: p.opts.dataID, Group: p.opts.group, OnChange: func(_, _, _, _ string) {
		// SDK dispatches callbacks in goroutines: content can arrive out of order.
		// Coalesce wakeups and re-read the current remote value in the delivery loop instead.
		select {
		case <-w.done:
			return
		default:
		}
		select {
		case w.changes <- struct{}{}:
		default:
		}
	}}
	if err := p.client.ListenConfig(param); err != nil {
		_ = p.client.CancelListenConfig(param)
		close(w.done)
		return nil, p.safe(ErrSubscribe)
	}
	// Recheck after registration to close the Load -> Bind race. A registration
	// or read failure leaves no apparently successful watcher and can be retried.
	if _, err := p.readLocked(ctx); err != nil {
		_ = p.client.CancelListenConfig(param)
		close(w.done)
		return nil, err
	}
	p.watcher = w
	select {
	case w.changes <- struct{}{}:
	default:
	}
	go w.run(onChange, onError)
	return w, nil
}

// run refreshes the shared Viper before notifying Bind. Callbacks execute
// outside provider/config locks and can re-enter Config methods or Close.
func (w *watcher) run(onChange func(provider.Change), onError func(error)) {
	defer close(w.finished)
	defer w.Close()
	w.provider.configMu.RLock()
	last := w.provider.v.AllSettings()
	w.provider.configMu.RUnlock()
	for {
		err := w.refresh()
		if w.ctx.Err() != nil {
			return
		}
		select {
		case <-w.done:
			return
		default:
		}
		if err != nil {
			if onError != nil {
				onError(err)
			}
			continue
		}
		w.provider.configMu.RLock()
		settings := w.provider.v.AllSettings()
		w.provider.configMu.RUnlock()
		if reflect.DeepEqual(last, settings) {
			continue
		}
		last = settings
		if onChange != nil {
			onChange(provider.Change{Config: w.provider.v, Event: provider.Event{Name: w.provider.resource(), Op: provider.EventWrite}})
		}
	}
}
func (w *watcher) refresh() error {
	// ListenConfig owns the subscription. A bounded consistency read also
	// covers notifications coalesced/lost while the SDK reconnects or refreshes
	// its internal listener cache; it does not implement a polling protocol.
	delay := 5 * time.Second
	if w.retry {
		delay = time.Second
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-w.ctx.Done():
		return w.ctx.Err()
	case <-w.done:
		return context.Canceled
	case <-w.changes:
	case <-timer.C:
	}
	err := w.provider.refresh(w.ctx)
	w.retry = err != nil
	return err
}
func (w *watcher) stopLocked() {
	w.once.Do(func() {
		close(w.done)
		if w.provider.client != nil && w.provider.client.CancelListenConfig(vo.ConfigParam{DataId: w.provider.opts.dataID, Group: w.provider.opts.group}) != nil {
			w.cancelErr = w.provider.safe(ErrSubscribe)
		}
	})
}
func (w *watcher) Close() error {
	w.provider.mu.Lock()
	defer w.provider.mu.Unlock()
	w.stopLocked()
	return w.cancelErr
}
