package main

import (
	"embed"
	"io/ioutil"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"
)

var logger *zap.Logger
var config Config

//go:embed  public/*
var embededFiles embed.FS

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
	e.GET("/", func(c echo.Context) error {
		body, err := ioutil.ReadFile("public/index.html")
		//body, err := embededFiles.ReadFile("public/index.html")
		if err != nil {
			c.HTML(http.StatusInternalServerError, err.Error())
		}
		c.HTML(http.StatusOK, string(body))
		return nil
	})
	e.GET("/ws", websocketHandler)

	// Start server
	e.Logger.Fatal(e.Start(":1323"))
}
