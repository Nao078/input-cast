package config

import "testing"

func TestParseComboFileAcceptsNotationAndExplicitSteps(t *testing.T) {
	body := []byte(`
title: Test Combos
sets:
  - id: basic
    name: Basic
    mode: command
    combos:
      - name: fireball punish
        steps:
          - notation: 236HP
          - direction: down
            buttons: [b3, HP]
`)
	combo, err := ParseComboFile(body)
	if err != nil {
		t.Fatalf("ParseComboFile returned error: %v", err)
	}
	if combo.Title != "Test Combos" {
		t.Fatalf("Title = %q", combo.Title)
	}
	set := combo.SetByID("basic")
	if set == nil {
		t.Fatal("SetByID(basic) returned nil")
	}
	steps := set.Combos[0].Steps
	if steps[0].Notation != "236HP" {
		t.Fatalf("steps[0].Notation = %q, want 236HP", steps[0].Notation)
	}
	if steps[1].Direction != "down" || len(steps[1].Buttons) != 2 || steps[1].Buttons[0] != "b3" {
		t.Fatalf("explicit step not decoded: %#v", steps[1])
	}
}

func TestParseComboFileDefaults(t *testing.T) {
	body := []byte(`
sets:
  - id: defaulted
    combos:
      - name: jab
        steps: [5LP]
`)
	combo, err := ParseComboFile(body)
	if err != nil {
		t.Fatalf("ParseComboFile returned error: %v", err)
	}
	set := combo.SetByID("defaulted")
	if set == nil {
		t.Fatal("SetByID(defaulted) returned nil")
	}
	if set.Name != "defaulted" {
		t.Fatalf("Name = %q, want defaulted", set.Name)
	}
	if set.Mode != "command" {
		t.Fatalf("Mode = %q, want command", set.Mode)
	}
}

func TestParseComboFileRejectsEmptySteps(t *testing.T) {
	body := []byte(`
sets:
  - id: bad
    combos:
      - name: empty
        steps:
          - {}
`)
	if _, err := ParseComboFile(body); err == nil {
		t.Fatal("ParseComboFile returned nil error for empty step")
	}
}

func TestParseComboFileRecipeStepsAllowSimpleOrderFallback(t *testing.T) {
	body := []byte(`
version: 1
recipes:
  - id: basic
    name: Basic
    steps:
      - input: 5LP
      - input: 5MP
`)
	combo, err := ParseComboFile(body)
	if err != nil {
		t.Fatalf("ParseComboFile returned error: %v", err)
	}
	if len(combo.Sets) != 1 || len(combo.Sets[0].Combos[0].Steps) != 2 {
		t.Fatalf("recipe not normalized: %#v", combo.Sets)
	}
}

func TestParseComboFileNormalizesCommandRecipeSchema(t *testing.T) {
	body := []byte(`
version: 1
game: Street Fighter 6
character: Ryu
commands:
  - id: hadoken
    name: 波動拳
    notation: 236P
    sequence:
      - dir: 2
      - dir: 3
      - dir: 6
      - button: P
        requirePress: true
    maxFrames: 20
    maxGapFrames: 8
    priority: 10
    cooldownFrames: 15
recipes:
  - id: ryu_basic_001
    name: 基本
    notation: 2MK > 236P
    steps:
      - input: 2MK
        label: しゃがみ中K
      - command: hadoken
        label: 波動拳
    priority: 10
practice:
  mode: playlist
  activeSet: ryu_beginner
practiceSets:
  - id: ryu_beginner
    name: リュウ基本練習
    mode: playlist
    recipes:
      - ryu_basic_001
    loop: true
    advanceOnComplete: true
`)
	combo, err := ParseComboFile(body)
	if err != nil {
		t.Fatalf("ParseComboFile returned error: %v", err)
	}
	if len(combo.Sets) != 1 {
		t.Fatalf("len(Sets) = %d, want 1", len(combo.Sets))
	}
	set := combo.SetByID("recipes")
	if set == nil {
		t.Fatal("SetByID(recipes) returned nil")
	}
	recipe := set.Combos[0]
	if recipe.ID != "ryu_basic_001" || recipe.Notation != "2MK > 236P" || recipe.Priority != 10 {
		t.Fatalf("recipe not normalized: %#v", recipe)
	}
	if recipe.Steps[0].Notation != "2MK" || recipe.Steps[0].Label != "しゃがみ中K" {
		t.Fatalf("input step not normalized: %#v", recipe.Steps[0])
	}
	if recipe.Steps[1].Notation != "236P" || recipe.Steps[1].Label != "波動拳" {
		t.Fatalf("command step not normalized: %#v", recipe.Steps[1])
	}
	if recipe.Steps[1].GapWindowFrames != 8 {
		t.Fatalf("command GapWindowFrames = %d, want 8", recipe.Steps[1].GapWindowFrames)
	}
	if combo.Commands[0].Sequence[0].Dir != "2" {
		t.Fatalf("numeric dir not normalized: %#v", combo.Commands[0].Sequence[0])
	}
	if combo.Practice.Mode != PracticeModePlaylist || combo.Practice.ActiveSet != "ryu_beginner" {
		t.Fatalf("practice not normalized: %#v", combo.Practice)
	}
	if len(combo.PracticeSets) != 1 || combo.PracticeSets[0].Recipes[0] != "ryu_basic_001" || !combo.PracticeSets[0].Loop || !combo.PracticeSets[0].AdvanceOnComplete {
		t.Fatalf("practiceSets not normalized: %#v", combo.PracticeSets)
	}
	state := combo.ActivePracticeState()
	if state.Mode != PracticeModePlaylist || state.ActiveSetID != "ryu_beginner" || state.ActiveRecipeID != "ryu_basic_001" || state.ActiveSetIndex != 0 {
		t.Fatalf("ActivePracticeState = %#v", state)
	}
}

func TestParseComboFileNormalizesMoveRecipeSchema(t *testing.T) {
	body := []byte(`
version: 1
commands:
  - id: hadoken
    name: 波動拳
    notation: 236P
    maxGapFrames: 8
moves:
  - id: ryu_5lp
    name: 立ち弱P
    notation: 5LP
    input: 5LP
    tags: [normal, light]
    cancelWindows:
      - type: chain
        start: 8
        end: 18
        targetTags: [light]
  - id: ryu_2lk
    name: しゃがみ弱K
    notation: 2LK
    input: 2LK
    tags: [normal, light]
    cancelWindows:
      - type: special
        start: 10
        end: 22
        targets: [ryu_hadoken]
  - id: ryu_hadoken
    name: 波動拳
    notation: 236P
    command: hadoken
    tags: [special]
recipes:
  - id: ryu_basic_001
    name: 基本
    notation: LP > 2LK > 236P
    steps:
      - move: ryu_5lp
      - move: ryu_2lk
      - move: ryu_hadoken
`)
	combo, err := ParseComboFile(body)
	if err != nil {
		t.Fatalf("ParseComboFile returned error: %v", err)
	}
	if len(combo.Moves) != 3 {
		t.Fatalf("len(Moves) = %d, want 3", len(combo.Moves))
	}
	recipe := combo.SetByID("recipes").Combos[0]
	if recipe.Steps[0].MoveID != "ryu_5lp" || recipe.Steps[0].Notation != "5LP" || recipe.Steps[0].Label != "立ち弱P" {
		t.Fatalf("move.input step not normalized: %#v", recipe.Steps[0])
	}
	if recipe.Steps[2].MoveID != "ryu_hadoken" || recipe.Steps[2].Command != "hadoken" || recipe.Steps[2].Notation != "236P" {
		t.Fatalf("move.command step not normalized: %#v", recipe.Steps[2])
	}
	if recipe.Steps[2].GapWindowFrames != 8 {
		t.Fatalf("move.command GapWindowFrames = %d, want 8", recipe.Steps[2].GapWindowFrames)
	}
}

func TestParseComboFileRejectsUndefinedRecipeMove(t *testing.T) {
	body := []byte(`
version: 1
recipes:
  - id: one
    name: One
    steps:
      - move: missing
`)
	if _, err := ParseComboFile(body); err == nil {
		t.Fatal("ParseComboFile returned nil error for undefined move")
	}
}

func TestParseComboFileRejectsUndefinedMoveCommand(t *testing.T) {
	body := []byte(`
version: 1
moves:
  - id: bad
    name: Bad
    command: missing
recipes:
  - id: one
    name: One
    steps:
      - move: bad
`)
	if _, err := ParseComboFile(body); err == nil {
		t.Fatal("ParseComboFile returned nil error for undefined move command")
	}
}

func TestParseComboFileRejectsInvalidMoveCancelWindow(t *testing.T) {
	body := []byte(`
version: 1
moves:
  - id: one
    name: One
    input: 5LP
    cancelWindows:
      - start: 20
        end: 8
recipes:
  - id: one
    name: One
    steps:
      - move: one
`)
	if _, err := ParseComboFile(body); err == nil {
		t.Fatal("ParseComboFile returned nil error for invalid cancel window")
	}
}

func TestParseComboFileRejectsUndefinedCancelWindowTarget(t *testing.T) {
	body := []byte(`
version: 1
moves:
  - id: one
    name: One
    input: 5LP
    cancelWindows:
      - start: 1
        end: 2
        targets: [missing]
recipes:
  - id: one
    name: One
    steps:
      - move: one
`)
	if _, err := ParseComboFile(body); err == nil {
		t.Fatal("ParseComboFile returned nil error for undefined cancel target")
	}
}

func TestParseComboFileRejectsMoveMixedWithInputOrCommand(t *testing.T) {
	body := []byte(`
version: 1
moves:
  - id: one
    name: One
    input: 5LP
recipes:
  - id: one
    name: One
    steps:
      - move: one
        input: 5LP
`)
	if _, err := ParseComboFile(body); err == nil {
		t.Fatal("ParseComboFile returned nil error for move mixed with input")
	}
}

func TestParseComboFileRejectsUndefinedPracticeRecipe(t *testing.T) {
	body := []byte(`
version: 1
recipes:
  - id: one
    name: One
    steps:
      - input: 5LP
practice:
  mode: focus
  activeRecipe: missing
`)
	if _, err := ParseComboFile(body); err == nil {
		t.Fatal("ParseComboFile returned nil error for undefined activeRecipe")
	}
}

func TestParseComboFileAllowsAutoDetectPracticeMode(t *testing.T) {
	body := []byte(`
version: 1
recipes:
  - id: one
    name: One
    steps:
      - input: 5LP
practice:
  mode: auto_detect
`)
	combo, err := ParseComboFile(body)
	if err != nil {
		t.Fatalf("ParseComboFile returned error: %v", err)
	}
	if combo.Practice.Mode != PracticeModeAutoDetect {
		t.Fatalf("Practice.Mode = %q, want auto_detect", combo.Practice.Mode)
	}
}
