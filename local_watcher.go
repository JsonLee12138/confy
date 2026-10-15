package confy

import (
	"context"
	"path/filepath"
	"sync"

	"github.com/fsnotify/fsnotify"
)

type localWatcher struct {
	ctx      context.Context
	provider *LocalProvider
	fs       *fsnotify.Watcher
	name     string
	once     sync.Once
	err      error
}

func (p *LocalProvider) Watch(ctx context.Context) (Watcher, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if p.watchPath == "" {
		if _, err := p.Load(ctx); err != nil {
			return nil, err
		}
	}
	name, err := filepath.Abs(p.watchPath)
	if err != nil {
		return nil, err
	}
	fs, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	if err = fs.Add(filepath.Dir(name)); err != nil {
		_ = fs.Close()
		return nil, err
	}
	return &localWatcher{ctx: ctx, provider: p, fs: fs, name: name}, nil
}
func (w *localWatcher) Next(ctx context.Context) (Change, error) {
	for {
		select {
		case <-w.ctx.Done():
			return Change{}, w.ctx.Err()
		case <-ctx.Done():
			return Change{}, ctx.Err()
		case err, ok := <-w.fs.Errors:
			if !ok {
				return Change{}, context.Canceled
			}
			return Change{}, err
		case event, ok := <-w.fs.Events:
			if !ok {
				return Change{}, context.Canceled
			}
			if filepath.Clean(event.Name) != w.name {
				continue
			}
			v, err := w.provider.Load(ctx)
			if err != nil {
				return Change{}, err
			}
			return Change{Config: v, Event: Event{Name: event.Name, Op: EventOp(event.Op)}}, nil
		}
	}
}
func (w *localWatcher) Close() error { w.once.Do(func() { w.err = w.fs.Close() }); return w.err }
