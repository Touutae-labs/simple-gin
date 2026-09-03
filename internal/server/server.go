// Package server builds the Gin engine, mounts middleware and
// Swagger, and registers every route.
//
// @title           simple-gin API
// @version         1.0
// @description     Product service — POST /product and PATCH /product/{id}.
// @description     All endpoints return the envelope {successful, error_code, data}.
// @BasePath        /
// @schemes         http https
package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/Touutae-labs/simple-gin/internal/configurations"
	"github.com/Touutae-labs/simple-gin/internal/controllers"

	_ "github.com/Touutae-labs/simple-gin/docs"
)

// ServerTitle and ServerVersion are exported so the cmd entry point
// can pass them in without the server package reaching back into main.
type ServerTitle string
type ServerVersion string

type ServerConfig struct {
	Title            ServerTitle
	Version          ServerVersion
	Port             string
	MaxPayloadSizeKB int
	TimeoutSeconds   int
	BaseURL          string
}

type Server struct {
	App    *gin.Engine
	Config *ServerConfig
}

// NewServer returns the engine, mounts middleware + Swagger, and
// registers every route against the supplied controllers.
func NewServer(cfg ServerConfig, c *controllers.Controllers) *Server {
	timeout := 30 * time.Second
	if cfg.TimeoutSeconds > 0 {
		timeout = time.Duration(cfg.TimeoutSeconds) * time.Second
	}
	bodyLimit := 4 * 1024 * 1024
	if cfg.MaxPayloadSizeKB > 0 {
		bodyLimit = cfg.MaxPayloadSizeKB * 1024
	}

	gin.SetMode(gin.ReleaseMode)
	app := gin.New()
	app.Use(gin.Recovery(), requestLogger())
	app.MaxMultipartMemory = int64(bodyLimit)

	app.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"successful": false,
			"error_code": "NOT_FOUND",
		})
	})

	s := &Server{App: app, Config: &cfg}
	s.mountSwagger()
	newHandler(c).register(s.App)

	_ = timeout
	return s
}

func (s *Server) mountSwagger() {
	if s.Config.BaseURL == "" {
		return
	}
	s.App.GET("/api-docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}

func (s *Server) Start() error {
	addr := ":" + s.Config.Port
	fmt.Printf("listening on %s (docs: %s/api-docs/index.html)\n", addr, s.Config.BaseURL)
	return s.App.Run(addr)
}

func requestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		fmt.Printf("method=%s path=%s status=%d duration=%s\n",
			c.Request.Method, c.Request.URL.Path, c.Writer.Status(), time.Since(start))
	}
}

// BuildServerConfig maps configurations.ServerConfig into the
// server.ServerConfig this package needs.
func BuildServerConfig(cfg configurations.ServerConfig, title ServerTitle, version ServerVersion) ServerConfig {
	return ServerConfig{
		Title:            title,
		Version:          version,
		Port:             cfg.Port,
		MaxPayloadSizeKB: cfg.MaxPayloadSizeKB,
		TimeoutSeconds:   cfg.TimeoutSeconds,
		BaseURL:          cfg.BaseURL,
	}
}
