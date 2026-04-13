package confy

import (
	"fmt"
	"reflect"
)

// Validate checks if the bound config implements the Validator interface and calls it.
func (c *Config) Validate() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.bound == nil {
		return nil
	}

	if v, ok := c.bound.(Validator); ok {
		if err := v.Validate(); err != nil {
			return fmt.Errorf("confy: validation failed: %w", err)
		}
	}

	return nil
}

// ValidateType checks struct fields with `required:"true"` are non-zero,
// and validates type compatibility via mapstructure tags.
func (c *Config) ValidateType(instance any) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	val := reflect.ValueOf(instance)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return ErrMustBeStruct
	}

	for i := 0; i < val.NumField(); i++ {
		field := val.Type().Field(i)
		fieldValue := val.Field(i)

		if tag := field.Tag.Get("required"); tag == "true" {
			if fieldValue.IsZero() {
				return fmt.Errorf("confy: required field %s is missing", field.Name)
			}
		}

		mapstructureTag := field.Tag.Get("mapstructure")
		if mapstructureTag != "" {
			configValue := c.v.Get(mapstructureTag)
			if configValue != nil && !fieldValue.Type().AssignableTo(reflect.TypeOf(configValue)) {
				return fmt.Errorf("confy: type mismatch for field %s: expected %s, got %T",
					field.Name, fieldValue.Type(), configValue)
			}
		}
	}

	return nil
}
