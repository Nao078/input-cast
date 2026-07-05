package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"image/color"
	"io"
	"math"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"leverless-overlay/internal/bridge"
	"leverless-overlay/internal/gamepad"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const (
	appID               = "net.input-cast.bridge"
	prefServerURL       = "server_url"
	prefCustomServer    = "custom_server"
	prefServerHost      = "server_host"
	prefAutoStart       = "auto_start"
	prefStartMinimized  = "start_minimized"
	prefCloseToTray     = "close_to_tray"
	prefScanIntervalMS  = "scan_interval_ms"
	defaultServerURL    = "http://localhost:8080/api/input/gamepad"
	defaultServerHost   = "localhost:8080"
	serverInputEndpoint = "/api/input/gamepad"
	defaultScanInterval = 2000
	healthInterval      = time.Second
	maxLogLines         = 60
	minScanInterval     = 100
	newProfileOption    = "New Profile..."
)

type bridgeUI struct {
	app       fyne.App
	window    fyne.Window
	client    *bridge.Client
	cancel    context.CancelFunc
	connected atomic.Bool
	running   bool
	logLines  []string

	connectionIcon   *widget.Icon
	connectionStatus *widget.Label
	facingStatus     *widget.Label
	facingRight      bool
	facingToggleDown bool
	customServer     *widget.Check
	serverHost       *widget.Entry
	configProfile    *widget.Select
	currentProfile   string
	comboFile        *widget.Select
	comboSet         *widget.Select
	comboMode        *widget.Select
	comboRecipe      *widget.Select
	comboPracticeSet *widget.Select
	comboLoop        *widget.Check
	comboAdvance     *widget.Check
	comboVolume      *widget.Slider
	comboVolumeLabel *widget.Label
	comboStatus      *widget.Label
	comboState       *bridge.ComboResponse
	comboLoading     bool
	scanInterval     *widget.Entry
	autoStart        *widget.Check
	startMinimized   *widget.Check
	closeToTray      *widget.Check
	status           *widget.Label
	device           *widget.Label
	lastSend         *widget.Label
	log              *widget.Entry
	startButton      *widget.Button
	stopButton       *widget.Button
	editButton       *widget.Button
	previewCard      fyne.CanvasObject
	devicePanel      fyne.CanvasObject
	logPanel         fyne.CanvasObject
	rightContent     *fyne.Container
	previewBody      *fyne.Container
	previewHolder    *fyne.Container
	editJSON         *widget.Entry
	currentPreview   *bridge.OverlayConfig
	editConfig       *bridge.OverlayConfig
	matrix           map[string]*canvas.Circle
	matrixInactive   map[string]color.NRGBA
	matrixActive     map[string]color.NRGBA
	matrixLabels     map[string]*canvas.Text
}

// 固定サイズを維持するためのカスタムレイアウト構造体
type fixedSizeLayout struct {
	size fyne.Size
}

type draggableRect struct {
	*canvas.Rectangle
	onDrag      func(dx, dy float32)
	onEnd       func()
	onSecondary func()
}

func newDraggableRect(fill color.NRGBA, onDrag func(dx, dy float32), onEnd func(), onSecondary func()) *draggableRect {
	rect := canvas.NewRectangle(fill)
	rect.StrokeColor = color.NRGBA{R: 220, G: 190, B: 80, A: 180}
	rect.StrokeWidth = 1
	return &draggableRect{Rectangle: rect, onDrag: onDrag, onEnd: onEnd, onSecondary: onSecondary}
}

func (d *draggableRect) Dragged(event *fyne.DragEvent) {
	if d.onDrag != nil {
		d.onDrag(event.Dragged.DX, event.Dragged.DY)
	}
}

func (d *draggableRect) DragEnd() {
	if d.onEnd != nil {
		d.onEnd()
	}
}

func (d *draggableRect) TappedSecondary(*fyne.PointEvent) {
	if d.onSecondary != nil {
		d.onSecondary()
	}
}

func newFixedSizeLayout(size fyne.Size) fyne.Layout {
	return &fixedSizeLayout{size: size}
}

func (f *fixedSizeLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	for _, obj := range objects {
		obj.Resize(f.size)
		obj.Move(fyne.NewPos(0, 0))
	}
}

func (f *fixedSizeLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	return f.size
}

func main() {
	a := app.NewWithID(appID)
	a.SetIcon(appIconResource)
	a.Settings().SetTheme(theme.DarkTheme())
	w := a.NewWindow("Input Cast Client")
	w.SetIcon(appIconResource)
	w.Resize(fyne.NewSize(920, 600))

	ui := newBridgeUI(a, w)
	w.SetContent(ui.content())
	trayAvailable := setupTray(a, w, ui)
	w.SetCloseIntercept(func() {
		if trayAvailable && ui.closeToTray.Checked {
			w.Hide()
			return
		}
		ui.stop()
		w.Close()
	})

	if ui.autoStart.Checked {
		ui.start()
	}

	w.Show()
	if trayAvailable && ui.autoStart.Checked && ui.startMinimized.Checked {
		w.Hide()
	}
	a.Run()
}

func newBridgeUI(a fyne.App, w fyne.Window) *bridgeUI {
	prefs := a.Preferences()
	customServer := prefs.BoolWithFallback(prefCustomServer, false)
	serverHost := prefs.StringWithFallback(prefServerHost, "")
	if serverHost == "" {
		serverHost = serverHostFromURL(prefs.StringWithFallback(prefServerURL, defaultServerURL))
	}
	if serverHost == "" {
		serverHost = defaultServerHost
	}

	ui := &bridgeUI{
		app:            a,
		window:         w,
		client:         bridge.NewClient(serverURLForSettings(customServer, serverHost)),
		matrix:         make(map[string]*canvas.Circle),
		matrixInactive: make(map[string]color.NRGBA),
		matrixActive:   make(map[string]color.NRGBA),
		matrixLabels:   make(map[string]*canvas.Text),
		facingRight:    true,
	}

	ui.customServer = widget.NewCheck("Use another server", func(value bool) {
		if ui.serverHost == nil {
			return
		}
		prefs.SetBool(prefCustomServer, value)
		ui.serverHost.SetText(normalizeServerHost(ui.serverHost.Text))
		ui.serverHost.SetPlaceHolder(serverHostPlaceholder(value))
		ui.setServerHostEnabled(value)
		ui.applyServerSettings()
	})
	ui.customServer.SetChecked(customServer)
	ui.serverHost = widget.NewEntry()
	ui.serverHost.SetPlaceHolder(serverHostPlaceholder(customServer))
	ui.serverHost.SetText(serverHost)
	ui.setServerHostEnabled(customServer)
	ui.serverHost.OnChanged = func(value string) {
		prefs.SetString(prefServerHost, normalizeServerHost(value))
		ui.applyServerSettings()
	}

	ui.configProfile = widget.NewSelect(nil, func(value string) {
		if value == newProfileOption {
			ui.promptNewProfile()
		}
	})
	ui.comboFile = widget.NewSelect(nil, func(value string) {
		ui.applyComboFileSelection(value)
	})
	ui.comboSet = widget.NewSelect(nil, func(value string) {
		ui.activateSelectedCombo()
	})
	ui.comboMode = widget.NewSelect([]string{
		string(bridge.PracticeModeFocus),
		string(bridge.PracticeModePlaylist),
		string(bridge.PracticeModeAutoDetect),
	}, func(value string) {
		ui.applyPracticeModeSelection(value)
	})
	ui.comboRecipe = widget.NewSelect(nil, func(value string) {
		ui.activateSelectedCombo()
	})
	ui.comboPracticeSet = widget.NewSelect(nil, func(value string) {
		ui.applyPracticeSetSelection(value)
		ui.activateSelectedCombo()
	})
	ui.comboLoop = widget.NewCheck("Loop", func(value bool) {
		ui.activateSelectedCombo()
	})
	ui.comboAdvance = widget.NewCheck("Advance on complete", func(value bool) {
		ui.activateSelectedCombo()
	})
	ui.comboVolumeLabel = widget.NewLabel("Volume: 70%")
	ui.comboVolume = widget.NewSlider(0, 100)
	ui.comboVolume.Step = 1
	ui.comboVolume.Value = 70
	ui.comboVolume.OnChanged = func(value float64) {
		ui.comboVolumeLabel.SetText(fmt.Sprintf("Volume: %.0f%%", value))
	}
	ui.comboStatus = widget.NewLabel("No combo YAML loaded")

	ui.scanInterval = widget.NewEntry()
	ui.scanInterval.SetText(strconv.Itoa(prefs.IntWithFallback(prefScanIntervalMS, defaultScanInterval)))
	ui.scanInterval.OnChanged = func(value string) {
		if ms, err := strconv.Atoi(strings.TrimSpace(value)); err == nil {
			prefs.SetInt(prefScanIntervalMS, ms)
		}
	}

	ui.autoStart = widget.NewCheck("Auto start", func(value bool) {
		prefs.SetBool(prefAutoStart, value)
	})
	ui.autoStart.SetChecked(prefs.BoolWithFallback(prefAutoStart, false))

	ui.startMinimized = widget.NewCheck("Start minimized", func(value bool) {
		prefs.SetBool(prefStartMinimized, value)
	})
	ui.startMinimized.SetChecked(prefs.BoolWithFallback(prefStartMinimized, false))

	ui.closeToTray = widget.NewCheck("Close to tray", func(value bool) {
		prefs.SetBool(prefCloseToTray, value)
	})
	ui.closeToTray.SetChecked(prefs.BoolWithFallback(prefCloseToTray, true))

	ui.status = widget.NewLabel("Stopped")
	ui.connectionIcon = widget.NewIcon(antennaResource(theme.ColorNameDisabled))
	ui.connectionStatus = widget.NewLabel("Not started")
	ui.facingStatus = widget.NewLabel(facingStatusText(true))
	ui.device = widget.NewLabel("-")
	ui.lastSend = widget.NewLabel("-")
	ui.log = widget.NewMultiLineEntry()
	ui.log.SetMinRowsVisible(6)
	ui.log.Disable()

	ui.startButton = widget.NewButtonWithIcon("Start", theme.MediaPlayIcon(), ui.start)
	ui.stopButton = widget.NewButtonWithIcon("Stop", theme.MediaStopIcon(), ui.stop)
	ui.stopButton.Disable()
	ui.editButton = widget.NewButtonWithIcon("", toolsIconResource, ui.enterEditMode)
	ui.editJSON = widget.NewMultiLineEntry()
	ui.editJSON.SetMinRowsVisible(12)

	return ui
}

func (ui *bridgeUI) content() fyne.CanvasObject {
	serverURLField := container.NewVBox(
		ui.customServer,
		ui.serverHost,
	)
	profileLoad := widget.NewButtonWithIcon("Load", theme.FolderOpenIcon(), ui.loadSelectedProfile)
	profileSave := widget.NewButtonWithIcon("Save", theme.DocumentSaveIcon(), func() {
		ui.saveCurrentPreviewAs(ui.selectedProfile())
	})
	profileField := container.NewVBox(
		ui.configProfile,
		container.NewGridWithColumns(2, profileLoad, profileSave),
	)
	comboUpload := widget.NewButtonWithIcon("Upload YAML", theme.UploadIcon(), ui.uploadComboYAML)
	comboReload := widget.NewButtonWithIcon("Reload", theme.ViewRefreshIcon(), ui.reloadCombos)
	comboVolumeSave := widget.NewButtonWithIcon("Save Volume", theme.DocumentSaveIcon(), ui.saveComboVolume)
	comboField := container.NewVBox(
		ui.comboFile,
		ui.comboSet,
		ui.comboMode,
		ui.comboRecipe,
		ui.comboPracticeSet,
		container.NewVBox(ui.comboLoop, ui.comboAdvance),
		ui.comboVolumeLabel,
		ui.comboVolume,
		comboVolumeSave,
		container.NewGridWithColumns(2, comboUpload, comboReload),
		ui.comboStatus,
	)
	scanIntervalField := container.NewVBox(
		widget.NewLabelWithStyle("Scan interval ms", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		ui.scanInterval,
		ui.autoStart,
		ui.startMinimized,
		ui.closeToTray,
	)
	safetyNote := widget.NewLabel("Safety: input-cast only visualizes device input. It does not read game memory, hook rendering APIs, use macros, or automate input.")
	safetyNote.Wrapping = fyne.TextWrapWord

	serverItem := widget.NewAccordionItem("Server", serverURLField)
	serverItem.Open = true
	comboItem := widget.NewAccordionItem("Combo", comboField)
	comboItem.Open = true
	configuration := widget.NewAccordion(
		serverItem,
		widget.NewAccordionItem("Profile", profileField),
		comboItem,
		widget.NewAccordionItem("Runtime", scanIntervalField),
		widget.NewAccordionItem("Safety", safetyNote),
	)
	configuration.MultiOpen = true

	actions := container.NewGridWithColumns(2, ui.startButton, ui.stopButton)
	configBody := container.NewBorder(nil, actions, nil, nil, container.NewVScroll(configuration))
	configPanel := widget.NewCard("Configuration", "", configBody)

	status := container.NewGridWithColumns(2,
		widget.NewLabel("App"), ui.status,
		widget.NewLabel("Device"), ui.device,
		widget.NewLabel("Last send"), ui.lastSend,
	)
	ui.devicePanel = widget.NewCard("Device Status", "", status)

	statusBar := container.NewHBox(
		ui.connectionIcon,
		ui.connectionStatus,
		layout.NewSpacer(),
		ui.facingStatus,
	)

	previewTitle := widget.NewLabelWithStyle("Input Preview", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	previewSubtitle := widget.NewLabel("Gamepad Status Matrix")
	previewHeader := container.NewBorder(nil, nil, nil, ui.editButton, container.NewVBox(previewTitle, previewSubtitle))
	ui.previewBody = container.NewMax(ui.matrixPanel())
	ui.previewCard = widget.NewCard("", "", container.NewVBox(previewHeader, ui.previewBody))
	ui.logPanel = widget.NewCard("Log", "Activity Log", ui.log)

	ui.rightContent = container.NewMax(container.NewVBox(ui.previewCard, ui.devicePanel, ui.logPanel))

	split := container.NewHSplit(configPanel, ui.rightContent)
	split.Offset = 0.28

	return container.NewBorder(nil, statusBar, nil, nil, split)
}

func (ui *bridgeUI) matrixPanel() fyne.CanvasObject {
	ui.previewHolder = container.New(newFixedSizeLayout(fyne.NewSize(580, 230)))
	cfg := ui.currentPreview
	if cfg == nil {
		cfg = defaultPreviewConfig()
	}
	ui.renderPreview(cfg)
	return container.NewCenter(ui.previewHolder)
}

func (ui *bridgeUI) renderPreview(cfg *bridge.OverlayConfig) {
	if cfg == nil {
		cfg = defaultPreviewConfig()
	}
	ui.currentPreview = cloneOverlayConfig(cfg)
	panelWidth := float32(580)
	panelHeight := float32(230)
	ui.matrix = make(map[string]*canvas.Circle)
	ui.matrixInactive = make(map[string]color.NRGBA)
	ui.matrixActive = make(map[string]color.NRGBA)
	ui.matrixLabels = make(map[string]*canvas.Text)

	title := canvas.NewText("Input Matrix", color.NRGBA{R: 225, G: 232, B: 238, A: 255})
	title.Alignment = fyne.TextAlignCenter
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.TextSize = 14
	title.Move(fyne.NewPos(0, panelHeight-26))
	title.Resize(fyne.NewSize(panelWidth, 20))

	panel := container.NewWithoutLayout(title)
	scale := previewScale(cfg, panelWidth, panelHeight)

	controller := canvas.NewRectangle(parseColor(cfg.Controller.Color, color.NRGBA{R: 68, G: 88, B: 102, A: 210}))
	controller.StrokeColor = color.NRGBA{R: 105, G: 128, B: 146, A: 230}
	controller.StrokeWidth = 2
	controller.CornerRadius = 8
	controller.Move(fyne.NewPos(scale.x(cfg.Controller.X), scale.y(cfg.Controller.Y)))
	controller.Resize(fyne.NewSize(scale.size(cfg.Controller.Width), scale.size(cfg.Controller.Height)))
	panel.Add(controller)

	add := func(button bridge.ButtonConfig) {
		if !button.IsVisible() {
			return
		}
		size := scale.size(button.Size)
		x := scale.x(button.X)
		y := scale.y(button.Y)
		label := button.Label
		if label == "" {
			label = button.ID
		}
		normalColor := parseColor(button.Color, inactiveColor())
		pressedColor := parseColor(button.PressedColor, matrixActiveColor(button.ID))
		c := canvas.NewCircle(normalColor)
		c.StrokeColor = color.NRGBA{R: 128, G: 146, B: 158, A: 210}
		c.StrokeWidth = 2
		c.Move(fyne.NewPos(x, y))
		c.Resize(fyne.NewSize(size, size))

		t := canvas.NewText(label, color.NRGBA{R: 220, G: 228, B: 235, A: 255})
		t.Alignment = fyne.TextAlignCenter
		t.TextStyle = fyne.TextStyle{Bold: true}
		t.TextSize = 12
		t.Move(fyne.NewPos(x, y+(size-16)/2))
		t.Resize(fyne.NewSize(size, 16))

		panel.Add(c)
		panel.Add(t)
		ui.matrix[button.ID] = c
		ui.matrixInactive[button.ID] = normalColor
		ui.matrixActive[button.ID] = pressedColor
		ui.matrixLabels[button.ID] = t
	}

	for _, button := range cfg.Buttons {
		add(button)
	}

	if ui.previewHolder != nil {
		ui.previewHolder.Objects = []fyne.CanvasObject{panel}
		ui.previewHolder.Refresh()
	}
}

func (ui *bridgeUI) enterEditMode() {
	cfg := ui.currentPreview
	if cfg == nil {
		cfg = defaultPreviewConfig()
	}
	ui.editConfig = cloneOverlayConfig(cfg)
	ui.syncEditJSON()
	ui.setEditModeLayout(true)
	ui.refreshEditPreview()
	ui.editButton.Disable()
	ui.editButton.Hide()
}

func (ui *bridgeUI) reloadEditConfig() {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	cfg, err := ui.client.FetchConfig(ctx)
	if err != nil {
		ui.appendLog("config load failed: " + err.Error())
		return
	}
	ui.editConfig = cloneOverlayConfig(cfg)
	ui.syncEditJSON()
	ui.refreshEditPreview()
}

func (ui *bridgeUI) loadProfilesAndCurrentConfig() {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	profiles, err := ui.client.FetchProfiles(ctx)
	if err != nil {
		ui.appendLog("profile load failed: " + err.Error())
	}
	cfg, err := ui.client.FetchConfig(ctx)
	if err != nil {
		ui.appendLog("config load failed: " + err.Error())
	}
	combos, comboErr := ui.client.FetchCombos(ctx)
	fyne.Do(func() {
		if profiles != nil {
			ui.applyProfiles(profiles)
		}
		if cfg != nil {
			ui.renderPreview(cfg)
			ui.applyComboVolume(cfg)
		}
		if comboErr == nil {
			ui.applyCombos(combos)
		} else {
			ui.comboStatus.SetText("Combo load failed")
			ui.appendLog("combo load failed: " + comboErr.Error())
		}
	})
}

func (ui *bridgeUI) loadSelectedProfile() {
	name := ui.selectedProfile()
	if name == "" {
		ui.appendLog("profile load failed: profile is empty")
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	current, err := ui.client.SwitchProfile(ctx, name)
	if err != nil {
		ui.appendLog("profile load failed: " + err.Error())
		return
	}
	cfg, err := ui.client.FetchConfig(ctx)
	if err != nil {
		ui.appendLog("config load failed: " + err.Error())
		return
	}
	profiles, _ := ui.client.FetchProfiles(ctx)
	fyne.Do(func() {
		if profiles != nil {
			ui.applyProfiles(profiles)
		}
		if current != "" {
			ui.currentProfile = current
			ui.configProfile.SetSelected(current)
		}
		ui.renderPreview(cfg)
		ui.applyComboVolume(cfg)
		ui.appendLog("profile loaded: " + current)
	})
}

func (ui *bridgeUI) saveCurrentPreviewAs(profile string) {
	cfg := ui.currentPreview
	if ui.editConfig != nil {
		cfg = ui.editConfig
	}
	if cfg == nil {
		cfg = defaultPreviewConfig()
	}
	ui.saveOverlayConfig(cloneOverlayConfig(cfg), profile)
}

func (ui *bridgeUI) applyComboVolume(cfg *bridge.OverlayConfig) {
	if cfg == nil || ui.comboVolume == nil || ui.comboVolumeLabel == nil {
		return
	}
	volume := cfg.ComboAudio.Volume
	if volume <= 0 {
		volume = 0.7
	}
	volume = math.Max(0, math.Min(1, volume))
	ui.comboVolume.Value = volume * 100
	ui.comboVolumeLabel.SetText(fmt.Sprintf("Volume: %.0f%%", ui.comboVolume.Value))
	ui.comboVolume.Refresh()
}

func (ui *bridgeUI) saveComboVolume() {
	cfg := ui.currentPreview
	if ui.editConfig != nil {
		cfg = ui.editConfig
	}
	if cfg == nil {
		cfg = defaultPreviewConfig()
	}
	next := cloneOverlayConfig(cfg)
	next.ComboAudio.Volume = math.Max(0, math.Min(100, ui.comboVolume.Value)) / 100
	ui.saveOverlayConfig(next, ui.selectedProfile())
}

func (ui *bridgeUI) reloadCombos() {
	go ui.loadCombos()
}

func (ui *bridgeUI) loadCombos() {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	combos, err := ui.client.FetchCombos(ctx)
	if err != nil {
		ui.appendLog("combo load failed: " + err.Error())
		fyne.Do(func() {
			ui.comboStatus.SetText("Combo load failed")
		})
		return
	}
	fyne.Do(func() {
		ui.applyCombos(combos)
	})
}

func (ui *bridgeUI) applyCombos(combos *bridge.ComboResponse) {
	ui.comboLoading = true
	defer func() { ui.comboLoading = false }()
	ui.comboState = combos
	if combos == nil || len(combos.Files) == 0 {
		ui.comboFile.Options = nil
		ui.comboFile.ClearSelected()
		ui.comboFile.Refresh()
		ui.comboSet.Options = nil
		ui.comboSet.ClearSelected()
		ui.comboSet.Refresh()
		ui.comboRecipe.Options = nil
		ui.comboRecipe.ClearSelected()
		ui.comboRecipe.Refresh()
		ui.comboPracticeSet.Options = nil
		ui.comboPracticeSet.ClearSelected()
		ui.comboPracticeSet.Refresh()
		ui.comboStatus.SetText("No combo YAML loaded")
		return
	}
	files := make([]string, 0, len(combos.Files))
	for _, file := range combos.Files {
		files = append(files, file.File)
	}
	ui.comboFile.Options = files
	ui.comboFile.Refresh()
	selectedFile := combos.Current.File
	if selectedFile == "" || !stringInSlice(files, selectedFile) {
		selectedFile = files[0]
	}
	ui.comboFile.SetSelected(selectedFile)
	ui.applyComboFileSelection(selectedFile)
	if combos.Current.SetID != "" {
		ui.comboSet.SetSelected(combos.Current.SetID)
	}
	ui.applyPracticeSelections()
	ui.updateComboStatus()
}

func (ui *bridgeUI) applyComboFileSelection(file string) {
	if ui.comboState == nil {
		return
	}
	selected := comboFileByName(ui.comboState, file)
	ui.comboLoading = true
	defer func() { ui.comboLoading = false }()
	if selected == nil {
		ui.comboSet.Options = nil
		ui.comboSet.ClearSelected()
		ui.comboSet.Refresh()
		ui.updateComboStatus()
		return
	}
	options := make([]string, 0, len(selected.Sets))
	for _, set := range selected.Sets {
		options = append(options, set.ID)
	}
	ui.comboSet.Options = options
	ui.comboSet.Refresh()
	if ui.comboState.Current.File == file && stringInSlice(options, ui.comboState.Current.SetID) {
		ui.comboSet.SetSelected(ui.comboState.Current.SetID)
	} else if len(options) > 0 {
		ui.comboSet.SetSelected(options[0])
	}
	ui.updateComboStatus()
}

func (ui *bridgeUI) applyPracticeSelections() {
	if ui.comboState == nil {
		return
	}
	ui.comboLoading = true
	defer func() { ui.comboLoading = false }()

	mode := string(ui.comboState.ActivePractice.Mode)
	if mode == "" {
		mode = string(bridge.PracticeModeFocus)
	}
	ui.comboMode.SetSelected(mode)

	recipes := comboRecipeOptions(ui.comboState)
	ui.comboRecipe.Options = recipes
	ui.comboRecipe.Refresh()
	if stringInSlice(recipes, ui.comboState.ActivePractice.ActiveRecipeID) {
		ui.comboRecipe.SetSelected(ui.comboState.ActivePractice.ActiveRecipeID)
	} else if len(recipes) > 0 {
		ui.comboRecipe.SetSelected(recipes[0])
	} else {
		ui.comboRecipe.ClearSelected()
	}

	sets := comboPracticeSetOptions(ui.comboState)
	ui.comboPracticeSet.Options = sets
	ui.comboPracticeSet.Refresh()
	if stringInSlice(sets, ui.comboState.ActivePractice.ActiveSetID) {
		ui.comboPracticeSet.SetSelected(ui.comboState.ActivePractice.ActiveSetID)
	} else if len(sets) > 0 {
		ui.comboPracticeSet.SetSelected(sets[0])
	} else {
		ui.comboPracticeSet.ClearSelected()
	}
	ui.applyPracticeSetSelection(ui.comboPracticeSet.Selected)
}

func (ui *bridgeUI) applyPracticeModeSelection(value string) {
	if ui.comboLoading {
		return
	}
	if value == string(bridge.PracticeModePlaylist) {
		if ui.comboPracticeSet.Selected == "" && len(ui.comboPracticeSet.Options) > 0 {
			ui.comboPracticeSet.SetSelected(ui.comboPracticeSet.Options[0])
		}
	} else if value == string(bridge.PracticeModeFocus) {
		if ui.comboRecipe.Selected == "" && len(ui.comboRecipe.Options) > 0 {
			ui.comboRecipe.SetSelected(ui.comboRecipe.Options[0])
		}
	}
	ui.activateSelectedCombo()
}

func (ui *bridgeUI) applyPracticeSetSelection(id string) {
	if ui.comboState == nil {
		return
	}
	set := practiceSetByID(ui.comboState, id)
	ui.comboLoading = true
	defer func() { ui.comboLoading = false }()
	if set == nil {
		ui.comboLoop.SetChecked(false)
		ui.comboAdvance.SetChecked(false)
		return
	}
	ui.comboLoop.SetChecked(set.Loop)
	ui.comboAdvance.SetChecked(set.AdvanceOnComplete)
	if len(set.Recipes) > 0 && !stringInSlice(set.Recipes, ui.comboRecipe.Selected) {
		ui.comboRecipe.SetSelected(set.Recipes[0])
	}
}

func (ui *bridgeUI) activateSelectedCombo() {
	if ui.comboLoading {
		return
	}
	file := strings.TrimSpace(ui.comboFile.Selected)
	setID := strings.TrimSpace(ui.comboSet.Selected)
	if file == "" || setID == "" {
		return
	}
	selection := ui.selectedComboSelection(file, setID)
	if ui.comboState != nil && comboSelectionMatches(ui.comboState.Current, selection) {
		ui.updateComboStatus()
		return
	}
	ui.comboLoading = true
	ui.comboStatus.SetText("Changing combo...")
	go ui.activateCombo(selection)
}

func (ui *bridgeUI) selectedComboSelection(file, setID string) bridge.ComboSelection {
	selection := bridge.ComboSelection{
		File:           file,
		SetID:          setID,
		Mode:           bridge.PracticeMode(strings.TrimSpace(ui.comboMode.Selected)),
		ActiveRecipe:   strings.TrimSpace(ui.comboRecipe.Selected),
		ActiveSet:      strings.TrimSpace(ui.comboPracticeSet.Selected),
		ActiveSetIndex: 0,
	}
	loop := ui.comboLoop.Checked
	advance := ui.comboAdvance.Checked
	selection.Loop = &loop
	selection.AdvanceOnComplete = &advance
	if set := practiceSetByID(ui.comboState, selection.ActiveSet); set != nil {
		for i, recipe := range set.Recipes {
			if recipe == selection.ActiveRecipe {
				selection.ActiveSetIndex = i
				break
			}
		}
	}
	return selection
}

func (ui *bridgeUI) activateCombo(selection bridge.ComboSelection) {
	defer fyne.Do(func() {
		ui.comboLoading = false
	})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := ui.client.ActivateComboSelection(ctx, selection); err != nil {
		ui.appendLog("combo activate failed: " + err.Error())
		fyne.Do(func() {
			ui.comboStatus.SetText("Combo activate failed")
		})
		return
	}
	combos, err := ui.client.FetchCombos(ctx)
	if err == nil {
		fyne.Do(func() {
			ui.applyCombos(combos)
		})
	} else {
		fyne.Do(func() {
			ui.comboStatus.SetText("Combo selected")
		})
	}
	ui.appendLog("combo selected: " + selection.File + " / " + selection.SetID)
}

func (ui *bridgeUI) uploadComboYAML() {
	dialogBox := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil {
			ui.appendLog("combo upload failed: " + err.Error())
			return
		}
		if reader == nil {
			return
		}
		defer reader.Close()
		body, err := io.ReadAll(reader)
		if err != nil {
			ui.appendLog("combo upload failed: " + err.Error())
			return
		}
		name := filepath.Base(reader.URI().Path())
		ui.comboStatus.SetText("Uploading combo...")
		go ui.uploadCombo(name, body)
	}, ui.window)
	dialogBox.SetFilter(storage.NewExtensionFileFilter([]string{".yaml", ".yml"}))
	dialogBox.Show()
}

func (ui *bridgeUI) uploadCombo(name string, body []byte) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := ui.client.UploadCombo(ctx, name, body); err != nil {
		ui.appendLog("combo upload failed: " + err.Error())
		fyne.Do(func() {
			ui.comboStatus.SetText("Combo upload failed")
		})
		return
	}
	ui.appendLog("combo uploaded: " + name)
	combos, err := ui.client.FetchCombos(ctx)
	if err != nil {
		fyne.Do(func() {
			ui.comboStatus.SetText("Combo uploaded")
		})
		return
	}
	fyne.Do(func() {
		ui.applyCombos(combos)
	})
}

func (ui *bridgeUI) updateComboStatus() {
	if ui.comboState == nil || len(ui.comboState.Files) == 0 {
		ui.comboStatus.SetText("No combo YAML loaded")
		return
	}
	file := comboFileByName(ui.comboState, ui.comboFile.Selected)
	if file == nil {
		ui.comboStatus.SetText("Select combo file")
		return
	}
	set := comboSetByID(file, ui.comboSet.Selected)
	if set == nil {
		ui.comboStatus.SetText("Select combo set")
		return
	}
	ui.comboStatus.SetText(set.Name + " (" + set.Mode + ")")
}

func (ui *bridgeUI) applyProfiles(profiles *bridge.ProfilesResponse) {
	if profiles == nil {
		return
	}
	options := append([]string(nil), profiles.Profiles...)
	options = append(options, newProfileOption)
	ui.configProfile.Options = options
	ui.configProfile.Refresh()
	if profiles.Current != "" {
		ui.currentProfile = profiles.Current
		ui.configProfile.SetSelected(profiles.Current)
	}
}

func (ui *bridgeUI) selectedProfile() string {
	selected := strings.TrimSpace(ui.configProfile.Selected)
	if selected == newProfileOption {
		return ""
	}
	return selected
}

func (ui *bridgeUI) promptNewProfile() {
	name := widget.NewEntry()
	name.SetPlaceHolder("SF6-2.json")
	content := container.NewVBox(
		widget.NewLabel("New profile name"),
		name,
	)
	d := dialog.NewCustomConfirm("New Profile", "Create", "Cancel", content, func(ok bool) {
		if !ok {
			ui.restoreCurrentProfileSelection()
			return
		}
		profile := withJSONExtension(name.Text)
		if profile == "" {
			ui.appendLog("profile create failed: profile is empty")
			ui.restoreCurrentProfileSelection()
			return
		}
		ui.saveCurrentPreviewAs(profile)
	}, ui.window)
	d.Resize(fyne.NewSize(420, 180))
	d.Show()
}

func (ui *bridgeUI) restoreCurrentProfileSelection() {
	if ui.currentProfile != "" {
		ui.configProfile.SetSelected(ui.currentProfile)
		return
	}
	ui.configProfile.ClearSelected()
}

func (ui *bridgeUI) exitEditMode() {
	ui.editConfig = nil
	ui.setEditModeLayout(false)
	ui.previewBody.Objects = []fyne.CanvasObject{ui.matrixPanel()}
	ui.previewBody.Refresh()
	ui.editButton.Show()
	ui.editButton.Enable()
}

func (ui *bridgeUI) saveEditConfig() {
	var cfg bridge.OverlayConfig
	if err := json.Unmarshal([]byte(ui.editJSON.Text), &cfg); err != nil {
		ui.appendLog("config parse failed: " + err.Error())
		return
	}
	if cfg.Controller.Width <= 0 || cfg.Controller.Height <= 0 {
		ui.appendLog("config parse failed: controller width/height must be positive")
		return
	}
	ui.saveOverlayConfig(&cfg, ui.selectedProfile())
}

func (ui *bridgeUI) saveOverlayConfig(cfg *bridge.OverlayConfig, profile string) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := ui.client.SaveOverlayConfig(ctx, cfg, profile); err != nil {
		ui.appendLog("config save failed: " + err.Error())
		return
	}
	profiles, _ := ui.client.FetchProfiles(ctx)
	fyne.Do(func() {
		ui.renderPreview(cfg)
		ui.applyComboVolume(cfg)
		if profiles != nil {
			ui.applyProfiles(profiles)
		}
		if profile != "" {
			name := withJSONExtension(profile)
			ui.currentProfile = name
			ui.configProfile.SetSelected(name)
		}
		if ui.editConfig != nil {
			ui.exitEditMode()
		}
		ui.appendLog("config saved")
	})
}

func (ui *bridgeUI) syncEditJSON() {
	if ui.editConfig == nil {
		return
	}
	body, err := json.MarshalIndent(ui.editConfig, "", "  ")
	if err != nil {
		ui.appendLog("config encode failed: " + err.Error())
		return
	}
	ui.editJSON.SetText(string(body))
}
func (ui *bridgeUI) refreshEditPreview() {
	if ui.editConfig == nil || ui.rightContent == nil {
		return
	}
	// 編集モード中の描画更新も、右側全体に反映させる
	ui.setEditModeLayout(true)
}

func (ui *bridgeUI) setEditModeLayout(editing bool) {
	if ui.rightContent == nil {
		return
	}
	if editing {
		// 編集モード：右側全体を丸ごと編集用コンテンツにする（狭いカードの中に閉じ込めない）
		ui.rightContent.Objects = []fyne.CanvasObject{ui.editModeContent()}
	} else {
		// 通常モード：いつもの3段並び（プレビュー、ステータス、ログ）に戻す
		ui.rightContent.Objects = []fyne.CanvasObject{
			container.NewVBox(ui.previewCard, ui.devicePanel, ui.logPanel),
		}
	}
	ui.rightContent.Refresh()
}

func (ui *bridgeUI) editModeContent() fyne.CanvasObject {
	save := widget.NewButtonWithIcon("Save", theme.DocumentSaveIcon(), ui.saveEditConfig)
	cancel := widget.NewButtonWithIcon("Cancel", theme.CancelIcon(), ui.exitEditMode)
	reload := widget.NewButtonWithIcon("Reload", theme.ViewRefreshIcon(), ui.reloadEditConfig)

	// 上部の操作ツールバー
	toolbar := container.NewHBox(save, cancel, reload, layout.NewSpacer())

	controllerEditor := ui.controllerEditor()
	colorEditor := ui.globalButtonColorEditor()
	jsonLabel := widget.NewLabelWithStyle("JSON fine tuning", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	// 編集用プレビュー、フォーム、JSONを縦に並べる
	scrollContent := container.NewVBox(
		widget.NewLabel("Right click a button to edit its settings."),
		ui.editPreviewPanel(),
		widget.NewSeparator(),
		container.NewGridWithColumns(2, controllerEditor, colorEditor),
		jsonLabel,
		ui.editJSON,
	)

	// スクロールで包む
	scroll := container.NewVScroll(scrollContent)

	// ツールバーを上に固定し、残りの領域すべてをスクロール可能にする
	return container.NewBorder(toolbar, nil, nil, nil, scroll)
}

func (ui *bridgeUI) controllerEditor() fyne.CanvasObject {
	cfg := ui.editConfig
	x := numberEntry(cfg.Controller.X)
	y := numberEntry(cfg.Controller.Y)
	width := numberEntry(cfg.Controller.Width)
	height := numberEntry(cfg.Controller.Height)
	colorValue := widget.NewEntry()
	colorValue.SetText(cfg.Controller.Color)
	imageValue := widget.NewEntry()
	imageValue.SetText(cfg.Controller.Image)

	apply := widget.NewButton("Apply controller", func() {
		ui.editConfig.Controller.X = parseEntryInt(x, ui.editConfig.Controller.X)
		ui.editConfig.Controller.Y = parseEntryInt(y, ui.editConfig.Controller.Y)
		ui.editConfig.Controller.Width = parseEntryInt(width, ui.editConfig.Controller.Width)
		ui.editConfig.Controller.Height = parseEntryInt(height, ui.editConfig.Controller.Height)
		ui.editConfig.Controller.Color = strings.TrimSpace(colorValue.Text)
		ui.editConfig.Controller.Image = strings.TrimSpace(imageValue.Text)
		ui.syncEditJSON()
		ui.refreshEditPreview()
	})

	return container.NewVBox(
		widget.NewLabelWithStyle("Controller", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		colorPreviewRow("Color", cfg.Controller.Color, colorValue, color.NRGBA{R: 68, G: 88, B: 102, A: 210}),
		widget.NewForm(
			widget.NewFormItem("X", x),
			widget.NewFormItem("Y", y),
			widget.NewFormItem("Width", width),
			widget.NewFormItem("Height", height),
			widget.NewFormItem("Image", imageValue),
		),
		apply,
	)
}

func (ui *bridgeUI) globalButtonColorEditor() fyne.CanvasObject {
	colorValue := widget.NewEntry()
	pressedValue := widget.NewEntry()
	textValue := widget.NewEntry()
	historyValue := widget.NewEntry()
	base := bridge.ButtonConfig{}
	if len(ui.editConfig.Buttons) > 0 {
		base = ui.editConfig.Buttons[0]
		colorValue.SetText(base.Color)
		pressedValue.SetText(base.PressedColor)
		textValue.SetText(base.TextColor)
		historyValue.SetText(base.HistoryColor)
	}

	apply := widget.NewButton("Apply to all buttons", func() {
		colorText := strings.TrimSpace(colorValue.Text)
		pressedText := strings.TrimSpace(pressedValue.Text)
		textText := strings.TrimSpace(textValue.Text)
		historyText := strings.TrimSpace(historyValue.Text)
		for i := range ui.editConfig.Buttons {
			ui.editConfig.Buttons[i].Color = colorText
			ui.editConfig.Buttons[i].PressedColor = pressedText
			ui.editConfig.Buttons[i].TextColor = textText
			ui.editConfig.Buttons[i].HistoryColor = historyText
		}
		ui.syncEditJSON()
		ui.refreshEditPreview()
	})

	return container.NewVBox(
		widget.NewLabelWithStyle("All Button Colors", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		colorPreviewRow("Color", base.Color, colorValue, matrixActiveColor(base.ID)),
		colorPreviewRow("Pressed", base.PressedColor, pressedValue, matrixActiveColor(base.ID)),
		colorPreviewRow("Text", base.TextColor, textValue, color.NRGBA{R: 245, G: 248, B: 250, A: 255}),
		colorPreviewRow("History", base.HistoryColor, historyValue, matrixActiveColor(base.ID)),
		apply,
	)
}

func (ui *bridgeUI) editPreviewPanel() fyne.CanvasObject {
	cfg := ui.editConfig
	if cfg == nil {
		cfg = defaultPreviewConfig()
	}
	panelWidth := float32(580)
	panelHeight := float32(230)
	scale := previewScale(cfg, panelWidth, panelHeight)
	panel := container.NewWithoutLayout()

	controller := canvas.NewRectangle(parseColor(cfg.Controller.Color, color.NRGBA{R: 68, G: 88, B: 102, A: 210}))
	controller.StrokeColor = color.NRGBA{R: 220, G: 190, B: 80, A: 230}
	controller.StrokeWidth = 2
	controller.CornerRadius = 8
	controller.Move(fyne.NewPos(scale.x(cfg.Controller.X), scale.y(cfg.Controller.Y)))
	controller.Resize(fyne.NewSize(scale.size(cfg.Controller.Width), scale.size(cfg.Controller.Height)))
	panel.Add(controller)
	var controllerHandle *draggableRect
	controllerHandle = newDraggableRect(color.NRGBA{R: 220, G: 190, B: 80, A: 20}, func(dx, dy float32) {
		ui.editConfig.Controller.X += int(math.Round(float64(dx / scale.scale)))
		ui.editConfig.Controller.Y += int(math.Round(float64(dy / scale.scale)))
		controller.Move(controller.Position().Add(fyne.NewDelta(dx, dy)))
		controllerHandle.Move(controllerHandle.Position().Add(fyne.NewDelta(dx, dy)))
		controller.Refresh()
		controllerHandle.Refresh()
		ui.syncEditJSON()
	}, ui.refreshEditPreview, nil)
	controllerHandle.Move(controller.Position())
	controllerHandle.Resize(controller.Size())
	panel.Add(controllerHandle)

	for i := range cfg.Buttons {
		button := cfg.Buttons[i]
		if !button.IsVisible() {
			continue
		}
		size := scale.size(button.Size)
		x := scale.x(button.X)
		y := scale.y(button.Y)
		label := button.Label
		if label == "" {
			label = button.ID
		}
		c := canvas.NewCircle(parseColor(button.Color, matrixActiveColor(button.ID)))
		c.StrokeColor = color.NRGBA{R: 220, G: 235, B: 245, A: 210}
		c.StrokeWidth = 2
		c.Move(fyne.NewPos(x, y))
		c.Resize(fyne.NewSize(size, size))
		t := canvas.NewText(label, color.NRGBA{R: 245, G: 248, B: 250, A: 255})
		t.Alignment = fyne.TextAlignCenter
		t.TextStyle = fyne.TextStyle{Bold: true}
		t.TextSize = 12
		t.Move(fyne.NewPos(x, y+(size-16)/2))
		t.Resize(fyne.NewSize(size, 16))
		panel.Add(c)
		panel.Add(t)
		index := i
		var handle *draggableRect
		handle = newDraggableRect(color.NRGBA{R: 220, G: 190, B: 80, A: 10}, func(dx, dy float32) {
			ui.editConfig.Buttons[index].X += int(math.Round(float64(dx / scale.scale)))
			ui.editConfig.Buttons[index].Y += int(math.Round(float64(dy / scale.scale)))
			delta := fyne.NewDelta(dx, dy)
			c.Move(c.Position().Add(delta))
			t.Move(t.Position().Add(delta))
			handle.Move(handle.Position().Add(delta))
			c.Refresh()
			t.Refresh()
			handle.Refresh()
			ui.syncEditJSON()
		}, ui.refreshEditPreview, func() {
			ui.openButtonSettings(index)
		})
		handle.Move(fyne.NewPos(x, y))
		handle.Resize(fyne.NewSize(size, size))
		panel.Add(handle)
	}

	panelContainer := container.New(newFixedSizeLayout(fyne.NewSize(panelWidth, panelHeight)), panel)
	return container.NewCenter(panelContainer)
}

func (ui *bridgeUI) openButtonSettings(index int) {
	if ui.editConfig == nil || index < 0 || index >= len(ui.editConfig.Buttons) {
		return
	}
	button := ui.editConfig.Buttons[index]
	visible := widget.NewCheck("Visible", nil)
	visible.SetChecked(button.IsVisible())
	label := widget.NewEntry()
	label.SetText(button.Label)
	historyLabel := widget.NewEntry()
	historyLabel.SetText(button.HistoryLabel)
	colorValue := widget.NewEntry()
	colorValue.SetText(button.Color)
	pressedValue := widget.NewEntry()
	pressedValue.SetText(button.PressedColor)
	textValue := widget.NewEntry()
	textValue.SetText(button.TextColor)
	size := numberEntry(button.Size)

	form := container.NewVBox(
		visible,
		widget.NewLabelWithStyle("Color Preview", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		colorPreviewRow("Color", button.Color, colorValue, matrixActiveColor(button.ID)),
		colorPreviewRow("Pressed", button.PressedColor, pressedValue, matrixActiveColor(button.ID)),
		colorPreviewRow("Text", button.TextColor, textValue, color.NRGBA{R: 245, G: 248, B: 250, A: 255}),
		widget.NewSeparator(),
		widget.NewForm(
			widget.NewFormItem("Label", label),
			widget.NewFormItem("History label", historyLabel),
			widget.NewFormItem("Size", size),
		),
	)
	content := container.New(newFixedSizeLayout(fyne.NewSize(720, 360)), container.NewVScroll(form))
	d := dialog.NewCustomConfirm("Button: "+button.ID, "Apply", "Cancel", content, func(ok bool) {
		if !ok || ui.editConfig == nil || index >= len(ui.editConfig.Buttons) {
			return
		}
		v := visible.Checked
		ui.editConfig.Buttons[index].Visible = &v
		ui.editConfig.Buttons[index].Label = label.Text
		ui.editConfig.Buttons[index].HistoryLabel = historyLabel.Text
		ui.editConfig.Buttons[index].Color = strings.TrimSpace(colorValue.Text)
		ui.editConfig.Buttons[index].PressedColor = strings.TrimSpace(pressedValue.Text)
		ui.editConfig.Buttons[index].TextColor = strings.TrimSpace(textValue.Text)
		ui.editConfig.Buttons[index].Size = parseEntryInt(size, ui.editConfig.Buttons[index].Size)
		ui.syncEditJSON()
		ui.refreshEditPreview()
	}, ui.window)
	d.Resize(fyne.NewSize(780, 470))
	d.Show()
}

func (ui *bridgeUI) applyServerSettings() {
	if ui == nil || ui.client == nil {
		return
	}
	custom := ui.customServer != nil && ui.customServer.Checked
	host := defaultServerHost
	if custom && ui.serverHost != nil {
		host = normalizeServerHost(ui.serverHost.Text)
		if host == "" {
			host = defaultServerHost
		}
		ui.app.Preferences().SetString(prefServerHost, host)
	}
	serverURL := serverURLForSettings(custom, host)
	ui.app.Preferences().SetString(prefServerURL, serverURL)
	ui.client.SetURL(serverURL)
}

func (ui *bridgeUI) setServerHostEnabled(enabled bool) {
	if ui.serverHost == nil {
		return
	}
	if enabled {
		ui.serverHost.Enable()
		return
	}
	ui.serverHost.Disable()
}

func serverURLForSettings(custom bool, host string) string {
	if !custom {
		host = defaultServerHost
	}
	host = normalizeServerHost(host)
	if host == "" {
		host = defaultServerHost
	}
	return "http://" + host + serverInputEndpoint
}

func serverHostPlaceholder(custom bool) string {
	if custom {
		return "192.168.0.10 or server-name"
	}
	return defaultServerHost
}

func normalizeServerHost(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	parsed := value
	if !strings.Contains(parsed, "://") {
		parsed = "//" + parsed
	}
	u, err := url.Parse(parsed)
	if err == nil && u.Host != "" {
		value = u.Host
	}
	value = strings.TrimPrefix(value, "http://")
	value = strings.TrimPrefix(value, "https://")
	value = strings.Trim(value, "/")
	if value == "localhost" {
		return defaultServerHost
	}
	if strings.Contains(value, ":") {
		return value
	}
	return value + ":8080"
}

func serverHostFromURL(value string) string {
	host := normalizeServerHost(value)
	if host == "" {
		return defaultServerHost
	}
	return host
}

func (ui *bridgeUI) start() {
	if ui.running {
		return
	}

	ui.applyServerSettings()

	interval := ui.scanIntervalDuration()
	ctx, cancel := context.WithCancel(context.Background())
	ui.cancel = cancel
	ui.running = true
	ui.setRunningState(true)
	ui.appendLog("started")
	ui.setConnection(false, "Connecting")

	updates := make(chan gamepad.Snapshot, 32)
	backend := gamepad.NewBackend(interval)
	go ui.watchConnection(ctx)
	go func() {
		if err := backend.Run(ctx, updates); err != nil && ctx.Err() == nil {
			updates <- gamepad.Snapshot{Err: err}
		}
	}()
	go ui.consumeUpdates(ctx, updates)
}

func (ui *bridgeUI) stop() {
	if !ui.running {
		return
	}
	if ui.cancel != nil {
		ui.cancel()
		ui.cancel = nil
	}
	ui.running = false
	ui.connected.Store(false)
	ui.setRunningState(false)
	ui.setConnection(false, "Not started")
	ui.appendLog("stopped")
}

func (ui *bridgeUI) watchConnection(ctx context.Context) {
	ticker := time.NewTicker(healthInterval)
	defer ticker.Stop()

	ui.checkConnection(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ui.checkConnection(ctx)
		}
	}
}

func (ui *bridgeUI) checkConnection(ctx context.Context) {
	if err := ui.client.Check(ctx); err != nil {
		ui.connected.Store(false)
		ui.setConnection(false, "Disconnected")
		return
	}
	wasConnected := ui.connected.Swap(true)
	ui.setConnection(true, "Connected")
	if !wasConnected {
		ui.loadProfilesAndCurrentConfig()
	}
}

func (ui *bridgeUI) consumeUpdates(ctx context.Context, updates <-chan gamepad.Snapshot) {
	for {
		select {
		case <-ctx.Done():
			return
		case snapshot := <-updates:
			if snapshot.Err != nil {
				ui.setStatus("Error")
				ui.appendLog(snapshot.Err.Error())
				continue
			}
			if snapshot.Message != "" {
				ui.appendLog(snapshot.Message)
			}
			if snapshot.DeviceName != "" {
				ui.setDevice(snapshot.DeviceName)
			}
			if snapshot.Buttons == nil {
				continue
			}
			ui.updateFacingFromButtons(snapshot.Buttons)
			ui.setPreview(snapshot.Buttons)
			if !ui.connected.Load() {
				continue
			}
			sent, err := ui.client.Send(ctx, bridge.State{
				DeviceID: "gamepad",
				Buttons:  bridge.CloneButtons(snapshot.Buttons),
			})
			if err != nil {
				ui.connected.Store(false)
				ui.setConnection(false, "Disconnected")
				continue
			}
			ui.setStatus("Running")
			if sent {
				ui.setLastSend(time.Now().Format("15:04:05"))
			}
		}
	}
}

func (ui *bridgeUI) updateFacingFromButtons(buttons map[string]bool) {
	pressed := buttons["up"] && (buttons["s1"] || buttons["ba"] || buttons["BA"] || buttons["back"] || buttons["BACK"])
	if !pressed {
		ui.facingToggleDown = false
		return
	}
	if ui.facingToggleDown {
		return
	}
	ui.facingToggleDown = true
	ui.facingRight = !ui.facingRight
	ui.setFacingStatus(ui.facingRight)
}

func (ui *bridgeUI) scanIntervalDuration() time.Duration {
	ms, err := strconv.Atoi(strings.TrimSpace(ui.scanInterval.Text))
	if err != nil || ms < minScanInterval {
		ms = defaultScanInterval
	}
	ui.app.Preferences().SetInt(prefScanIntervalMS, ms)
	ui.scanInterval.SetText(strconv.Itoa(ms))
	return time.Duration(ms) * time.Millisecond
}

func (ui *bridgeUI) setRunningState(running bool) {
	fyne.Do(func() {
		if running {
			ui.status.SetText("Running")
			ui.startButton.Disable()
			ui.stopButton.Enable()
			ui.customServer.Disable()
			ui.serverHost.Disable()
			ui.scanInterval.Disable()
			return
		}
		ui.status.SetText("Stopped")
		ui.startButton.Enable()
		ui.stopButton.Disable()
		ui.customServer.Enable()
		ui.setServerHostEnabled(ui.customServer.Checked)
		ui.scanInterval.Enable()
	})
}

func (ui *bridgeUI) setConnection(connected bool, label string) {
	fyne.Do(func() {
		if !ui.running {
			ui.connectionIcon.SetResource(antennaResource(theme.ColorNameDisabled))
			ui.connectionStatus.SetText("Not started")
			return
		}
		if connected {
			ui.connectionIcon.SetResource(antennaResource(theme.ColorNameSuccess))
			ui.connectionStatus.SetText(label)
			return
		}
		ui.connectionIcon.SetResource(antennaResource(theme.ColorNameError))
		ui.connectionStatus.SetText(label)
	})
}

func (ui *bridgeUI) setStatus(value string) {
	fyne.Do(func() {
		ui.status.SetText(value)
	})
}

func (ui *bridgeUI) setDevice(value string) {
	fyne.Do(func() {
		ui.device.SetText(value)
	})
}

func (ui *bridgeUI) setLastSend(value string) {
	fyne.Do(func() {
		ui.lastSend.SetText(value)
	})
}

func (ui *bridgeUI) setFacingStatus(right bool) {
	fyne.Do(func() {
		ui.facingStatus.SetText(facingStatusText(right))
	})
}

func facingStatusText(right bool) string {
	if right {
		return "Facing: 右向き (1P)"
	}
	return "Facing: 左向き (2P)"
}

func (ui *bridgeUI) setPreview(buttons map[string]bool) {
	fyne.Do(func() {
		for id, circle := range ui.matrix {
			if buttons[id] {
				circle.FillColor = ui.matrixActive[id]
				circle.StrokeColor = color.NRGBA{R: 220, G: 235, B: 245, A: 255}
			} else {
				circle.FillColor = ui.matrixInactive[id]
				circle.StrokeColor = color.NRGBA{R: 128, G: 146, B: 158, A: 210}
			}
			circle.Refresh()
		}
	})
}

func (ui *bridgeUI) appendLog(message string) {
	fyne.Do(func() {
		line := fmt.Sprintf("%s %s", time.Now().Format("15:04:05"), message)
		ui.logLines = append([]string{line}, ui.logLines...)
		if len(ui.logLines) > maxLogLines {
			ui.logLines = ui.logLines[:maxLogLines]
		}
		ui.log.SetText(strings.Join(ui.logLines, "\n"))
	})
}

func setupTray(a fyne.App, w fyne.Window, ui *bridgeUI) bool {
	if !shouldEnableTray() {
		if ui.closeToTray != nil {
			ui.closeToTray.Disable()
		}
		return false
	}
	desk, ok := a.(desktop.App)
	if !ok {
		return false
	}
	menu := fyne.NewMenu("Input Cast Client",
		fyne.NewMenuItem("Show Window", func() {
			w.Show()
			w.RequestFocus()
		}),
		fyne.NewMenuItem("Start", ui.start),
		fyne.NewMenuItem("Stop", ui.stop),
		fyne.NewMenuItem("Quit", func() {
			ui.stop()
			a.Quit()
		}),
	)
	desk.SetSystemTrayMenu(menu)
	desk.SetSystemTrayIcon(trayIconResource)
	desk.SetSystemTrayWindow(w)
	return true
}

func shouldEnableTray() bool {
	switch runtime.GOOS {
	case "windows", "darwin":
		return true
	case "linux":
		if os.Geteuid() == 0 || os.Getenv("SUDO_USER") != "" {
			return false
		}
		return os.Getenv("DBUS_SESSION_BUS_ADDRESS") != ""
	default:
		return false
	}
}

func antennaResource(color fyne.ThemeColorName) fyne.Resource {
	return theme.NewColoredResource(antennaIconResource, color)
}

var antennaIconResource = fyne.NewStaticResource("antenna.svg", []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M12 21V11" fill="none" stroke="#000" stroke-width="2" stroke-linecap="round"/>
<circle cx="12" cy="9" r="2" fill="#000"/>
<path d="M8 13a6 6 0 0 1 0-8M16 13a6 6 0 0 0 0-8M5 16a10 10 0 0 1 0-14M19 16a10 10 0 0 0 0-14" fill="none" stroke="#000" stroke-width="2" stroke-linecap="round"/>
</svg>`))

//go:embed assets/display-icon.png
var appIconBytes []byte

var appIconResource = fyne.NewStaticResource("display-icon.png", appIconBytes)

//go:embed assets/tray-icon.png
var trayIconBytes []byte

var trayIconResource = fyne.NewStaticResource("tray-icon.png", trayIconBytes)

var toolsIconResource = fyne.NewStaticResource("tools.svg", []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M4 20l6.6-6.6" fill="none" stroke="#000" stroke-width="2" stroke-linecap="round"/>
<path d="M14 3l7 7-2.6 2.6-2.1-2.1-4.5 4.5-2.8-2.8 4.5-4.5L11.4 5.6z" fill="none" stroke="#000" stroke-width="2" stroke-linejoin="round"/>
<path d="M5.5 4.5l4 4M3.8 6.2l4-4 2.5 2.5-4 4z" fill="none" stroke="#000" stroke-width="2" stroke-linejoin="round"/>
</svg>`))

func inactiveColor() color.NRGBA {
	return color.NRGBA{R: 108, G: 124, B: 135, A: 150}
}

type previewTransform struct {
	minX  float32
	minY  float32
	scale float32
	padX  float32
	padY  float32
}

func (t previewTransform) x(value int) float32 {
	return t.padX + (float32(value)-t.minX)*t.scale
}

func (t previewTransform) y(value int) float32 {
	return t.padY + (float32(value)-t.minY)*t.scale
}

func (t previewTransform) size(value int) float32 {
	size := float32(value) * t.scale
	if size < 10 {
		return 10
	}
	return size
}

func previewScale(cfg *bridge.OverlayConfig, width, height float32) previewTransform {
	minX := float32(cfg.Controller.X)
	minY := float32(cfg.Controller.Y)
	maxX := float32(cfg.Controller.X + cfg.Controller.Width)
	maxY := float32(cfg.Controller.Y + cfg.Controller.Height)
	for _, button := range cfg.Buttons {
		if !button.IsVisible() {
			continue
		}
		x := float32(button.X)
		y := float32(button.Y)
		size := float32(button.Size)
		if x < minX {
			minX = x
		}
		if y < minY {
			minY = y
		}
		if x+size > maxX {
			maxX = x + size
		}
		if y+size > maxY {
			maxY = y + size
		}
	}

	padding := float32(18)
	usableWidth := width - padding*2
	usableHeight := height - padding*2 - 18
	sourceWidth := maxX - minX
	sourceHeight := maxY - minY
	if sourceWidth <= 0 {
		sourceWidth = 1
	}
	if sourceHeight <= 0 {
		sourceHeight = 1
	}
	scale := usableWidth / sourceWidth
	if hScale := usableHeight / sourceHeight; hScale < scale {
		scale = hScale
	}
	return previewTransform{
		minX:  minX,
		minY:  minY,
		scale: scale,
		padX:  padding + (usableWidth-sourceWidth*scale)/2,
		padY:  padding + (usableHeight-sourceHeight*scale)/2,
	}
}

func parseColor(value string, fallback color.NRGBA) color.NRGBA {
	trimmed := strings.TrimPrefix(strings.TrimSpace(value), "#")
	if len(trimmed) != 6 {
		return fallback
	}
	n, err := strconv.ParseUint(trimmed, 16, 32)
	if err != nil {
		return fallback
	}
	return color.NRGBA{
		R: uint8(n >> 16),
		G: uint8(n >> 8),
		B: uint8(n),
		A: fallback.A,
	}
}

func numberEntry(value int) *widget.Entry {
	entry := widget.NewEntry()
	entry.SetText(strconv.Itoa(value))
	return entry
}

func parseEntryInt(entry *widget.Entry, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(entry.Text))
	if err != nil {
		return fallback
	}
	return value
}

func colorPreviewRow(label, current string, entry *widget.Entry, fallback color.NRGBA) fyne.CanvasObject {
	currentSwatch := colorSwatch(parseColor(current, fallback))
	currentValue := widget.NewLabel(currentColorText(current))
	nextRect := canvas.NewRectangle(parseColor(entry.Text, fallback))
	nextRect.CornerRadius = 4
	entry.OnChanged = func(value string) {
		nextRect.FillColor = parseColor(value, fallback)
		nextRect.Refresh()
	}
	arrow := widget.NewLabel("→")
	arrow.Alignment = fyne.TextAlignCenter
	nextField := container.NewHBox(
		fixedSwatch(nextRect),
		container.New(newFixedSizeLayout(fyne.NewSize(180, 40)), entry),
	)
	return container.NewBorder(nil, nil,
		container.New(newFixedSizeLayout(fyne.NewSize(88, 40)), widget.NewLabel(label)),
		nil,
		container.NewHBox(
			currentSwatch,
			container.New(newFixedSizeLayout(fyne.NewSize(80, 40)), currentValue),
			container.New(newFixedSizeLayout(fyne.NewSize(24, 40)), arrow),
			nextField,
		),
	)
}

func currentColorText(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "-"
	}
	return trimmed
}

func colorSwatch(fill color.NRGBA) fyne.CanvasObject {
	rect := canvas.NewRectangle(fill)
	rect.CornerRadius = 4
	return fixedSwatch(rect)
}

func fixedSwatch(rect *canvas.Rectangle) fyne.CanvasObject {
	rect.StrokeColor = color.NRGBA{R: 230, G: 236, B: 240, A: 180}
	rect.StrokeWidth = 1
	return container.New(newFixedSizeLayout(fyne.NewSize(72, 28)), rect)
}

func withJSONExtension(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return ""
	}
	if strings.EqualFold(filepathExt(trimmed), ".json") {
		return trimmed
	}
	return trimmed + ".json"
}

func filepathExt(name string) string {
	index := strings.LastIndex(name, ".")
	if index < 0 {
		return ""
	}
	return name[index:]
}

func stringInSlice(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func comboFileByName(combos *bridge.ComboResponse, name string) *bridge.ComboFileSummary {
	if combos == nil {
		return nil
	}
	for i := range combos.Files {
		if combos.Files[i].File == name {
			return &combos.Files[i]
		}
	}
	return nil
}

func comboSetByID(file *bridge.ComboFileSummary, id string) *bridge.ComboSetSummary {
	if file == nil {
		return nil
	}
	for i := range file.Sets {
		if file.Sets[i].ID == id {
			return &file.Sets[i]
		}
	}
	return nil
}

func comboRecipeOptions(combos *bridge.ComboResponse) []string {
	if combos == nil || combos.ActiveSet == nil {
		return nil
	}
	options := make([]string, 0, len(combos.ActiveSet.Combos))
	for _, combo := range combos.ActiveSet.Combos {
		if combo.ID != "" {
			options = append(options, combo.ID)
		}
	}
	return options
}

func comboPracticeSetOptions(combos *bridge.ComboResponse) []string {
	if combos == nil {
		return nil
	}
	options := make([]string, 0, len(combos.PracticeSets))
	for _, set := range combos.PracticeSets {
		options = append(options, set.ID)
	}
	return options
}

func practiceSetByID(combos *bridge.ComboResponse, id string) *bridge.PracticeSet {
	if combos == nil {
		return nil
	}
	for i := range combos.PracticeSets {
		if combos.PracticeSets[i].ID == id {
			return &combos.PracticeSets[i]
		}
	}
	return nil
}

func comboSelectionMatches(left, right bridge.ComboSelection) bool {
	return left.File == right.File &&
		left.SetID == right.SetID &&
		left.Mode == right.Mode &&
		left.ActiveRecipe == right.ActiveRecipe &&
		left.ActiveSet == right.ActiveSet &&
		left.ActiveSetIndex == right.ActiveSetIndex &&
		boolPointerValue(left.Loop) == boolPointerValue(right.Loop) &&
		boolPointerValue(left.AdvanceOnComplete) == boolPointerValue(right.AdvanceOnComplete)
}

func boolPointerValue(value *bool) bool {
	return value != nil && *value
}

func defaultPreviewConfig() *bridge.OverlayConfig {
	visible := true
	return &bridge.OverlayConfig{
		ComboDisplay: bridge.ComboDisplayConfig{Enabled: true, X: 398, Y: 120, Width: 420, Height: 520, ShowBorder: true},
		ComboAudio:   bridge.ComboAudioConfig{Volume: 0.7},
		Controller:   bridge.ControllerConfig{X: 855, Y: 946, Width: 210, Height: 120, Color: "#5f7e91"},
		Buttons: []bridge.ButtonConfig{
			{ID: "left", Label: "←", Visible: &visible, X: 860, Y: 987, Size: 24},
			{ID: "down", Label: "↓", Visible: &visible, X: 889, Y: 979, Size: 24},
			{ID: "right", Label: "→", Visible: &visible, X: 918, Y: 981, Size: 24},
			{ID: "up", Label: "↑", Visible: &visible, X: 936, Y: 1035, Size: 24},
			{ID: "b1", Label: "A", Visible: &visible, X: 947, Y: 999, Size: 24},
			{ID: "b2", Label: "B", Visible: &visible, X: 976, Y: 991, Size: 24},
			{ID: "b3", Label: "X", Visible: &visible, X: 948, Y: 969, Size: 24},
			{ID: "b4", Label: "Y", Visible: &visible, X: 975, Y: 961, Size: 24},
			{ID: "l1", Label: "LB", Visible: &visible, X: 1032, Y: 960, Size: 24},
			{ID: "l2", Label: "LT", Visible: &visible, X: 1033, Y: 990, Size: 24},
			{ID: "r1", Label: "RB", Visible: &visible, X: 1003, Y: 958, Size: 24},
			{ID: "r2", Label: "RT", Visible: &visible, X: 1004, Y: 988, Size: 24},
			{ID: "l3", Label: "LS", Visible: &visible, X: 909, Y: 1023, Size: 24},
			{ID: "r3", Label: "RS", Visible: &visible, X: 969, Y: 1020, Size: 24},
		},
	}
}

func cloneOverlayConfig(src *bridge.OverlayConfig) *bridge.OverlayConfig {
	if src == nil {
		return nil
	}
	dst := &bridge.OverlayConfig{
		ComboDisplay: src.ComboDisplay,
		ComboAudio:   src.ComboAudio,
		Controller:   src.Controller,
		Buttons:      make([]bridge.ButtonConfig, len(src.Buttons)),
	}
	copy(dst.Buttons, src.Buttons)
	return dst
}

func activeDirectionColor() color.NRGBA {
	return color.NRGBA{R: 152, G: 190, B: 112, A: 255}
}

func activeShoulderColor() color.NRGBA {
	return color.NRGBA{R: 112, G: 166, B: 205, A: 255}
}

func activeSystemColor() color.NRGBA {
	return color.NRGBA{R: 172, G: 148, B: 204, A: 255}
}

func activeStickColor() color.NRGBA {
	return color.NRGBA{R: 160, G: 174, B: 184, A: 255}
}

func matrixActiveColor(id string) color.NRGBA {
	switch id {
	case "up", "down", "left", "right":
		return activeDirectionColor()
	case "l1", "l2", "r1", "r2":
		return activeShoulderColor()
	case "s1", "s2", "a1", "a2":
		return activeSystemColor()
	case "l3", "r3":
		return activeStickColor()
	case "b1":
		return color.NRGBA{R: 105, G: 166, B: 92, A: 255}
	case "b2":
		return color.NRGBA{R: 192, G: 85, B: 89, A: 255}
	case "b3":
		return color.NRGBA{R: 78, G: 139, B: 178, A: 255}
	case "b4":
		return color.NRGBA{R: 174, G: 185, B: 82, A: 255}
	default:
		return color.NRGBA{R: 150, G: 170, B: 184, A: 255}
	}
}
