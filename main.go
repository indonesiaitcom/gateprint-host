package main

import (
	"embed"
	"flag"

	"github.com/getlantern/systray"
	"github.com/mkhsnw/gateprint/pkg/windows"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/windows/icon.ico
var iconData []byte

func main() {
	startHidden := flag.Bool("hidden", false, "Start app hidden in tray")
	flag.Parse()

	app := NewApp()

	windows.HideConsoleWindow()

	go func() {
		systray.Run(func() {
			systray.SetIcon(iconData)
			systray.SetTitle("Gateprint")
			systray.SetTooltip("Gateprint API is running")

			mShow := systray.AddMenuItem("Show Window", "Show the Desktop UI")
			mQuit := systray.AddMenuItem("Quit", "Stop the server and quit")

			for {
				select {
				case <-mShow.ClickedCh:
					app.Show()
				case <-mQuit.ClickedCh:
					systray.Quit()
					app.Quit()
				}
			}
		}, func() {
		})
	}()

	err := wails.Run(&options.App{
		Title:             "Gateprint",
		Width:             600,
		Height:            800,
		HideWindowOnClose: true,
		StartHidden:       *startHidden,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
