package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type OverlayConfig struct {
	Width                 int  `json:"width"`
	Height                int  `json:"height"`
	BackgroundTransparent bool `json:"background_transparent"`
}

type HistoryConfig struct {
	Enabled    bool `json:"enabled"`
	ShowBorder bool `json:"show_border"`
	X          int  `json:"x"`
	Y          int  `json:"y"`
	Width      int  `json:"width"`
	Height     int  `json:"height"`
	MaxEntries int  `json:"max_entries"`
}

type ControllerConfig struct {
	X      int    `json:"x"`
	Y      int    `json:"y"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Color  string `json:"color,omitempty"`
	Image  string `json:"image,omitempty"`
}

type ButtonDef struct {
	ID           string  `json:"id"`
	Label        string  `json:"label"`
	HistoryLabel string  `json:"history_label,omitempty"`
	HistoryColor string  `json:"history_color,omitempty"`
	Visible      *bool   `json:"visible,omitempty"`
	X            int     `json:"x"`
	Y            int     `json:"y"`
	Size         int     `json:"size"`
	Color        string  `json:"color,omitempty"`
	PressedColor string  `json:"pressed_color,omitempty"`
	TextColor    string  `json:"text_color,omitempty"`
	Opacity      float64 `json:"opacity,omitempty"`
}

type Config struct {
	Overlay    OverlayConfig                `json:"overlay"`
	History    HistoryConfig                `json:"history"`
	Controller ControllerConfig             `json:"controller"`
	Buttons    []ButtonDef                  `json:"buttons"`
	Mappings   map[string]map[string]string `json:"mappings,omitempty"`
}

func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Config
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

func Save(path string, c *Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0644)
}
