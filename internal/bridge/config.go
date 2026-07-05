package bridge

type OverlayConfig struct {
	ComboDisplay ComboDisplayConfig `json:"combo_display"`
	ComboAudio   ComboAudioConfig   `json:"combo_audio"`
	Controller   ControllerConfig   `json:"controller"`
	Buttons      []ButtonConfig     `json:"buttons"`
}

type ComboDisplayConfig struct {
	Enabled    bool `json:"enabled"`
	ShowBorder bool `json:"show_border"`
	X          int  `json:"x"`
	Y          int  `json:"y"`
	Width      int  `json:"width"`
	Height     int  `json:"height"`
}

type ComboAudioConfig struct {
	Volume float64 `json:"volume"`
}

type ComboResponse struct {
	Current        ComboSelection      `json:"current"`
	Files          []ComboFileSummary  `json:"files"`
	ActiveSet      *ComboSet           `json:"active_set,omitempty"`
	Commands       []ComboCommand      `json:"commands,omitempty"`
	Moves          []MoveDefinition    `json:"moves,omitempty"`
	Practice       PracticeConfig      `json:"practice,omitempty"`
	PracticeSets   []PracticeSet       `json:"practiceSets,omitempty"`
	ActivePractice ActivePracticeState `json:"activePractice,omitempty"`
}

type PracticeMode string

const (
	PracticeModeFocus      PracticeMode = "focus"
	PracticeModePlaylist   PracticeMode = "playlist"
	PracticeModeAutoDetect PracticeMode = "auto_detect"
)

type PracticeConfig struct {
	Mode         PracticeMode `json:"mode,omitempty"`
	ActiveRecipe string       `json:"activeRecipe,omitempty"`
	ActiveSet    string       `json:"activeSet,omitempty"`
}

type PracticeSet struct {
	ID                string   `json:"id"`
	Name              string   `json:"name"`
	Mode              string   `json:"mode,omitempty"`
	Recipes           []string `json:"recipes"`
	Loop              bool     `json:"loop,omitempty"`
	AdvanceOnComplete bool     `json:"advanceOnComplete,omitempty"`
}

type ActivePracticeState struct {
	Mode           PracticeMode `json:"mode"`
	ActiveRecipeID string       `json:"activeRecipeId,omitempty"`
	ActiveSetID    string       `json:"activeSetId,omitempty"`
	ActiveSetIndex int          `json:"activeSetIndex,omitempty"`
}

type ComboCommand struct {
	ID             string              `json:"id"`
	Name           string              `json:"name"`
	Notation       string              `json:"notation"`
	Sequence       []ComboCommandInput `json:"sequence,omitempty"`
	MaxFrames      int                 `json:"maxFrames,omitempty"`
	MaxGapFrames   int                 `json:"maxGapFrames,omitempty"`
	Priority       int                 `json:"priority,omitempty"`
	CooldownFrames int                 `json:"cooldownFrames,omitempty"`
}

type ComboCommandInput struct {
	Dir          string `json:"dir,omitempty"`
	Button       string `json:"button,omitempty"`
	RequirePress bool   `json:"requirePress,omitempty"`
}

type MoveDefinition struct {
	ID                          string         `json:"id"`
	Name                        string         `json:"name"`
	Notation                    string         `json:"notation"`
	Category                    string         `json:"category,omitempty"`
	Strength                    string         `json:"strength,omitempty"`
	Input                       string         `json:"input,omitempty"`
	CommandID                   string         `json:"command,omitempty"`
	Tags                        []string       `json:"tags,omitempty"`
	Startup                     int            `json:"startup,omitempty"`
	Active                      int            `json:"active,omitempty"`
	Recovery                    int            `json:"recovery,omitempty"`
	ActiveFrames                string         `json:"activeFrames,omitempty"`
	RecoveryFrames              string         `json:"recoveryFrames,omitempty"`
	TotalFrames                 string         `json:"totalFrames,omitempty"`
	HitAdvantage                string         `json:"hitAdvantage,omitempty"`
	GuardAdvantage              string         `json:"guardAdvantage,omitempty"`
	Cancel                      string         `json:"cancel,omitempty"`
	Damage                      string         `json:"damage,omitempty"`
	ComboScaling                []string       `json:"comboScaling,omitempty"`
	DriveGaugeGainHit           string         `json:"driveGaugeGainHit,omitempty"`
	DriveGaugeLossGuard         string         `json:"driveGaugeLossGuard,omitempty"`
	DriveGaugeLossPunishCounter string         `json:"driveGaugeLossPunishCounter,omitempty"`
	SuperArtGaugeGain           string         `json:"superArtGaugeGain,omitempty"`
	Attribute                   string         `json:"attribute,omitempty"`
	Notes                       []string       `json:"notes,omitempty"`
	CancelWindows               []CancelWindow `json:"cancelWindows,omitempty"`
}

type CancelWindow struct {
	Type       string   `json:"type"`
	Start      int      `json:"start"`
	End        int      `json:"end"`
	Targets    []string `json:"targets,omitempty"`
	TargetTags []string `json:"targetTags,omitempty"`
}

type ComboSelection struct {
	File              string       `json:"file"`
	SetID             string       `json:"set_id"`
	Mode              PracticeMode `json:"mode,omitempty"`
	ActiveRecipe      string       `json:"activeRecipe,omitempty"`
	ActiveSet         string       `json:"activeSet,omitempty"`
	ActiveSetIndex    int          `json:"activeSetIndex,omitempty"`
	Loop              *bool        `json:"loop,omitempty"`
	AdvanceOnComplete *bool        `json:"advanceOnComplete,omitempty"`
}

type ComboFileSummary struct {
	File  string            `json:"file"`
	Title string            `json:"title"`
	Sets  []ComboSetSummary `json:"sets"`
}

type ComboSetSummary struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Mode string `json:"mode"`
}

type ComboSet struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Mode   string  `json:"mode"`
	Combos []Combo `json:"combos"`
}

type Combo struct {
	ID       string      `json:"id,omitempty"`
	Name     string      `json:"name"`
	Notation string      `json:"notation,omitempty"`
	Priority int         `json:"priority,omitempty"`
	Steps    []ComboStep `json:"steps"`
}

type ComboStep struct {
	MoveID          string   `json:"move,omitempty"`
	Command         string   `json:"command,omitempty"`
	Notation        string   `json:"notation,omitempty"`
	Label           string   `json:"label,omitempty"`
	Direction       string   `json:"direction,omitempty"`
	Buttons         []string `json:"buttons,omitempty"`
	GapWindowFrames int      `json:"gap_window_frames,omitempty"`
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
