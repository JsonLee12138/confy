// Read a single Nacos configuration with the same Bind/ENV API as local files.
package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"strconv"

	"github.com/JsonLee12138/confy"
	"github.com/JsonLee12138/confy/provider/nacos"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
)

type App struct {
	Server struct {
		Host string `mapstructure:"host" default:"127.0.0.1"`
		Port int    `mapstructure:"port" default:"8080"`
	} `mapstructure:"server"`
	Database struct {
		Password string `mapstructure:"password"`
	} `mapstructure:"database"`
}

// Validate rejects server ports outside the TCP port range when explicitly called by Config.
func (a *App) Validate() error {
	if a.Server.Port < 1 || a.Server.Port > 65535 {
		return fmt.Errorf("invalid server port")
	}
	return nil
}

// main reads one Nacos document using connection settings from ENV and validates the bound app.
// It does not enable watching or print database credentials.
func main() {
	address := os.Getenv("NACOS_SERVER_ADDR")
	if address == "" {
		address = "127.0.0.1:8848"
	}
	host, portText, err := net.SplitHostPort(address)
	if err != nil {
		log.Fatal("NACOS_SERVER_ADDR must be host:port")
	}
	port, err := strconv.ParseUint(portText, 10, 16)
	if err != nil || port == 0 {
		log.Fatal("invalid Nacos server port")
	}
	dataID := os.Getenv("NACOS_DATA_ID")
	if dataID == "" {
		dataID = "app.yaml"
	}
	cfg, err := confy.New(
		confy.WithProvider(nacos.New(
			nacos.WithServers([]constant.ServerConfig{{IpAddr: host, Port: port}}),
			nacos.WithClientConfig(constant.ClientConfig{
				NamespaceId: os.Getenv("NACOS_NAMESPACE_ID"),
				Username:    os.Getenv("NACOS_USERNAME"),
				Password:    os.Getenv("NACOS_PASSWORD"),
				TimeoutMs:   5000,
			}),
			nacos.WithDataID(dataID),
			nacos.WithGroup(os.Getenv("NACOS_GROUP")),
			nacos.WithFormat(confy.YAML),
		)),
		confy.WithEnvPrefix("MYAPP"),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer cfg.Close()
	var app App
	if err = cfg.Bind(&app); err != nil {
		log.Fatal(err)
	}
	if err = cfg.Validate(); err != nil {
		log.Fatal(err)
	}
	// For example, MYAPP_SERVER_PORT replaces the server.port value from Nacos.
	// The password is read from configuration/ENV but never printed.
	fmt.Printf("server=%s:%d\n", app.Server.Host, app.Server.Port)
}
