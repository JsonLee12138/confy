package confy

import "os"

// Option is a function that configures a configOptions instance.
type Option func(*configOptions)

type configOptions struct {
	basePath    string
	fileName    string
	fileType    FileType
	envPrefix   string
	watchable   bool
	onChange    func(Event)
	loadAll     bool
	dotEnvFiles []string
	dotEnvDir   string
	encAlgo     string
	encKey      []byte
}

func defaultOptions() *configOptions {
	basePath := os.Getenv("CONFIG_PATH")
	if basePath == "" {
		basePath = "config"
	}
	return &configOptions{
		basePath:  basePath,
		fileName:  "config",
		fileType:  YAML,
		watchable: false,
		loadAll:   false,
	}
}

// WithPath sets the directory to search for config files.
func WithPath(path string) Option {
	return func(o *configOptions) {
		o.basePath = path
	}
}

// WithFile sets the base config file name (without extension).
func WithFile(name string) Option {
	return func(o *configOptions) {
		o.fileName = name
	}
}

// WithFileType sets the default config file format.
func WithFileType(ft FileType) Option {
	return func(o *configOptions) {
		o.fileType = ft
	}
}

// WithEnvPrefix sets the environment variable prefix.
func WithEnvPrefix(prefix string) Option {
	return func(o *configOptions) {
		o.envPrefix = prefix
	}
}

// WithWatch enables or disables file watching for hot-reload.
func WithWatch(enable bool) Option {
	return func(o *configOptions) {
		o.watchable = enable
	}
}

// WithOnChange sets the callback for config file changes.
func WithOnChange(fn func(Event)) Option {
	return func(o *configOptions) {
		o.onChange = fn
	}
}

// WithLoadAll enables loading all config files in the directory.
func WithLoadAll(enable bool) Option {
	return func(o *configOptions) {
		o.loadAll = enable
	}
}

// WithDotEnv adds a .env file to load.
func WithDotEnv(path string) Option {
	return func(o *configOptions) {
		o.dotEnvFiles = append(o.dotEnvFiles, path)
	}
}

// WithDotEnvAuto enables automatic .env file discovery based on environment mode.
// It discovers and loads .env files in priority order: .env → .env.local → .env.{env} → .env.{env}.local
// The dir parameter specifies the directory to search in (defaults to "." if empty).
func WithDotEnvAuto(dir ...string) Option {
	return func(o *configOptions) {
		if len(dir) > 0 && dir[0] != "" {
			o.dotEnvDir = dir[0]
		} else {
			o.dotEnvDir = "."
		}
	}
}

// WithEncryption enables config value encryption/decryption.
func WithEncryption(algo string, key []byte) Option {
	return func(o *configOptions) {
		o.encAlgo = algo
		o.encKey = key
	}
}
