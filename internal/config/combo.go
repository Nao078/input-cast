package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type ComboFile struct {
	Version      int              `json:"version,omitempty" yaml:"version,omitempty"`
	Game         string           `json:"game,omitempty" yaml:"game,omitempty"`
	Character    string           `json:"character,omitempty" yaml:"character,omitempty"`
	Title        string           `json:"title" yaml:"title"`
	Commands     []ComboCommand   `json:"commands,omitempty" yaml:"commands,omitempty"`
	Moves        []MoveDefinition `json:"moves,omitempty" yaml:"moves,omitempty"`
	Recipes      []ComboRecipe    `json:"recipes,omitempty" yaml:"recipes,omitempty"`
	Practice     PracticeConfig   `json:"practice,omitempty" yaml:"practice,omitempty"`
	PracticeSets []PracticeSet    `json:"practiceSets,omitempty" yaml:"practiceSets,omitempty"`
	Sets         []ComboSet       `json:"sets" yaml:"sets"`
}

type PracticeMode string

const (
	PracticeModeFocus      PracticeMode = "focus"
	PracticeModePlaylist   PracticeMode = "playlist"
	PracticeModeAutoDetect PracticeMode = "auto_detect"
)

type PracticeConfig struct {
	Mode         PracticeMode `json:"mode,omitempty" yaml:"mode,omitempty"`
	ActiveRecipe string       `json:"activeRecipe,omitempty" yaml:"activeRecipe,omitempty"`
	ActiveSet    string       `json:"activeSet,omitempty" yaml:"activeSet,omitempty"`
}

type PracticeSet struct {
	ID                string   `json:"id" yaml:"id"`
	Name              string   `json:"name" yaml:"name"`
	Mode              string   `json:"mode,omitempty" yaml:"mode,omitempty"`
	Recipes           []string `json:"recipes" yaml:"recipes"`
	Loop              bool     `json:"loop,omitempty" yaml:"loop,omitempty"`
	AdvanceOnComplete bool     `json:"advanceOnComplete,omitempty" yaml:"advanceOnComplete,omitempty"`
}

type ActivePracticeState struct {
	Mode           PracticeMode `json:"mode"`
	ActiveRecipeID string       `json:"activeRecipeId,omitempty"`
	ActiveSetID    string       `json:"activeSetId,omitempty"`
	ActiveSetIndex int          `json:"activeSetIndex,omitempty"`
}

type ComboCommand struct {
	ID             string              `json:"id" yaml:"id"`
	Name           string              `json:"name" yaml:"name"`
	Notation       string              `json:"notation" yaml:"notation"`
	Sequence       []ComboCommandInput `json:"sequence,omitempty" yaml:"sequence,omitempty"`
	MaxFrames      int                 `json:"maxFrames,omitempty" yaml:"maxFrames,omitempty"`
	MaxGapFrames   int                 `json:"maxGapFrames,omitempty" yaml:"maxGapFrames,omitempty"`
	Priority       int                 `json:"priority,omitempty" yaml:"priority,omitempty"`
	CooldownFrames int                 `json:"cooldownFrames,omitempty" yaml:"cooldownFrames,omitempty"`
}

type ComboCommandInput struct {
	Dir          string `json:"dir,omitempty" yaml:"dir,omitempty"`
	Button       string `json:"button,omitempty" yaml:"button,omitempty"`
	RequirePress bool   `json:"requirePress,omitempty" yaml:"requirePress,omitempty"`
}

type MoveDefinition struct {
	ID       string `json:"id" yaml:"id"`
	Name     string `json:"name" yaml:"name"`
	Notation string `json:"notation" yaml:"notation"`
	Category string `json:"category,omitempty" yaml:"category,omitempty"`
	Strength string `json:"strength,omitempty" yaml:"strength,omitempty"`

	Input     string   `json:"input,omitempty" yaml:"input,omitempty"`
	CommandID string   `json:"command,omitempty" yaml:"command,omitempty"`
	Tags      []string `json:"tags,omitempty" yaml:"tags,omitempty"`

	Startup        int    `json:"startup,omitempty" yaml:"startup,omitempty"`
	Active         int    `json:"active,omitempty" yaml:"active,omitempty"`
	Recovery       int    `json:"recovery,omitempty" yaml:"recovery,omitempty"`
	ActiveFrames   string `json:"activeFrames,omitempty" yaml:"activeFrames,omitempty"`
	RecoveryFrames string `json:"recoveryFrames,omitempty" yaml:"recoveryFrames,omitempty"`
	TotalFrames    string `json:"totalFrames,omitempty" yaml:"totalFrames,omitempty"`

	HitAdvantage   string   `json:"hitAdvantage,omitempty" yaml:"hitAdvantage,omitempty"`
	GuardAdvantage string   `json:"guardAdvantage,omitempty" yaml:"guardAdvantage,omitempty"`
	Cancel         string   `json:"cancel,omitempty" yaml:"cancel,omitempty"`
	Damage         string   `json:"damage,omitempty" yaml:"damage,omitempty"`
	ComboScaling   []string `json:"comboScaling,omitempty" yaml:"comboScaling,omitempty"`

	DriveGaugeGainHit           string   `json:"driveGaugeGainHit,omitempty" yaml:"driveGaugeGainHit,omitempty"`
	DriveGaugeLossGuard         string   `json:"driveGaugeLossGuard,omitempty" yaml:"driveGaugeLossGuard,omitempty"`
	DriveGaugeLossPunishCounter string   `json:"driveGaugeLossPunishCounter,omitempty" yaml:"driveGaugeLossPunishCounter,omitempty"`
	SuperArtGaugeGain           string   `json:"superArtGaugeGain,omitempty" yaml:"superArtGaugeGain,omitempty"`
	Attribute                   string   `json:"attribute,omitempty" yaml:"attribute,omitempty"`
	Notes                       []string `json:"notes,omitempty" yaml:"notes,omitempty"`

	CancelWindows []CancelWindow `json:"cancelWindows,omitempty" yaml:"cancelWindows,omitempty"`
}

type CancelWindow struct {
	Type       string   `json:"type" yaml:"type"`
	Start      int      `json:"start" yaml:"start"`
	End        int      `json:"end" yaml:"end"`
	Targets    []string `json:"targets,omitempty" yaml:"targets,omitempty"`
	TargetTags []string `json:"targetTags,omitempty" yaml:"targetTags,omitempty"`
}

func (s *ComboCommandInput) UnmarshalYAML(value *yaml.Node) error {
	var raw struct {
		Dir          interface{} `yaml:"dir,omitempty"`
		Button       string      `yaml:"button,omitempty"`
		RequirePress bool        `yaml:"requirePress,omitempty"`
	}
	if err := value.Decode(&raw); err != nil {
		return err
	}
	switch dir := raw.Dir.(type) {
	case int:
		s.Dir = fmt.Sprintf("%d", dir)
	case int64:
		s.Dir = fmt.Sprintf("%d", dir)
	case string:
		s.Dir = strings.TrimSpace(dir)
	case nil:
		s.Dir = ""
	default:
		s.Dir = fmt.Sprintf("%v", dir)
	}
	s.Button = strings.TrimSpace(raw.Button)
	s.RequirePress = raw.RequirePress
	return nil
}

type ComboRecipe struct {
	ID       string            `json:"id" yaml:"id"`
	Name     string            `json:"name" yaml:"name"`
	Notation string            `json:"notation,omitempty" yaml:"notation,omitempty"`
	Steps    []ComboRecipeStep `json:"steps" yaml:"steps"`
	Priority int               `json:"priority,omitempty" yaml:"priority,omitempty"`
}

type ComboRecipeStep struct {
	MoveID   string `json:"move,omitempty" yaml:"move,omitempty"`
	Input    string `json:"input,omitempty" yaml:"input,omitempty"`
	Command  string `json:"command,omitempty" yaml:"command,omitempty"`
	Label    string `json:"label,omitempty" yaml:"label,omitempty"`
	Optional bool   `json:"optional,omitempty" yaml:"optional,omitempty"`
}

type ComboSet struct {
	ID     string  `json:"id" yaml:"id"`
	Name   string  `json:"name" yaml:"name"`
	Mode   string  `json:"mode" yaml:"mode"`
	Combos []Combo `json:"combos" yaml:"combos"`
}

type Combo struct {
	ID       string      `json:"id,omitempty" yaml:"id,omitempty"`
	Name     string      `json:"name" yaml:"name"`
	Notation string      `json:"notation,omitempty" yaml:"notation,omitempty"`
	Priority int         `json:"priority,omitempty" yaml:"priority,omitempty"`
	Steps    []ComboStep `json:"steps" yaml:"steps"`
}

type ComboStep struct {
	MoveID          string   `json:"move,omitempty" yaml:"move,omitempty"`
	Command         string   `json:"command,omitempty" yaml:"command,omitempty"`
	Notation        string   `json:"notation,omitempty" yaml:"notation,omitempty"`
	Label           string   `json:"label,omitempty" yaml:"label,omitempty"`
	Direction       string   `json:"direction,omitempty" yaml:"direction,omitempty"`
	Buttons         []string `json:"buttons,omitempty" yaml:"buttons,omitempty"`
	GapWindowFrames int      `json:"gap_window_frames,omitempty" yaml:"gap_window_frames,omitempty"`
}

func (s *ComboStep) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.ScalarNode:
		s.Notation = strings.TrimSpace(value.Value)
		return nil
	case yaml.MappingNode:
		type comboStep ComboStep
		var decoded comboStep
		if err := value.Decode(&decoded); err != nil {
			return err
		}
		*s = ComboStep(decoded)
		s.Label = strings.TrimSpace(s.Label)
		s.Direction = strings.TrimSpace(s.Direction)
		for i := range s.Buttons {
			s.Buttons[i] = strings.TrimSpace(s.Buttons[i])
		}
		return nil
	default:
		return fmt.Errorf("combo step must be a string or mapping")
	}
}

func LoadComboFile(path string) (*ComboFile, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseComboFile(body)
}

func ParseComboFile(body []byte) (*ComboFile, error) {
	var combo ComboFile
	if err := yaml.Unmarshal(body, &combo); err != nil {
		return nil, err
	}
	if err := combo.Validate(); err != nil {
		return nil, err
	}
	return &combo, nil
}

func (c *ComboFile) Validate() error {
	if c == nil {
		return fmt.Errorf("combo file is nil")
	}
	commands, err := c.normalizeCommands()
	if err != nil {
		return err
	}
	moves, err := c.normalizeMoves(commands)
	if err != nil {
		return err
	}
	if err := c.normalizeRecipes(commands, moves); err != nil {
		return err
	}
	if err := c.normalizePractice(); err != nil {
		return err
	}
	if len(c.Sets) == 0 {
		return fmt.Errorf("combo file must contain at least one set")
	}
	seenSets := map[string]bool{}
	for i := range c.Sets {
		set := &c.Sets[i]
		set.ID = strings.TrimSpace(set.ID)
		set.Name = strings.TrimSpace(set.Name)
		set.Mode = strings.ToLower(strings.TrimSpace(set.Mode))
		if set.ID == "" {
			return fmt.Errorf("sets[%d].id is required", i)
		}
		if seenSets[set.ID] {
			return fmt.Errorf("sets[%d].id is duplicated", i)
		}
		seenSets[set.ID] = true
		if set.Name == "" {
			set.Name = set.ID
		}
		if set.Mode == "" {
			set.Mode = "command"
		}
		if set.Mode != "normal" && set.Mode != "command" {
			return fmt.Errorf("sets[%d].mode must be normal or command", i)
		}
		if len(set.Combos) == 0 {
			return fmt.Errorf("sets[%d].combos must contain at least one combo", i)
		}
		for j := range set.Combos {
			combo := &set.Combos[j]
			combo.Name = strings.TrimSpace(combo.Name)
			if combo.Name == "" {
				return fmt.Errorf("sets[%d].combos[%d].name is required", i, j)
			}
			if len(combo.Steps) == 0 {
				return fmt.Errorf("sets[%d].combos[%d].steps must contain at least one step", i, j)
			}
			for k := range combo.Steps {
				step := &combo.Steps[k]
				step.MoveID = strings.TrimSpace(step.MoveID)
				step.Command = strings.TrimSpace(step.Command)
				step.Notation = strings.TrimSpace(step.Notation)
				step.Label = strings.TrimSpace(step.Label)
				step.Direction = strings.TrimSpace(step.Direction)
				if step.GapWindowFrames < 0 {
					return fmt.Errorf("sets[%d].combos[%d].steps[%d].gap_window_frames must be >= 0", i, j, k)
				}
				if step.MoveID != "" {
					if _, ok := moves[step.MoveID]; !ok {
						return fmt.Errorf("sets[%d].combos[%d].steps[%d].move %q is not defined", i, j, k, step.MoveID)
					}
				}
				if step.Command != "" {
					if _, ok := commands[step.Command]; !ok {
						return fmt.Errorf("sets[%d].combos[%d].steps[%d].command %q is not defined", i, j, k, step.Command)
					}
				}
				if step.Notation == "" && step.Direction == "" && len(step.Buttons) == 0 && step.MoveID == "" && step.Command == "" {
					return fmt.Errorf("sets[%d].combos[%d].steps[%d] is empty", i, j, k)
				}
			}
		}
	}
	return nil
}

func (c *ComboFile) normalizeCommands() (map[string]ComboCommand, error) {
	commands := map[string]ComboCommand{}
	for i := range c.Commands {
		cmd := c.Commands[i]
		cmd.ID = strings.TrimSpace(cmd.ID)
		cmd.Name = strings.TrimSpace(cmd.Name)
		cmd.Notation = strings.TrimSpace(cmd.Notation)
		if cmd.ID == "" {
			return nil, fmt.Errorf("commands[%d].id is required", i)
		}
		if cmd.Notation == "" {
			return nil, fmt.Errorf("commands[%d].notation is required", i)
		}
		if commands[cmd.ID].ID != "" {
			return nil, fmt.Errorf("commands[%d].id is duplicated", i)
		}
		commands[cmd.ID] = cmd
		c.Commands[i] = cmd
	}
	return commands, nil
}

func (c *ComboFile) normalizeMoves(commands map[string]ComboCommand) (map[string]MoveDefinition, error) {
	moves := map[string]MoveDefinition{}
	for i := range c.Moves {
		move := c.Moves[i]
		move.ID = strings.TrimSpace(move.ID)
		move.Name = strings.TrimSpace(move.Name)
		move.Notation = strings.TrimSpace(move.Notation)
		move.Input = strings.TrimSpace(move.Input)
		move.CommandID = strings.TrimSpace(move.CommandID)
		for j := range move.Tags {
			move.Tags[j] = strings.TrimSpace(move.Tags[j])
		}
		if move.ID == "" {
			return nil, fmt.Errorf("moves[%d].id is required", i)
		}
		if move.Name == "" {
			return nil, fmt.Errorf("moves[%d].name is required", i)
		}
		if move.Input == "" && move.CommandID == "" {
			return nil, fmt.Errorf("moves[%d] must contain input or command", i)
		}
		if move.CommandID != "" {
			if _, ok := commands[move.CommandID]; !ok {
				return nil, fmt.Errorf("moves[%d].command %q is not defined", i, move.CommandID)
			}
		}
		if moves[move.ID].ID != "" {
			return nil, fmt.Errorf("moves[%d].id is duplicated", i)
		}
		moves[move.ID] = move
		c.Moves[i] = move
	}
	for i := range c.Moves {
		for j := range c.Moves[i].CancelWindows {
			window := &c.Moves[i].CancelWindows[j]
			window.Type = strings.TrimSpace(window.Type)
			if window.Start < 0 {
				return nil, fmt.Errorf("moves[%d].cancelWindows[%d].start must be >= 0", i, j)
			}
			if window.End < 0 {
				return nil, fmt.Errorf("moves[%d].cancelWindows[%d].end must be >= 0", i, j)
			}
			if window.End < window.Start {
				return nil, fmt.Errorf("moves[%d].cancelWindows[%d].end must be >= start", i, j)
			}
			for k := range window.Targets {
				window.Targets[k] = strings.TrimSpace(window.Targets[k])
				if _, ok := moves[window.Targets[k]]; !ok {
					return nil, fmt.Errorf("moves[%d].cancelWindows[%d].targets[%d] %q is not defined", i, j, k, window.Targets[k])
				}
			}
			for k := range window.TargetTags {
				window.TargetTags[k] = strings.TrimSpace(window.TargetTags[k])
			}
		}
	}
	return moves, nil
}

func (c *ComboFile) normalizeRecipes(commands map[string]ComboCommand, moves map[string]MoveDefinition) error {
	if c == nil || len(c.Sets) > 0 || len(c.Recipes) == 0 {
		return nil
	}
	setName := strings.TrimSpace(c.Character)
	if setName == "" {
		setName = strings.TrimSpace(c.Game)
	}
	if setName == "" {
		setName = "Recipes"
	} else {
		setName += " Recipes"
	}
	set := ComboSet{ID: "recipes", Name: setName, Mode: "command"}
	for i := range c.Recipes {
		recipe := c.Recipes[i]
		recipe.ID = strings.TrimSpace(recipe.ID)
		recipe.Name = strings.TrimSpace(recipe.Name)
		recipe.Notation = strings.TrimSpace(recipe.Notation)
		if recipe.ID == "" {
			return fmt.Errorf("recipes[%d].id is required", i)
		}
		if recipe.Name == "" {
			recipe.Name = recipe.ID
		}
		if len(recipe.Steps) == 0 {
			return fmt.Errorf("recipes[%d].steps must contain at least one step", i)
		}
		combo := Combo{ID: recipe.ID, Name: recipe.Name, Notation: recipe.Notation, Priority: recipe.Priority}
		for j := range recipe.Steps {
			step, err := recipeStepToComboStep(recipe.Steps[j], commands, moves, i, j)
			if err != nil {
				return err
			}
			combo.Steps = append(combo.Steps, step)
		}
		set.Combos = append(set.Combos, combo)
		c.Recipes[i] = recipe
	}
	c.Sets = []ComboSet{set}
	if c.Title == "" {
		c.Title = strings.TrimSpace(c.Game)
		if c.Title != "" && strings.TrimSpace(c.Character) != "" {
			c.Title += " - " + strings.TrimSpace(c.Character)
		}
	}
	return nil
}

func (c *ComboFile) normalizePractice() error {
	if c == nil || len(c.Recipes) == 0 {
		return nil
	}
	recipeIDs := map[string]bool{}
	for _, recipe := range c.Recipes {
		recipeIDs[strings.TrimSpace(recipe.ID)] = true
	}
	mode := PracticeMode(strings.TrimSpace(string(c.Practice.Mode)))
	if mode == "" {
		mode = PracticeModeFocus
	}
	switch mode {
	case PracticeModeFocus, PracticeModePlaylist, PracticeModeAutoDetect:
	default:
		return fmt.Errorf("practice.mode must be focus, playlist, or auto_detect")
	}
	c.Practice.Mode = mode
	c.Practice.ActiveRecipe = strings.TrimSpace(c.Practice.ActiveRecipe)
	c.Practice.ActiveSet = strings.TrimSpace(c.Practice.ActiveSet)
	if c.Practice.ActiveRecipe == "" && len(c.Recipes) > 0 {
		c.Practice.ActiveRecipe = strings.TrimSpace(c.Recipes[0].ID)
	}
	if c.Practice.ActiveRecipe != "" && !recipeIDs[c.Practice.ActiveRecipe] {
		return fmt.Errorf("practice.activeRecipe %q is not defined", c.Practice.ActiveRecipe)
	}
	seenSets := map[string]bool{}
	for i := range c.PracticeSets {
		set := &c.PracticeSets[i]
		set.ID = strings.TrimSpace(set.ID)
		set.Name = strings.TrimSpace(set.Name)
		set.Mode = strings.TrimSpace(set.Mode)
		if set.ID == "" {
			return fmt.Errorf("practiceSets[%d].id is required", i)
		}
		if seenSets[set.ID] {
			return fmt.Errorf("practiceSets[%d].id is duplicated", i)
		}
		seenSets[set.ID] = true
		if set.Name == "" {
			set.Name = set.ID
		}
		if set.Mode == "" {
			set.Mode = string(PracticeModePlaylist)
		}
		if len(set.Recipes) == 0 {
			return fmt.Errorf("practiceSets[%d].recipes must contain at least one recipe", i)
		}
		for j := range set.Recipes {
			set.Recipes[j] = strings.TrimSpace(set.Recipes[j])
			if !recipeIDs[set.Recipes[j]] {
				return fmt.Errorf("practiceSets[%d].recipes[%d] %q is not defined", i, j, set.Recipes[j])
			}
		}
	}
	if c.Practice.ActiveSet == "" && mode == PracticeModePlaylist && len(c.PracticeSets) > 0 {
		c.Practice.ActiveSet = c.PracticeSets[0].ID
	}
	if c.Practice.ActiveSet != "" && !seenSets[c.Practice.ActiveSet] {
		return fmt.Errorf("practice.activeSet %q is not defined", c.Practice.ActiveSet)
	}
	return nil
}

func (c *ComboFile) ActivePracticeState() ActivePracticeState {
	if c == nil {
		return ActivePracticeState{Mode: PracticeModeFocus}
	}
	mode := c.Practice.Mode
	if mode == "" {
		mode = PracticeModeFocus
	}
	state := ActivePracticeState{Mode: mode, ActiveRecipeID: c.Practice.ActiveRecipe, ActiveSetID: c.Practice.ActiveSet}
	if mode == PracticeModePlaylist {
		if set := c.PracticeSetByID(c.Practice.ActiveSet); set != nil && len(set.Recipes) > 0 {
			state.ActiveRecipeID = set.Recipes[0]
			state.ActiveSetIndex = 0
		}
	}
	if state.ActiveRecipeID == "" && len(c.Recipes) > 0 {
		state.ActiveRecipeID = c.Recipes[0].ID
	}
	return state
}

func (c *ComboFile) PracticeSetByID(id string) *PracticeSet {
	if c == nil {
		return nil
	}
	for i := range c.PracticeSets {
		if c.PracticeSets[i].ID == id {
			return &c.PracticeSets[i]
		}
	}
	return nil
}

func recipeStepToComboStep(step ComboRecipeStep, commands map[string]ComboCommand, moves map[string]MoveDefinition, recipeIndex, stepIndex int) (ComboStep, error) {
	step.MoveID = strings.TrimSpace(step.MoveID)
	step.Input = strings.TrimSpace(step.Input)
	step.Command = strings.TrimSpace(step.Command)
	step.Label = strings.TrimSpace(step.Label)
	if step.MoveID != "" && (step.Input != "" || step.Command != "") {
		return ComboStep{}, fmt.Errorf("recipes[%d].steps[%d] must use move or input/command, not both", recipeIndex, stepIndex)
	}
	if step.Input != "" && step.Command != "" {
		return ComboStep{}, fmt.Errorf("recipes[%d].steps[%d] must use input or command, not both", recipeIndex, stepIndex)
	}
	if step.MoveID != "" {
		move, ok := moves[step.MoveID]
		if !ok {
			return ComboStep{}, fmt.Errorf("recipes[%d].steps[%d].move %q is not defined", recipeIndex, stepIndex, step.MoveID)
		}
		label := step.Label
		if label == "" {
			label = move.Name
		}
		notation := move.Input
		command := move.CommandID
		gapWindowFrames := 0
		if notation == "" && command != "" {
			cmd, ok := commands[command]
			if !ok {
				return ComboStep{}, fmt.Errorf("recipes[%d].steps[%d].move %q command %q is not defined", recipeIndex, stepIndex, step.MoveID, command)
			}
			notation = cmd.Notation
			if gapWindowFrames == 0 {
				gapWindowFrames = cmd.MaxGapFrames
			}
		}
		if notation == "" {
			notation = move.Notation
		}
		return ComboStep{
			MoveID:          step.MoveID,
			Command:         command,
			Notation:        notation,
			Label:           label,
			GapWindowFrames: gapWindowFrames,
		}, nil
	}
	if step.Input != "" {
		return ComboStep{Notation: step.Input, Label: step.Label}, nil
	}
	if step.Command != "" {
		cmd, ok := commands[step.Command]
		if !ok {
			return ComboStep{}, fmt.Errorf("recipes[%d].steps[%d].command %q is not defined", recipeIndex, stepIndex, step.Command)
		}
		label := step.Label
		if label == "" {
			label = cmd.Name
		}
		return ComboStep{Command: step.Command, Notation: cmd.Notation, Label: label, GapWindowFrames: cmd.MaxGapFrames}, nil
	}
	return ComboStep{}, fmt.Errorf("recipes[%d].steps[%d] must contain move, input, or command", recipeIndex, stepIndex)
}

func (c *ComboFile) SetByID(id string) *ComboSet {
	if c == nil {
		return nil
	}
	for i := range c.Sets {
		if c.Sets[i].ID == id {
			return &c.Sets[i]
		}
	}
	return nil
}

func (c *ComboFile) ComboByID(id string) *Combo {
	if c == nil {
		return nil
	}
	for i := range c.Sets {
		for j := range c.Sets[i].Combos {
			if c.Sets[i].Combos[j].ID == id {
				return &c.Sets[i].Combos[j]
			}
		}
	}
	return nil
}
