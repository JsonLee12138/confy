package main

import (
	"fmt"
	"github.com/creasty/defaults"
	"github.com/spf13/viper"
	"log"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/JsonLee12138/confy"
	"github.com/gin-gonic/gin"
)

// App 应用配置结构
type App struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Log      LogConfig      `mapstructure:"log"`
}

type ServerConfig struct {
	Port int    `mapstructure:"port" default:"8080"`
	Host string `mapstructure:"host" default:"0.0.0.0"`
	Mode string `mapstructure:"mode" default:"release"`
}

type DatabaseConfig struct {
	Host         string `mapstructure:"host" default:"127.0.0.1"`
	Port         int    `mapstructure:"port" default:"5432"`
	Name         string `mapstructure:"name" default:"myapp"`
	User         string `mapstructure:"user" default:"postgres"`
	Password     string `mapstructure:"password" default:"postgres"`
	MaxOpenConns int    `mapstructure:"max_open_conns" default:"10"`
	MaxIdleConns int    `mapstructure:"max_idle_conns" default:"5"`
}

type LogConfig struct {
	Level  string `mapstructure:"level" default:"info"`
	Format string `mapstructure:"format" default:"json"`
	Output string `mapstructure:"output" default:"stdout"`
}

// Validate 实现 confy.Validator 接口，用于配置校验
func (a *App) Validate() error {
	if a.Server.Port < 1 || a.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", a.Server.Port)
	}
	if a.Database.Port < 1 {
		return fmt.Errorf("invalid database port: %d", a.Database.Port)
	}
	return nil
}

func main() {
	var current atomic.Pointer[App]
	var publishMu sync.Mutex
	var cfg *confy.Config
	// Publish an independently decoded immutable view. Never read the target
	// struct while Confy may be updating it in the background.
	publish := func() error {
		publishMu.Lock()
		defer publishMu.Unlock()
		snapshot, err := cfg.Snapshot()
		if err != nil {
			return err
		}
		v := viper.New()
		if err = v.MergeConfigMap(snapshot); err != nil {
			return err
		}
		next := new(App)
		if err = v.Unmarshal(next); err != nil {
			return err
		}
		if err = defaults.Set(next); err != nil {
			return err
		}
		current.Store(next)
		return nil
	}
	var err error
	// --- 1. 初始化 confy ---
	cfg, err = confy.New(
		confy.WithPath("config"),
		confy.WithEnvPrefix("MYAPP"),
		confy.WithWatch(true), // 开启热更新
		confy.WithOnChange(func(e confy.Event) {
			if err := publish(); err != nil {
				log.Printf("[confy] view publication failed")
				return
			}
			log.Printf("[confy] config file changed: %s (op=%d)", e.Name, e.Op)
		}),
	)
	if err != nil {
		log.Fatalf("failed to init confy: %v", err)
	}

	defer cfg.Close()

	// --- 2. 绑定配置（含结构体默认值） ---
	var bound App
	if err := cfg.BindWithDefaults(&bound); err != nil {
		log.Fatalf("failed to bind config: %v", err)
	}

	// --- 3. 校验配置 ---
	if err := cfg.Validate(); err != nil {
		log.Fatalf("config validation failed: %v", err)
	}

	if err := publish(); err != nil {
		log.Fatal(err)
	}
	app := current.Load()

	log.Printf("loaded config: server=%s:%d, db=%s@%s:%d/%s, log=%s",
		app.Server.Host, app.Server.Port,
		app.Database.User, app.Database.Host, app.Database.Port, app.Database.Name,
		app.Log.Level,
	)

	// --- 4. 启动 Gin ---
	if app.Server.Mode == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()

	// GET /config — 查看当前配置（脱敏）
	r.GET("/config", func(c *gin.Context) {
		app := current.Load()
		safeDB := app.Database
		safeDB.Password = "***"
		c.JSON(http.StatusOK, gin.H{
			"server":   app.Server,
			"database": safeDB,
			"log":      app.Log,
		})
	})

	// GET /health — 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	addr := fmt.Sprintf("%s:%d", app.Server.Host, app.Server.Port)
	log.Printf("server starting at %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
