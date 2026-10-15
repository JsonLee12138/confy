package confy

import (
	"errors"

	"github.com/JsonLee12138/confy/core"
)

var (
	ErrInvalidOptions   = errors.New("confy: invalid or conflicting provider options")
	ErrInvalidProvider  = errors.New("confy: provider returned nil configuration")
	ErrWatchUnsupported = errors.New("confy: provider does not support watching")
	ErrClosed           = core.ErrClosed
	ErrBoundTarget      = errors.New("confy: provider supports only one binding target")
	ErrCandidate        = errors.New("confy: candidate decode, decryption or validation failed")
	ErrNilConfig        = errors.New("confy: config instance is nil")
	ErrNilTarget        = errors.New("confy: target instance is nil")
	ErrNoConfigFiles    = errors.New("confy: no valid configuration files found")
	ErrNoSnapshot       = errors.New("confy: no snapshot available to restore")
	ErrNilSnapshot      = errors.New("confy: snapshot is nil")
	ErrEmptyExportPath  = errors.New("confy: export path is empty")
	ErrMustBeStruct     = errors.New("confy: config must be a struct")
	ErrCycleDetected    = errors.New("confy: circular base inheritance detected")
)
