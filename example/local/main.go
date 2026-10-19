// Run from example/: go run ./local
package main

import (
	"fmt"
	"log"

	"github.com/JsonLee12138/confy"
	"github.com/JsonLee12138/confy/provider/local"
)

// main reads inherited local configuration and applies MYAPP environment overrides.
func main() {
	cfg, err := confy.New(
		confy.WithProvider(local.New(local.WithPath("config"), local.WithFile("config"), local.WithLoadAll(false))),
		confy.WithEnvPrefix("MYAPP"),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer cfg.Close()
	var app struct {
		Server struct {
			Host string `mapstructure:"host"`
			Port int    `mapstructure:"port" default:"8080"`
		} `mapstructure:"server"`
	}
	if err := cfg.Bind(&app); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("server=%s:%d\n", app.Server.Host, app.Server.Port)
}
