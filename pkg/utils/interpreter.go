package utils

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
	"github.com/mkhsnw/gateprint/pkg/entity"
)

// Wrapper agar *printer.Printer punya method Read (syarat interface escpos)
type printerWrapper struct {
	*printer.Printer
}

func (w printerWrapper) Write(p []byte) (n int, err error) { return w.Printer.Write(p) }
func (w printerWrapper) Read(p []byte) (n int, err error)  { return 0, fmt.Errorf("read not supported") }

func formatRow(left, right string, width int) string {
	spaceNeeded := width - len(left) - len(right)
	if spaceNeeded < 1 {
		return left + " " + right
	}
	return left + strings.Repeat(" ", spaceNeeded) + right
}

func printImage(p *printer.Printer, img image.Image) {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	bytesWidth := (width + 7) / 8
	imgBW := make([]byte, bytesWidth*height)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			_, _, _, a := img.At(x, y).RGBA()
			if a > 0x7fff {
				c := color.GrayModel.Convert(img.At(x, y)).(color.Gray)
				if c.Y < 128 {
					pos := y*bytesWidth + x/8
					imgBW[pos] |= 0x80 >> (uint(x) % 8)
				}
			}
		}
	}

	cmd := []byte{0x1D, 0x76, 0x30, 0x00, byte(bytesWidth % 256), byte(bytesWidth / 256), byte(height % 256), byte(height / 256)}
	cmd = append(cmd, imgBW...)
	p.Write(cmd)
}

func ExecuteCustomPrint(job entity.PrintJob) error {
	printerDevice, err := printer.Open(job.PrinterName)
	if err != nil {
		return err
	}
	defer printerDevice.Close()

	printerDevice.StartDocument("CustomPrint", "RAW")
	printerDevice.StartPage()

	p, _ := escpos.NewPrinterByRW(&printerWrapper{printerDevice})
	p.Init()

	// Looping perintah yang dikirim dari JS
	for _, cmd := range job.Commands {
		switch cmd.Action {
		case "text":
			p.PrintLn(fmt.Sprintf("%v", cmd.Value))
		case "align":
			switch cmd.Value {
			case "center":
				p.Align(escpos.AlignCenter)
			case "right":
				p.Align(escpos.AlignRight)
			default:
				p.Align(escpos.AlignLeft)
			}
		case "size":
			// value dikirim [2, 2]
			if v, ok := cmd.Value.([]interface{}); ok && len(v) == 2 {
				p.Size(uint8(v[0].(float64)), uint8(v[1].(float64)))
			}
		case "image":
			base64Str := fmt.Sprintf("%v", cmd.Value)
			// Hapus prefix data:image/png;base64, jika ada
			if idx := strings.Index(base64Str, ","); idx != -1 {
				base64Str = base64Str[idx+1:]
			}
			imgData, err := base64.StdEncoding.DecodeString(base64Str)
			if err == nil {
				img, _, err := image.Decode(bytes.NewReader(imgData))
				if err == nil {
					printImage(printerDevice, img)
				} else {
					fmt.Println("Gagal decode image:", err)
				}
			} else {
				fmt.Println("Gagal decode base64:", err)
			}
		case "imageUrl":
			urlStr := fmt.Sprintf("%v", cmd.Value)

			// Gunakan client dengan timeout
			client := &http.Client{Timeout: 15 * time.Second}
			resp, err := client.Get(urlStr)
			if err != nil {
				fmt.Println("Network error:", err)
				continue
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				fmt.Println("URL error, status:", resp.StatusCode)
				continue
			}

			img, format, err := image.Decode(resp.Body)
			if err != nil {
				fmt.Println("Gagal decode gambar ("+format+"):", err)
				continue
			}
			printImage(printerDevice, img)
		case "qr":
			if v, ok := cmd.Value.(map[string]interface{}); ok {
				content := fmt.Sprintf("%v", v["content"])
				size := int(v["size"].(float64))
				p.QR(content, size)
			} else {
				p.QR(fmt.Sprintf("%v", cmd.Value), 6)
			}
		case "feed":
			p.Feed(3)
		case "underline": // Tambahkan ini
			if val, ok := cmd.Value.(bool); ok {
				p.Underline(val)
			}
		case "cut":
			p.Cut()
		}
	}

	p.End()
	printerDevice.EndPage()
	return nil
}
