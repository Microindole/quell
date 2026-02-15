package main

import (
	"fmt"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

func startGUI() {
	app := NewApp()

	err := wails.Run(&options.App{
		Title:     "Quell",
		Width:     1200,
		Height:    800,
		MinWidth:  900,
		MinHeight: 600,
		Frameless: true,
		AssetServer: &assetserver.Options{
			Assets: webFS,
		},
		BackgroundColour: &options.RGBA{R: 32, G: 32, B: 32, A: 255},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
		Windows: &windows.Options{
			WebviewIsTransparent:              false,
			WindowIsTranslucent:               false,
			DisableFramelessWindowDecorations: false,
			Theme:                             windows.Dark,
		},
	})

	if err != nil {
		fmt.Printf("GUI 启动失败: %v\n", err)
		os.Exit(1)
	}
}
