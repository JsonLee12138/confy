package confy

import "github.com/JsonLee12138/confy/core"

// Public errors alias core errors so errors.Is works across packages.
var (
	ErrEnvSchema        = core.ErrEnvSchema
	ErrEnvTarget        = core.ErrEnvTarget
	ErrEnvDecode        = core.ErrEnvDecode
	ErrInvalidOptions   = core.ErrInvalidOptions
	ErrInvalidProvider  = core.ErrInvalidProvider
	ErrWatchUnsupported = core.ErrWatchUnsupported
	ErrClosed           = core.ErrClosed
	ErrBoundTarget      = core.ErrBoundTarget
	ErrCandidate        = core.ErrCandidate
	ErrNilConfig        = core.ErrNilConfig
	ErrNotLoaded        = core.ErrNotLoaded
	ErrNilTarget        = core.ErrNilTarget
	ErrNoConfigFiles    = core.ErrNoConfigFiles
	ErrNoSnapshot       = core.ErrNoSnapshot
	ErrNilSnapshot      = core.ErrNilSnapshot
	ErrEmptyExportPath  = core.ErrEmptyExportPath
	ErrMustBeStruct     = core.ErrMustBeStruct
	ErrCycleDetected    = core.ErrCycleDetected
)
