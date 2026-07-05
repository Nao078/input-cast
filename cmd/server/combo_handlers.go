package main

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"leverless-overlay/internal/config"
	"leverless-overlay/internal/ws"
)

type comboSelection struct {
	File              string              `json:"file"`
	SetID             string              `json:"set_id"`
	Mode              config.PracticeMode `json:"mode,omitempty"`
	ActiveRecipe      string              `json:"activeRecipe,omitempty"`
	ActiveSet         string              `json:"activeSet,omitempty"`
	ActiveSetIndex    int                 `json:"activeSetIndex,omitempty"`
	Loop              *bool               `json:"loop,omitempty"`
	AdvanceOnComplete *bool               `json:"advanceOnComplete,omitempty"`
}

type comboFileSummary struct {
	File  string            `json:"file"`
	Title string            `json:"title"`
	Sets  []comboSetSummary `json:"sets"`
}

type comboSetSummary struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Mode string `json:"mode"`
}

type comboResponse struct {
	Current        comboSelection             `json:"current"`
	Files          []comboFileSummary         `json:"files"`
	ActiveSet      *config.ComboSet           `json:"active_set,omitempty"`
	Commands       []config.ComboCommand      `json:"commands,omitempty"`
	Moves          []config.MoveDefinition    `json:"moves,omitempty"`
	Practice       config.PracticeConfig      `json:"practice,omitempty"`
	PracticeSets   []config.PracticeSet       `json:"practiceSets,omitempty"`
	ActivePractice config.ActivePracticeState `json:"activePractice,omitempty"`
}

type comboService struct {
	mu         sync.Mutex
	dir        string
	activePath string
	hub        *ws.Hub
}

func newComboService(dir, activePath string, hub *ws.Hub) *comboService {
	return &comboService{dir: dir, activePath: activePath, hub: hub}
}

func (s *comboService) registerHandlers() {
	http.HandleFunc("/api/combos", s.handleCombos)
	http.HandleFunc("/api/combos/upload", s.handleUpload)
	http.HandleFunc("/api/combos/active", s.handleActive)
}

func (s *comboService) handleCombos(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	payload, err := s.response()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(payload)
}

func (s *comboService) handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseMultipartForm(2 << 20); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	file, header, err := r.FormFile("combo")
	if err != nil {
		file, header, err = r.FormFile("file")
	}
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer file.Close()

	body, err := io.ReadAll(io.LimitReader(file, 2<<20))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	combo, err := config.ParseComboFile(body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	name := sanitizeComboFilename(header.Filename)
	if name == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if err := os.MkdirAll(s.dir, 0755); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	path := filepath.Join(s.dir, name)
	if err := os.WriteFile(path, body, 0644); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	current := comboSelection{File: name, SetID: combo.Sets[0].ID}
	if err := s.saveActive(current); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	s.broadcast()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"file":    name,
		"set_id":  current.SetID,
		"message": "uploaded",
	})
}

func (s *comboService) handleActive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var req comboSelection
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	req.File = filepath.Base(strings.TrimSpace(req.File))
	req.SetID = strings.TrimSpace(req.SetID)
	req.Mode = config.PracticeMode(strings.TrimSpace(string(req.Mode)))
	req.ActiveRecipe = strings.TrimSpace(req.ActiveRecipe)
	req.ActiveSet = strings.TrimSpace(req.ActiveSet)
	if req.File == "." || req.File == "" || req.SetID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	combo, err := config.LoadComboFile(filepath.Join(s.dir, req.File))
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if combo.SetByID(req.SetID) == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if err := validatePracticeSelection(combo, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.saveActive(req); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	s.broadcast()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(req)
}

func (s *comboService) response() (*comboResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.responseLocked()
}

func (s *comboService) responseLocked() (*comboResponse, error) {
	files, err := s.listFilesLocked()
	if err != nil {
		return nil, err
	}
	current := s.loadActiveLocked()
	active, comboFile := activeSetFor(s.dir, files, current)
	if active == nil && len(files) > 0 && len(files[0].Sets) > 0 {
		current = comboSelection{File: files[0].File, SetID: files[0].Sets[0].ID}
		active, comboFile = activeSetFor(s.dir, files, current)
		_ = s.saveActiveLocked(current)
	}
	response := &comboResponse{Current: current, Files: files, ActiveSet: active}
	if comboFile != nil {
		applyPracticeSelection(comboFile, current)
		response.Commands = comboFile.Commands
		response.Moves = comboFile.Moves
		response.Practice = comboFile.Practice
		response.PracticeSets = comboFile.PracticeSets
		response.ActivePractice = activePracticeStateFor(comboFile, current)
	}
	return response, nil
}

func validatePracticeSelection(combo *config.ComboFile, current *comboSelection) error {
	if combo == nil || current == nil {
		return nil
	}
	switch current.Mode {
	case "", config.PracticeModeFocus, config.PracticeModePlaylist, config.PracticeModeAutoDetect:
	default:
		return &practiceSelectionError{"practice mode must be focus, playlist, or auto_detect"}
	}
	if current.ActiveRecipe != "" && combo.ComboByID(current.ActiveRecipe) == nil {
		return &practiceSelectionError{"active recipe is not defined"}
	}
	if current.ActiveSet != "" {
		set := combo.PracticeSetByID(current.ActiveSet)
		if set == nil {
			return &practiceSelectionError{"active practice set is not defined"}
		}
		if current.ActiveSetIndex < 0 || current.ActiveSetIndex >= len(set.Recipes) {
			current.ActiveSetIndex = 0
		}
	}
	return nil
}

type practiceSelectionError struct {
	message string
}

func (e *practiceSelectionError) Error() string {
	return e.message
}

func applyPracticeSelection(combo *config.ComboFile, current comboSelection) {
	if combo == nil {
		return
	}
	if current.Mode != "" {
		combo.Practice.Mode = current.Mode
	}
	if current.ActiveRecipe != "" {
		combo.Practice.ActiveRecipe = current.ActiveRecipe
	}
	if current.ActiveSet != "" {
		combo.Practice.ActiveSet = current.ActiveSet
	}
	if current.ActiveSet != "" {
		if set := combo.PracticeSetByID(current.ActiveSet); set != nil {
			if current.Loop != nil {
				set.Loop = *current.Loop
			}
			if current.AdvanceOnComplete != nil {
				set.AdvanceOnComplete = *current.AdvanceOnComplete
			}
			if current.ActiveSetIndex >= 0 && current.ActiveSetIndex < len(set.Recipes) {
				combo.Practice.ActiveRecipe = set.Recipes[current.ActiveSetIndex]
			}
		}
	}
}

func activePracticeStateFor(combo *config.ComboFile, current comboSelection) config.ActivePracticeState {
	state := combo.ActivePracticeState()
	if current.Mode != "" {
		state.Mode = current.Mode
	}
	if current.ActiveRecipe != "" {
		state.ActiveRecipeID = current.ActiveRecipe
	}
	if current.ActiveSet != "" {
		state.ActiveSetID = current.ActiveSet
		if set := combo.PracticeSetByID(current.ActiveSet); set != nil && len(set.Recipes) > 0 {
			index := current.ActiveSetIndex
			if index < 0 || index >= len(set.Recipes) {
				index = 0
			}
			state.ActiveSetIndex = index
			state.ActiveRecipeID = set.Recipes[index]
		}
	}
	return state
}

func (s *comboService) listFilesLocked() ([]comboFileSummary, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []comboFileSummary{}, nil
		}
		return nil, err
	}
	out := []comboFileSummary{}
	for _, entry := range entries {
		if entry.IsDir() || !isComboYAML(entry.Name()) {
			continue
		}
		combo, err := config.LoadComboFile(filepath.Join(s.dir, entry.Name()))
		if err != nil {
			continue
		}
		summary := comboFileSummary{File: entry.Name(), Title: combo.Title}
		for _, set := range combo.Sets {
			summary.Sets = append(summary.Sets, comboSetSummary{
				ID:   set.ID,
				Name: set.Name,
				Mode: set.Mode,
			})
		}
		out = append(out, summary)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].File < out[j].File })
	return out, nil
}

func (s *comboService) loadActiveLocked() comboSelection {
	body, err := os.ReadFile(s.activePath)
	if err != nil {
		return comboSelection{}
	}
	var current comboSelection
	if err := json.Unmarshal(body, &current); err != nil {
		return comboSelection{}
	}
	current.File = filepath.Base(strings.TrimSpace(current.File))
	current.SetID = strings.TrimSpace(current.SetID)
	if current.File == "." {
		current.File = ""
	}
	return current
}

func (s *comboService) saveActive(current comboSelection) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveActiveLocked(current)
}

func (s *comboService) saveActiveLocked(current comboSelection) error {
	if err := os.MkdirAll(filepath.Dir(s.activePath), 0755); err != nil {
		return err
	}
	body, err := json.MarshalIndent(current, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.activePath, append(body, '\n'), 0644)
}

func (s *comboService) broadcast() {
	if s.hub == nil {
		return
	}
	payload, err := s.response()
	if err != nil {
		return
	}
	msg := map[string]interface{}{"type": "combo_config", "combo_config": payload}
	if body, err := json.Marshal(msg); err == nil {
		s.hub.Broadcast(body)
	}
}

func activeSetFor(dir string, files []comboFileSummary, current comboSelection) (*config.ComboSet, *config.ComboFile) {
	if current.File == "" || current.SetID == "" {
		return nil, nil
	}
	for _, file := range files {
		if file.File != current.File {
			continue
		}
		combo, err := config.LoadComboFile(filepath.Join(dir, current.File))
		if err != nil {
			return nil, nil
		}
		return combo.SetByID(current.SetID), combo
	}
	return nil, nil
}

func sanitizeComboFilename(name string) string {
	name = filepath.Base(strings.TrimSpace(name))
	if name == "." || name == string(filepath.Separator) || !isComboYAML(name) {
		return ""
	}
	return name
}

func isComboYAML(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return ext == ".yaml" || ext == ".yml"
}

func comboConfigMessage(payload *comboResponse) map[string]interface{} {
	if payload == nil {
		payload = &comboResponse{}
	}
	return map[string]interface{}{"type": "combo_config", "combo_config": payload}
}
