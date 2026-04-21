package main

import (
	"context"
	"fmt"

	"github.com/alexbrainman/printer"
	"github.com/mkhsnw/gateprint/pkg/entity"
	"github.com/mkhsnw/gateprint/pkg/server"
	"github.com/mkhsnw/gateprint/pkg/service"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx context.Context
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	go func() {
		server.StartServer(ctx)
	}()
}

func (a *App) GetPrinters() ([]string, error) {
	names, err := printer.ReadNames()
	if err != nil {
		return nil, fmt.Errorf("failed to read printers: %v", err)
	}
	return names, nil
}

func (a *App) GetSelectedPrinter() string {
	return service.GetDefaultPrinter()
}

func (a *App) SetSelectedPrinter(name string) {
	service.SetDefaultPrinter(name)
}

func (a *App) PrintTest(name string) error {
	job := entity.PrintJob{
		PrinterName: name,
		Commands: []entity.PrintCommand{
			{Action: "align", Value: "center"},
			{Action: "text", Value: "--- TEST PRINT ---"},
			{Action: "text", Value: "Thermal Bridge Desktop UI"},
			{Action: "text", Value: "Status: OK"},
			{Action: "feed", Value: 3.0},
			{Action: "cut", Value: nil},
		},
	}
	return service.ExecutePrint(job)
}

func (a *App) GetAppKey() string {
	return server.GetAppKey()
}

func (a *App) SetAppKey(key string) {
	server.SetAppKey(key)
}

func (a *App) GetServerPort() string {
	return server.GetServerPort()
}

func (a *App) SetServerPort(port string) {
	server.SetServerPort(port)
}

// Show the application window
func (a *App) Show() {
	if a.ctx != nil {
		runtime.WindowShow(a.ctx)
	}
}

// Quit the application entirely
func (a *App) Quit() {
	if a.ctx != nil {
		runtime.Quit(a.ctx)
	}
}
