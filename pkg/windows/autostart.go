package windows

import (
	"fmt"
	"os"
	"syscall"

	"golang.org/x/sys/windows/registry"
)

const AutoStartKey = `Software\Microsoft\Windows\CurrentVersion\Run`
const AppName = "Gateprint"

func IsAutoStartEnabled() bool {
	key, err := registry.OpenKey(registry.CURRENT_USER, AutoStartKey, registry.READ)
	if err != nil {
		return false
	}
	defer key.Close()

	_, _, err = key.GetStringValue(AppName)
	return err == nil
}

func EnableAutoStart() error {
	exePath, err := os.Executable()
	if err != nil {
		return err
	}

	key, _, err := registry.CreateKey(registry.CURRENT_USER, AutoStartKey, registry.WRITE)
	if err != nil {
		return err
	}
	defer key.Close()

	// Menambahkan argumen --hidden agar saat startup aplikasi tidak muncul GUI
	value := fmt.Sprintf(`"%s" --hidden`, exePath)
	return key.SetStringValue(AppName, value)
}

func DisableAutoStart() error {
	key, err := registry.OpenKey(registry.CURRENT_USER, AutoStartKey, registry.WRITE)
	if err != nil {
		return err
	}
	defer key.Close()

	return key.DeleteValue(AppName)
}

func HideConsoleWindow() {
	consoleWnd := syscall.MustLoadDLL("kernel32.dll").MustFindProc("GetConsoleWindow")
	hwnd, _, _ := consoleWnd.Call()
	if hwnd != 0 {
		showWindow := syscall.MustLoadDLL("user32.dll").MustFindProc("ShowWindow")
		showWindow.Call(hwnd, 0)
	}
}
