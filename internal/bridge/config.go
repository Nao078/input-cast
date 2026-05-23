package bridge

type OverlayConfig struct {
	Controller ControllerConfig `json:"controller"`
	Buttons    []ButtonConfig   `json:"buttons"`
}

type ControllerConfig struct {
	X      int    `json:"x"`
	Y      int    `json:"y"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Color  string `json:"color,omitempty"`
	Image  string `json:"image,omitempty"`
}

type ButtonConfig struct {
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

func (b ButtonConfig) IsVisible() bool {
	return b.Visible == nil || *b.Visible
}
