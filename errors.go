package confy

import "errors"

var (
	ErrNilConfig       = errors.New("confy: config instance is nil")
	ErrNilTarget       = errors.New("confy: target instance is nil")
	ErrNoConfigFiles   = errors.New("confy: no valid configuration files found")
	ErrNoSnapshot      = errors.New("confy: no snapshot available to restore")
	ErrNilSnapshot     = errors.New("confy: snapshot is nil")
	ErrEmptyExportPath = errors.New("confy: export path is empty")
	ErrMustBeStruct    = errors.New("confy: config must be a struct")
	ErrCycleDetected   = errors.New("confy: circular base inheritance detected")
)
