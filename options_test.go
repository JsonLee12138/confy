package confy

import (
	"testing"
)

func TestDefaultOptions(t *testing.T) {
	opts := defaultOptions()

	if opts.basePath != "config" {
		t.Errorf("expected basePath 'config', got '%s'", opts.basePath)
	}
	if opts.fileName != "config" {
		t.Errorf("expected fileName 'config', got '%s'", opts.fileName)
	}
	if opts.fileType != YAML {
		t.Errorf("expected fileType 'yaml', got '%s'", opts.fileType)
	}
	if opts.watchable {
		t.Error("expected watchable to be false by default")
	}
	if opts.loadAll {
		t.Error("expected loadAll to be false by default")
	}
}

func TestWithPath(t *testing.T) {
	opts := defaultOptions()
	WithPath("myconfig")(opts)

	if opts.basePath != "myconfig" {
		t.Errorf("expected basePath 'myconfig', got '%s'", opts.basePath)
	}
}

func TestWithFile(t *testing.T) {
	opts := defaultOptions()
	WithFile("app")(opts)

	if opts.fileName != "app" {
		t.Errorf("expected fileName 'app', got '%s'", opts.fileName)
	}
}

func TestWithFileType(t *testing.T) {
	opts := defaultOptions()
	WithFileType(JSON)(opts)

	if opts.fileType != JSON {
		t.Errorf("expected fileType 'json', got '%s'", opts.fileType)
	}
}

func TestWithEnvPrefix(t *testing.T) {
	opts := defaultOptions()
	WithEnvPrefix("MYAPP")(opts)

	if opts.envPrefix != "MYAPP" {
		t.Errorf("expected envPrefix 'MYAPP', got '%s'", opts.envPrefix)
	}
}

func TestWithWatch(t *testing.T) {
	opts := defaultOptions()
	WithWatch(true)(opts)

	if !opts.watchable {
		t.Error("expected watchable to be true")
	}
}

func TestWithLoadAll(t *testing.T) {
	opts := defaultOptions()
	WithLoadAll(true)(opts)

	if !opts.loadAll {
		t.Error("expected loadAll to be true")
	}
}

func TestWithDotEnv(t *testing.T) {
	opts := defaultOptions()
	WithDotEnv(".env")(opts)
	WithDotEnv(".env.local")(opts)

	if len(opts.dotEnvFiles) != 2 {
		t.Errorf("expected 2 dotenv files, got %d", len(opts.dotEnvFiles))
	}
}

func TestWithEncryption(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")
	opts := defaultOptions()
	WithEncryption("aes-gcm", key)(opts)

	if opts.encAlgo != "aes-gcm" {
		t.Errorf("expected encAlgo 'aes-gcm', got '%s'", opts.encAlgo)
	}
	if len(opts.encKey) != 32 {
		t.Errorf("expected 32-byte key, got %d bytes", len(opts.encKey))
	}
}
