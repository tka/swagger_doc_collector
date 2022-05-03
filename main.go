package main

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"
)

var logger *zap.Logger
var config Config

func main() {
	initializeConfig()

	logger = initializeLogger()
	defer logger.Sync()

	downloader := Downloader{}
	downloader.Start()
	defer downloader.Stop()

	// Echo instance
	e := echo.New()

	// Middleware
	e.Use(ZapLogger(logger))
	e.Use(middleware.Recover())
	e.GET("/docs", listDocsHandler)
	e.GET("/docs/diff", docDiffDetailHandler)
	e.Static("/", "public")
	e.GET("/ws", websocketHandler)

	// Start server
	e.Logger.Fatal(e.Start(":1323"))
}
