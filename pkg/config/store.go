package config

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
)

type AppData struct {
	AppKey         string `json:"app_key"`
	DefaultPrinter string `json:"default_printer"`
	ServerPort     string `json:"server_port"`
	AllowedOrigin  string `json:"allowed_origin"`
	// ListenLocal binds the HTTP server to 127.0.0.1 only when true.
	// Default (false) keeps the original behaviour of listening on all interfaces.
	ListenLocal bool `json:"listen_local,omitempty"`
}

var dataFilePath = "thermal_data.json"

// InitStore ensures the data file exists or gets the path
func InitStore() {
	configDir, err := os.UserConfigDir()
	if err == nil {
		appConfigDir := filepath.Join(configDir, "GatePrint")
		os.MkdirAll(appConfigDir, 0755)
		dataFilePath = filepath.Join(appConfigDir, "thermal_data.json")
	} else {
		execPath, err := os.Executable()
		if err == nil {
			dataFilePath = filepath.Join(filepath.Dir(execPath), "thermal_data.json")
		}
	}
}

// LoadData reads the saved configuration from disk
func LoadData() AppData {
	var data AppData
	file, err := os.ReadFile(dataFilePath)
	if err != nil {
		return data // returns empty struct if not exists
	}
	if err := json.Unmarshal(file, &data); err != nil {
		log.Printf("Error unmarshaling data: %v", err)
	}
	return data
}

// SaveData saves the configuration to disk
func SaveData(data AppData) error {
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(dataFilePath, bytes, 0644)
}
