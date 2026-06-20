package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	appBindings, err := NewApp()
	if err != nil {
		log.Fatalf("init: %v", err)
	}

	mainApp := application.New(application.Options{
		Name:        "goscan-ui",
		Description: "Interface goscan — findings, scripts, scan",
		Assets: application.AssetOptions{
			Handler: application.BundledAssetFileServer(assets),
		},
		Services: []application.Service{
			application.NewService(appBindings),
		},
		PostShutdown: appBindings.Shutdown,
	})

	appBindings.wails = mainApp

	mainApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:     "goscan",
		Width:     1400,
		Height:    900,
		MinWidth:  1024,
		MinHeight: 640,
		URL:       "/",
	})

	if err := mainApp.Run(); err != nil {
		log.Fatalf("run: %v", err)
	}
}
