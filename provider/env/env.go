// Package env binds environment variables through Viper using a struct schema.
package env

import (
	"context"
	"reflect"
	"strings"

	"github.com/JsonLee12138/confy/core"
	"github.com/JsonLee12138/confy/provider"
	"github.com/spf13/viper"
)

// Provider reads only the ENV keys declared by a struct's mapstructure tags.
// It does not read files, discover unrelated process variables, or watch ENV.
// Missing/empty variables use Viper's default ENV semantics (zero values).
type Provider struct {
	schema reflect.Type
	prefix string
}

// New collects a struct schema and an optional ENV prefix (without
// the trailing underscore). Schema values are ignored; only its type is used.
// Nested structs/pointers and value-struct mapstructure:",squash" are supported.
// Recursive structs, pointer squash and dynamic mapstructure:",remain" schemas
// are rejected during Load.
func New(schema any, prefix string) *Provider {
	return &Provider{schema: reflect.TypeOf(schema), prefix: prefix}
}
func (*Provider) Type() provider.ProviderType { return provider.ProviderEnv }

// Load registers declared keys so Viper.Unmarshal can discover pure ENV values.
// Each call returns an independent Viper, with no defaults or explicit overrides.
func (p *Provider) Load(ctx context.Context) (*viper.Viper, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if p == nil || p.schema == nil {
		return nil, core.ErrEnvSchema
	}
	schema := structType(p.schema)
	if schema.Kind() != reflect.Struct {
		return nil, core.ErrEnvSchema
	}
	v := viper.New()
	v.SetEnvPrefix(p.prefix)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	if err := bindStructEnv(ctx, v, schema, "", make(map[reflect.Type]bool)); err != nil {
		return nil, err
	}
	return v, nil
}

// BindEnv reads a fresh ENV-only configuration into target. This is useful for
// bootstrap settings (e.g. Nacos connection parameters) before constructing the
// application's Provider. Failure leaves target unchanged; success replaces it.
// Defaults/required tags and business validation remain the caller's concern.
func BindEnv(target any, prefix string) error {
	value := reflect.ValueOf(target)
	if !value.IsValid() || value.Kind() != reflect.Ptr || value.IsNil() || value.Elem().Kind() != reflect.Struct {
		return core.ErrEnvTarget
	}
	v, err := New(target, prefix).Load(context.Background())
	if err != nil {
		return err
	}
	candidate := reflect.New(value.Elem().Type())
	if err = v.Unmarshal(candidate.Interface()); err != nil {
		// Decoder errors may include password/token values from ENV.
		return core.ErrEnvDecode
	}
	value.Elem().Set(candidate.Elem())
	return nil
}

func structType(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}
func bindStructEnv(ctx context.Context, v *viper.Viper, t reflect.Type, path string, ancestors map[reflect.Type]bool) error {
	if ancestors[t] {
		return core.ErrEnvSchema
	}
	ancestors[t] = true
	defer delete(ancestors, t)
	for i := 0; i < t.NumField(); i++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		field := t.Field(i)
		if field.PkgPath != "" {
			continue
		} // unexported fields are not configuration
		tag := strings.Split(field.Tag.Get("mapstructure"), ",")
		name := tag[0]
		if name == "-" {
			continue
		}
		squash := false
		for _, option := range tag[1:] {
			switch option {
			case "squash":
				squash = true
			case "remain":
				return core.ErrEnvSchema
			}
		}
		fieldType := structType(field.Type)
		if squash {
			if fieldType.Kind() != reflect.Struct || field.Type.Kind() == reflect.Ptr {
				return core.ErrEnvSchema
			}
			if err := bindStructEnv(ctx, v, fieldType, path, ancestors); err != nil {
				return err
			}
			continue
		}
		if name == "" {
			name = field.Name
		}
		key := name
		if path != "" {
			key = path + "." + key
		}
		if fieldType.Kind() == reflect.Struct {
			if err := bindStructEnv(ctx, v, fieldType, key, ancestors); err != nil {
				return err
			}
		} else if err := v.BindEnv(key); err != nil {
			return core.ErrEnvSchema
		}
	}
	return nil
}

var _ provider.Provider = (*Provider)(nil)
