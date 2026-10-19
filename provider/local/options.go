package local

import (
	"os"

	"github.com/JsonLee12138/confy/core"
)

// Option is a function that configures a Options instance.
type Option func(*Options)

type Options struct {
	BasePath      string
	FileName      string
	FileType      core.FileType
	LoadAll 	  bool
}

func DefaultOptions() *Options {
	basePath := os.Getenv("CONFIG_PATH")
	if basePath == "" {
		basePath = "config"
	}
	return &Options{
		BasePath:  basePath,
		FileName:  "config",
		FileType:  core.YAML,
		LoadAll: true,
	}
}

// WithPath sets the directory to search for config files.
func WithPath(path string) Option {
	return func(o *Options) {
		o.BasePath = path
	}
}

// WithFile sets the base config file name (without extension).
func WithFile(name string) Option {
	return func(o *Options) {
		o.FileName = name
	}
}

// WithFileType sets the default config file format.
func WithFileType(ft core.FileType) Option {
	return func(o *Options) {
		o.FileType = ft
	}
}

// WithLoadAll enables loading all config files in the directory.
func WithLoadAll(enable bool) Option {
	return func(o *Options) {
		o.LoadAll = enable
	}
}
