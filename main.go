package main

import (
	"embed"
	"log"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

// webviewUserDataPath keeps the WebView2 user data folder in a dedicated,
// exe-name-independent location. The go-webview2 default is
// %APPDATA%\<exe name including extension>, which both fragments one folder
// per downloaded binary name and can hold stale browser state that newer
// WebView2 runtimes fail against (0x800700AA on controller creation).
// When the preferred location is not writable (restricted-integrity process
// contexts cannot create files under the user profile), the folder falls back
// to the executable's own directory, which shares the process integrity level.
func webviewUserDataPath() string {
	preferred := filepath.Join(localAppDataDir(), "GitRepoMirror", "WebView2")
	if dirWritable(preferred) {
		return preferred
	}
	if exe, err := os.Executable(); err == nil {
		alongside := filepath.Join(filepath.Dir(exe), "WebView2")
		if dirWritable(alongside) {
			return alongside
		}
	}
	return filepath.Join(os.TempDir(), "GitRepoMirror", "WebView2")
}

func localAppDataDir() string {
	if base := os.Getenv("LOCALAPPDATA"); base != "" {
		return base
	}
	return os.TempDir()
}

func dirWritable(dir string) bool {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return false
	}
	probe, err := os.CreateTemp(dir, ".writable-*")
	if err != nil {
		return false
	}
	name := probe.Name()
	probe.Close()
	os.Remove(name)
	return true
}

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
			WebviewUserDataPath:               webviewUserDataPath(),
			WebviewBrowserPath:                "",
			ZoomFactor:                        1,
			DisableFramelessWindowDecorations: false,
			IsZoomControlEnabled:              false,
		},
	}); err != nil {
		log.Fatal(err)
	}
}
