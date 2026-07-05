package main

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestComboUploadListAndActivate(t *testing.T) {
	dir := t.TempDir()
	service := newComboService(dir, filepath.Join(dir, ".active-combo"), nil)
	body := []byte(`
title: Demo
moves:
  - id: demo_5lp
    name: Standing LP
    input: 5LP
sets:
  - id: normal
    name: Normal Routes
    mode: normal
    combos:
      - name: target
        steps:
          - notation: 5LP
          - 5MP
  - id: command
    name: Command Routes
    mode: command
    combos:
      - name: fireball
        steps: [236HP]
`)

	var upload bytes.Buffer
	writer := multipart.NewWriter(&upload)
	part, err := writer.CreateFormFile("combo", "demo.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(body); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/combos/upload", &upload)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()
	service.handleUpload(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("upload status = %d, body = %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/combos", nil)
	rec = httptest.NewRecorder()
	service.handleCombos(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status = %d", rec.Code)
	}
	var listed comboResponse
	if err := json.NewDecoder(rec.Body).Decode(&listed); err != nil {
		t.Fatal(err)
	}
	if listed.Current.File != "demo.yaml" || listed.Current.SetID != "normal" {
		t.Fatalf("current = %#v", listed.Current)
	}
	if listed.ActiveSet == nil || listed.ActiveSet.ID != "normal" {
		t.Fatalf("active set = %#v", listed.ActiveSet)
	}
	if len(listed.Moves) != 1 || listed.Moves[0].ID != "demo_5lp" {
		t.Fatalf("moves = %#v", listed.Moves)
	}

	activate, _ := json.Marshal(comboSelection{File: "demo.yaml", SetID: "command"})
	req = httptest.NewRequest(http.MethodPost, "/api/combos/active", bytes.NewReader(activate))
	rec = httptest.NewRecorder()
	service.handleActive(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("activate status = %d", rec.Code)
	}

	payload, err := service.response()
	if err != nil {
		t.Fatal(err)
	}
	if payload.Current.SetID != "command" || payload.ActiveSet == nil || payload.ActiveSet.ID != "command" {
		t.Fatalf("payload after activate = %#v", payload)
	}
}

func TestComboActiveAcceptsPracticeSelection(t *testing.T) {
	dir := t.TempDir()
	service := newComboService(dir, filepath.Join(dir, ".active-combo"), nil)
	body := []byte(`
version: 1
commands:
  - id: fireball
    name: Hadoken
    notation: 236P
moves:
  - id: ryu_5lp
    name: Standing LP
    input: 5LP
  - id: ryu_hadoken
    name: Hadoken
    command: fireball
recipes:
  - id: light
    name: Light Route
    steps:
      - move: ryu_5lp
  - id: fireball
    name: Fireball Route
    steps:
      - move: ryu_hadoken
practice:
  mode: playlist
  activeSet: routes
practiceSets:
  - id: routes
    name: Routes
    recipes: [light, fireball]
    loop: false
    advanceOnComplete: false
`)
	if err := os.WriteFile(filepath.Join(dir, "demo.yaml"), body, 0644); err != nil {
		t.Fatal(err)
	}
	loop := true
	advance := true
	activate, _ := json.Marshal(comboSelection{
		File:              "demo.yaml",
		SetID:             "recipes",
		Mode:              "playlist",
		ActiveSet:         "routes",
		ActiveRecipe:      "fireball",
		ActiveSetIndex:    1,
		Loop:              &loop,
		AdvanceOnComplete: &advance,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/combos/active", bytes.NewReader(activate))
	rec := httptest.NewRecorder()
	service.handleActive(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("activate status = %d, body = %s", rec.Code, rec.Body.String())
	}
	payload, err := service.response()
	if err != nil {
		t.Fatal(err)
	}
	if payload.ActivePractice.ActiveRecipeID != "fireball" || payload.ActivePractice.ActiveSetIndex != 1 {
		t.Fatalf("active practice = %#v", payload.ActivePractice)
	}
	if len(payload.PracticeSets) != 1 || !payload.PracticeSets[0].Loop || !payload.PracticeSets[0].AdvanceOnComplete {
		t.Fatalf("practice sets = %#v", payload.PracticeSets)
	}
}
