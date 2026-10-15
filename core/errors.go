package core

import "errors"

// ErrClosed marks mutation or provider access after shutdown.
var ErrClosed = errors.New("confy: config is closed")
