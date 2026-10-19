package e2e_test

type e2eSettings struct {
	Name    string `mapstructure:"name"`
	Removed string `mapstructure:"removed"`
	Timeout int    `mapstructure:"timeout" default:"30"`
	Enabled bool   `mapstructure:"enabled" default:"true"`
	Server  struct {
		Port int `mapstructure:"port"`
	} `mapstructure:"server"`
}
