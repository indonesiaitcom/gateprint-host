package service

import (
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"time"

	"github.com/alexbrainman/printer"
	"github.com/mect/go-escpos"
	"github.com/mkhsnw/gateprint/pkg/config"
	"github.com/mkhsnw/gateprint/pkg/entity"
)

var httpClient = &http.Client{Timeout: 15 * time.Second}

type printerWrapper struct {
	*printer.Printer
}

func (w printerWrapper) Write(p []byte) (n int, err error) { return w.Printer.Write(p) }
func (w printerWrapper) Read(p []byte) (n int, err error)  { return 0, fmt.Errorf("not supported") }

var defaultPrinterName string
var isPrinterInitialized bool

func initPrinterState() {
	if !isPrinterInitialized {
		config.InitStore()
		data := config.LoadData()
		defaultPrinterName = data.DefaultPrinter
		isPrinterInitialized = true
	}
}

func SetDefaultPrinter(name string) {
	initPrinterState()
	defaultPrinterName = name

	// Persist
	data := config.LoadData()
	data.DefaultPrinter = name
	config.SaveData(data)
}

func GetDefaultPrinter() string {
	initPrinterState()
	return defaultPrinterName
}

func ExecutePrint(job entity.PrintJob) error {

	if job.PrinterName == "" {
		if defaultPrinterName != "" {
			job.PrinterName = defaultPrinterName
		} else {
			return fmt.Errorf("printer_name is required and no default printer is set")
		}
	}

	pDev, err := printer.Open(job.PrinterName)
	if err != nil {
		return fmt.Errorf("could not open printer '%s': %v", job.PrinterName, err)
	}
	defer pDev.Close()

	pDev.StartDocument("Receipt", "RAW")
	pDev.StartPage()

	esc, _ := escpos.NewPrinterByRW(&printerWrapper{pDev})
	esc.Init()

	for _, cmd := range job.Commands {
		switch cmd.Action {
		case "text":
			esc.PrintLn(fmt.Sprintf("%v", cmd.Value))
		case "align":
			setAlign(esc, cmd.Value)
		case "size":
			if v, ok := cmd.Value.([]interface{}); ok && len(v) == 2 {
				esc.Size(uint8(v[0].(float64)), uint8(v[1].(float64)))
			}
		case "underline":
			if v, ok := cmd.Value.(bool); ok {
				esc.Underline(v)
			}
		case "imageUrl":
			handleImageUrl(pDev, esc, fmt.Sprintf("%v", cmd.Value))
		case "qr":
			handleQR(esc, cmd.Value)
		case "feed":
			if v, ok := cmd.Value.(float64); ok {
				esc.Feed(int(v))
			} else {
				esc.Feed(3)
			}
		case "cut":
			esc.Cut()
		}
	}

	esc.End()
	pDev.EndPage()
	return nil
}

func setAlign(esc *escpos.Printer, val interface{}) {
	switch val {
	case "center":
		esc.Align(escpos.AlignCenter)
	case "right":
		esc.Align(escpos.AlignRight)
	default:
		esc.Align(escpos.AlignLeft)
	}
}

func handleQR(esc *escpos.Printer, val interface{}) {
	if v, ok := val.(map[string]interface{}); ok {
		esc.QR(fmt.Sprintf("%v", v["content"]), int(v["size"].(float64)))
	} else {
		esc.QR(fmt.Sprintf("%v", val), 6)
	}
}

func handleImageUrl(pDev *printer.Printer, esc *escpos.Printer, url string) {
	resp, err := httpClient.Get(url)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	img, _, err := image.Decode(resp.Body)
	if err == nil {
		printImage(pDev, img)
		esc.PrintLn("")
	}
}

func printImage(p *printer.Printer, img image.Image) {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	bw := (w + 7) / 8
	data := make([]byte, bw*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := color.GrayModel.Convert(img.At(x, y)).(color.Gray)
			if c.Y < 128 {
				data[y*bw+x/8] |= 0x80 >> (uint(x) % 8)
			}
		}
	}
	p.Write(append([]byte{0x1D, 0x76, 0x30, 0x00, byte(bw % 256), byte(bw / 256), byte(h % 256), byte(h / 256)}, data...))
}
