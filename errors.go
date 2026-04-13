package confy

import "fmt"

var (
	ErrNilConfig       = fmt.Errorf("confy: config instance is nil")
	ErrNilTarget       = fmt.Errorf("confy: target instance is nil")
	ErrNoConfigFiles   = fmt.Errorf("confy: no valid configuration files found")
	ErrNoSnapshot      = fmt.Errorf("confy: no snapshot available to restore")
	ErrNilSnapshot     = fmt.Errorf("confy: snapshot is nil")
	ErrEmptyExportPath = fmt.Errorf("confy: export path is empty")
	ErrMustBeStruct    = fmt.Errorf("confy: config must be a struct")
	ErrCycleDetected   = fmt.Errorf("confy: circular base inheritance detected")
)
