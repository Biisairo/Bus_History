package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type AppSettings struct {
	StoragePath string `json:"storagePath"`
	ServiceKey  string `json:"serviceKey"`
	StartHour   int    `json:"startHour"`  // 0-23
	EndHour     int    `json:"endHour"`    // 0-23
	IntervalMs  int    `json:"intervalMs"` // ms
}

func GetSettingsPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".bus_history", "settings.json")
}

func LoadAppSettings() (*AppSettings, error) {
	path := GetSettingsPath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return &AppSettings{}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var settings AppSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, err
	}

	return &settings, nil
}

func SaveAppSettings(settings *AppSettings) error {
	path := GetSettingsPath()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
