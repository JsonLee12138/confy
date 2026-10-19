package core

import (
	"time"

	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"
)

// DecodeOption adds RFC3339 time.Time decoding while preserving Viper's
// duration and comma-separated weak-slice conversions, including numeric slices.
// Apply it to every binding view, including temporary decrypted configurations.
func DecodeOption() viper.DecoderConfigOption {
	return viper.DecodeHook(mapstructure.ComposeDecodeHookFunc(
		mapstructure.StringToTimeDurationHookFunc(),
		mapstructure.StringToWeakSliceHookFunc(","),
		mapstructure.StringToTimeHookFunc(time.RFC3339Nano),
	))
}
