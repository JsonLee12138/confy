package confy

import "github.com/JsonLee12138/confy/provider/env"

// BindEnv binds environment variables to a struct without loading files.
// Prefix excludes the trailing underscore; use "" for no prefix.
// Failure leaves instance unchanged. It does not apply defaults or validation.
func BindEnv(instance any, prefix string) error {
	return env.BindEnv(instance, prefix)
}
