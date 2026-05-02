package service

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"strings"
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
		
		if defaultPrinterName == "" {
			if defPrinter, err := printer.Default(); err == nil && defPrinter != "" {
				defaultPrinterName = defPrinter
				data.DefaultPrinter = defaultPrinterName
				config.SaveData(data)
			} else {
				names, err := printer.ReadNames()
				if err == nil && len(names) > 0 {
					defaultPrinterName = names[0]
					data.DefaultPrinter = defaultPrinterName
					config.SaveData(data)
				}
			}
		}
		
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

	if err := pDev.StartDocument("Receipt", "RAW"); err != nil {
		return fmt.Errorf("failed to start document: %v", err)
	}
	if err := pDev.StartPage(); err != nil {
		return fmt.Errorf("failed to start page: %v", err)
	}

	esc, _ := escpos.NewPrinterByRW(&printerWrapper{pDev})
	if err := esc.Init(); err != nil {
		return fmt.Errorf("failed to init escpos: %v", err)
	}

	for _, cmd := range job.Commands {
		var cmdErr error
		switch cmd.Action {
		case "text":
			cmdErr = esc.PrintLn(fmt.Sprintf("%v", cmd.Value))
		case "align":
			cmdErr = setAlign(esc, cmd.Value)
		case "size":
			if v, ok := cmd.Value.([]interface{}); ok && len(v) == 2 {
				cmdErr = esc.Size(uint8(v[0].(float64)), uint8(v[1].(float64)))
			}
		case "underline":
			if v, ok := cmd.Value.(bool); ok {
				cmdErr = esc.Underline(v)
			}
		case "imageUrl":
			cmdErr = handleImageUrl(pDev, esc, fmt.Sprintf("%v", cmd.Value))
		case "image":
			cmdErr = handleImageBase64(pDev, esc, fmt.Sprintf("%v", cmd.Value))
		case "qr":
			cmdErr = handleQR(esc, cmd.Value)
		case "feed":
			if v, ok := cmd.Value.(float64); ok {
				cmdErr = esc.Feed(int(v))
			} else {
				cmdErr = esc.Feed(3)
			}
		case "cut":
			cmdErr = esc.Cut()
		case "barcode":
			cmdErr = handleBarcode(esc, cmd.Value)
		case "pdf417":
			cmdErr = handlePdf417(esc, cmd.Value)
		case "datamatrix":
			cmdErr = handleDataMatrix(esc, cmd.Value)
		case "smooth":
			if v, ok := cmd.Value.(bool); ok {
				cmdErr = esc.Smooth(v)
			}
		default:
			cmdErr = fmt.Errorf("unknown action: %s", cmd.Action)
		}

		if cmdErr != nil {
			return fmt.Errorf("failed on command '%s': %v", cmd.Action, cmdErr)
		}
	}

	if err := esc.End(); err != nil {
		return fmt.Errorf("failed to end escpos: %v", err)
	}
	if err := pDev.EndPage(); err != nil {
		return fmt.Errorf("failed to end page: %v", err)
	}
	if err := pDev.EndDocument(); err != nil {
		return fmt.Errorf("failed to end document: %v", err)
	}
	return nil
}

func setAlign(esc *escpos.Printer, val interface{}) error {
	switch val {
	case "center":
		return esc.Align(escpos.AlignCenter)
	case "right":
		return esc.Align(escpos.AlignRight)
	default:
		return esc.Align(escpos.AlignLeft)
	}
}

func handleQR(esc *escpos.Printer, val interface{}) error {
	if v, ok := val.(map[string]interface{}); ok {
		return esc.QR(fmt.Sprintf("%v", v["content"]), int(v["size"].(float64)))
	}
	return esc.QR(fmt.Sprintf("%v", val), 6)
}

func handleImageUrl(pDev *printer.Printer, esc *escpos.Printer, url string) error {
	resp, err := httpClient.Get(url)
	if err != nil {
		return fmt.Errorf("failed to fetch image: %v", err)
	}
	defer resp.Body.Close()
	img, _, err := image.Decode(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to decode image: %v", err)
	}
	if err := printImage(pDev, img); err != nil {
		return fmt.Errorf("failed to print image: %v", err)
	}
	return esc.PrintLn("")
}

func printImage(p *printer.Printer, img image.Image) error {
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
	_, err := p.Write(append([]byte{0x1D, 0x76, 0x30, 0x00, byte(bw % 256), byte(bw / 256), byte(h % 256), byte(h / 256)}, data...))
	return err
}

func handleImageBase64(pDev *printer.Printer, esc *escpos.Printer, b64Data string) error {
	if idx := strings.Index(b64Data, "base64,"); idx != -1 {
		b64Data = b64Data[idx+7:]
	}

	imgBytes, err := base64.StdEncoding.DecodeString(b64Data)
	if err != nil {
		return fmt.Errorf("failed to decode base64 string: %v", err)
	}

	img, _, err := image.Decode(bytes.NewReader(imgBytes))
	if err != nil {
		return fmt.Errorf("failed to decode image data: %v", err)
	}

	if err := printImage(pDev, img); err != nil {
		return fmt.Errorf("failed to print base64 image: %v", err)
	}
	return esc.PrintLn("")
}

func handleBarcode(esc *escpos.Printer, val interface{}) error {
	if v, ok := val.(map[string]interface{}); ok {
		content := fmt.Sprintf("%v", v["content"])

		bType := "CODE39"
		if t, ok := v["type"].(string); ok {
			bType = t
		}

		switch bType {
		case "UPC-A":
			return esc.Barcode(content, escpos.BarcodeTypeUPCA)
		case "UPC-E":
			return esc.Barcode(content, escpos.BarcodeTypeUPCE)
		case "EAN13":
			return esc.Barcode(content, escpos.BarcodeTypeEAN13)
		case "EAN8":
			return esc.Barcode(content, escpos.BarcodeTypeEAN8)
		case "CODE39":
			return esc.Barcode(content, escpos.BarcodeTypeCODE39)
		case "ITF":
			return esc.Barcode(content, escpos.BarcodeTypeITF)
		case "CODE128":
			return esc.Barcode(content, escpos.BarcodeTypeCODE128)
		case "CODABAR":
			return esc.Barcode(content, escpos.BarcodeTypeCODABAR)
		default:
			return esc.Barcode(content, escpos.BarcodeTypeCODE39)
		}
	}
	return fmt.Errorf("invalid barcode value type")
}

func handlePdf417(esc *escpos.Printer, val interface{}) error {
	if v, ok := val.(map[string]interface{}); ok {
		size := 3
		if s, ok := v["size"].(float64); ok {
			size = int(s)
		}
		return esc.PDF417(fmt.Sprintf("%v", v["content"]), size)
	}
	return esc.PDF417(fmt.Sprintf("%v", val), 3)
}

func handleDataMatrix(esc *escpos.Printer, val interface{}) error {
	if v, ok := val.(map[string]interface{}); ok {
		size := 3
		if s, ok := v["size"].(float64); ok {
			size = int(s)
		}
		return esc.DataMatrix(fmt.Sprintf("%v", v["content"]), size)
	}
	return esc.DataMatrix(fmt.Sprintf("%v", val), 3)
}
