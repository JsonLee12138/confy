// Run from example/: MYAPP_SERVER_HOST=127.0.0.1 MYAPP_SERVER_PORT=9090 go run ./env
package main

import (
	"fmt"
	"log"

	"github.com/JsonLee12138/confy"
	"github.com/JsonLee12138/confy/provider/env"
)

type App struct {
	Server struct {
		Host string `mapstructure:"host"`
		Port int    `mapstructure:"port" default:"8080"`
	} `mapstructure:"server"`
}

// main demonstrates direct ENV bootstrap binding and the ENV Provider Config path.
func main() {
	// Direct binding is useful for connection settings before creating a Provider.
	var bootstrap App
	if err := confy.BindEnv(&bootstrap, "MYAPP"); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("direct=%s:%d\n", bootstrap.Server.Host, bootstrap.Server.Port)

	// ENV can also be a Config source and use the common Bind/defaults pipeline.
	cfg, err := confy.New(confy.WithProvider(env.New(App{}, "MYAPP")))
	if err != nil {
		log.Fatal(err)
	}
	defer cfg.Close()
	var app App
	if err := cfg.Bind(&app); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("server=%s:%d\n", app.Server.Host, app.Server.Port)
}
