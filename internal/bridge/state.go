package bridge

var ButtonIDs = []string{
	"up", "down", "left", "right",
	"b1", "b2", "b3", "b4",
	"l1", "l2", "r1", "r2",
	"s1", "s2", "l3", "r3",
	"a1", "a2",
}

type State struct {
	DeviceID string          `json:"device_id"`
	Buttons  map[string]bool `json:"buttons"`
}

func NewButtons() map[string]bool {
	buttons := make(map[string]bool, len(ButtonIDs))
	for _, id := range ButtonIDs {
		buttons[id] = false
	}
	return buttons
}

func CloneButtons(src map[string]bool) map[string]bool {
	dst := NewButtons()
	for id, pressed := range src {
		dst[id] = pressed
	}
	return dst
}
