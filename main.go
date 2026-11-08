package main

import (
	"embed"
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/getlantern/systray"
	"github.com/mkhsnw/gateprint/pkg/config"
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
	appKey := flag.String("app-key", "", "Save this App-Key to thermal_data.json before the server starts")
	port := flag.String("port", "", "Save this server port to thermal_data.json before the server starts")
	listenLocal := flag.Bool("listen-local", false, "Listen on 127.0.0.1 only (saved); -listen-local=false listens on all interfaces")
	flag.Parse()

	if err := persistStartupFlags(*appKey, *port, *listenLocal); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(2)
	}

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

// persistStartupFlags writes -app-key, -port and -listen-local into
// thermal_data.json when they were passed explicitly, so installers can
// configure GatePrint (e.g. `gateprint.exe -hidden -app-key X`) before the
// HTTP server reads its configuration. Flags that were not passed leave the
// stored values untouched.
func persistStartupFlags(appKey, port string, listenLocal bool) error {
	passed := map[string]bool{}
	flag.Visit(func(f *flag.Flag) { passed[f.Name] = true })
	if !passed["app-key"] && !passed["port"] && !passed["listen-local"] {
		return nil
	}

	if passed["port"] {
		portNumber, err := strconv.Atoi(port)
		if err != nil || portNumber < 1 || portNumber > 65535 {
			return fmt.Errorf("invalid -port %q: must be a number between 1 and 65535", port)
		}
	}

	config.InitStore()
	data := config.LoadData()
	if passed["app-key"] {
		data.AppKey = appKey
	}
	if passed["port"] {
		data.ServerPort = port
	}
	if passed["listen-local"] {
		data.ListenLocal = listenLocal
	}
	return config.SaveData(data)
}
