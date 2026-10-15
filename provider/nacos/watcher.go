package nacos

import (
	"context"
	"sync"
	"time"

	"github.com/JsonLee12138/confy"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
)

type watcher struct {
	ctx       context.Context
	provider  *Provider
	changes   chan struct{}
	done      chan struct{}
	once      sync.Once
	retry     bool // accessed only by the single Next consumer
	cancelErr error
}

func (p *Provider) Watch(ctx context.Context) (confy.Watcher, error) {
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
			return nil, p.safe(ErrSubscribe)
		}
	}
	w := &watcher{ctx: ctx, provider: p, changes: make(chan struct{}, 1), done: make(chan struct{})}
	param := vo.ConfigParam{DataId: p.opts.dataID, Group: p.opts.group, OnChange: func(_, _, _, _ string) {
		// SDK dispatches callbacks in goroutines: content can arrive out of order.
		// Coalesce wakeups and re-read the current remote value in Next instead.
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
	return w, nil
}
func (w *watcher) Next(ctx context.Context) (confy.Change, error) {
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
		return confy.Change{}, w.ctx.Err()
	case <-ctx.Done():
		return confy.Change{}, ctx.Err()
	case <-w.done:
		return confy.Change{}, context.Canceled
	case <-w.changes:
	case <-timer.C:
	}
	p := w.provider
	p.mu.Lock()
	defer p.mu.Unlock()
	select {
	case <-w.done:
		return confy.Change{}, context.Canceled
	default:
	}
	content, err := p.readLocked(ctx)
	w.retry = err != nil
	if err != nil {
		return confy.Change{}, err
	}
	v, err := parse(content, p.opts.format)
	if err != nil {
		return confy.Change{}, p.safe(ErrDocument)
	}
	return confy.Change{Config: v, Event: confy.Event{Name: p.resource(), Op: confy.EventWrite}}, nil
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
