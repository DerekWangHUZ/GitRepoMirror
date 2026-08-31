package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()
	if err := wails.Run(&options.App{
		Title:            "GitRepoMirror",
		Width:            1180,
		Height:           760,
		MinWidth:         900,
		MinHeight:        600,
		Frameless:        true,
		DisableResize:    false,
		BackgroundColour: &options.RGBA{R: 245, G: 245, B: 247, A: 1},
		AssetServer:      &assetserver.Options{Assets: assets},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Bind:             []interface{}{app},
		Windows: &windows.Options{
			WebviewIsTransparent:              false,
			WindowIsTranslucent:               false,
			DisableWindowIcon:                 true,
			WebviewUserDataPath:               "",
			WebviewBrowserPath:                "",
			ZoomFactor:                        1,
			DisableFramelessWindowDecorations: false,
			IsZoomControlEnabled:              false,
		},
	}); err != nil {
		log.Fatal(err)
	}
}
