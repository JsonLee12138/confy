package core

import "errors"

var (
	ErrEnvSchema        = errors.New("confy: invalid or unsupported ENV struct schema")
	ErrEnvTarget        = errors.New("confy: ENV target must be a non-nil pointer to a struct")
	ErrEnvDecode        = errors.New("confy: failed to decode environment configuration")
	ErrInvalidOptions   = errors.New("confy: invalid or conflicting provider options")
	ErrInvalidProvider  = errors.New("confy: provider returned nil configuration")
	ErrWatchUnsupported = errors.New("confy: provider does not support watching")
	ErrClosed           = errors.New("confy: config is closed")
	ErrBoundTarget      = errors.New("confy: provider supports only one binding target")
	ErrCandidate        = errors.New("confy: candidate decode, decryption or validation failed")
	ErrNilConfig        = errors.New("confy: config instance is nil")
	ErrNotLoaded        = errors.New("confy: configuration has not been loaded")
	ErrNilTarget        = errors.New("confy: target instance is nil")
	ErrNoConfigFiles    = errors.New("confy: no valid configuration files found")
	ErrNoSnapshot       = errors.New("confy: no snapshot available to restore")
	ErrNilSnapshot      = errors.New("confy: snapshot is nil")
	ErrEmptyExportPath  = errors.New("confy: export path is empty")
	ErrMustBeStruct     = errors.New("confy: config must be a struct")
	ErrCycleDetected    = errors.New("confy: circular base inheritance detected")
)
