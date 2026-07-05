(() => {
  const status = document.getElementById('status')
  const facingStatus = document.getElementById('facing-status')
  const log = document.getElementById('log')
  const customServerInput = document.getElementById('custom-server')
  const serverHostInput = document.getElementById('server-host')
  const gamepadsPanel = document.getElementById('gamepads')
  const hidStatus = document.getElementById('hid-status')
  const comboFile = document.getElementById('combo-file')
  const comboSet = document.getElementById('combo-set')
  const comboMode = document.getElementById('combo-mode')
  const comboRecipe = document.getElementById('combo-recipe')
  const comboPracticeSet = document.getElementById('combo-practice-set')
  const comboLoop = document.getElementById('combo-loop')
  const comboAdvance = document.getElementById('combo-advance')
  const comboUpload = document.getElementById('combo-upload')
  const comboStatus = document.getElementById('combo-status')
  const layoutSvg = document.getElementById('layout-editor')
  const layoutTarget = document.getElementById('layout-target')
  const layoutStatus = document.getElementById('layout-status')
  const configProfile = document.getElementById('config-profile')
  const configName = document.getElementById('config-name')
  const layoutInputs = {
    visible: document.getElementById('layout-visible'),
    label: document.getElementById('layout-label'),
    historyLabel: document.getElementById('layout-history-label'),
    historyColor: document.getElementById('layout-history-color'),
    color: document.getElementById('layout-color'),
    bgColor: document.getElementById('layout-bg-color'),
    bgImage: document.getElementById('layout-bg-image'),
    bgUpload: document.getElementById('layout-bg-upload'),
    comboAudioVolume: document.getElementById('combo-audio-volume'),
    comboAudioVolumeLabel: document.getElementById('combo-audio-volume-label'),
    x: document.getElementById('layout-x'),
    y: document.getElementById('layout-y'),
    size: document.getElementById('layout-size'),
    width: document.getElementById('layout-width'),
    height: document.getElementById('layout-height')
  }
  const buttonMap = {
    0: 'b1',
    1: 'b2',
    2: 'b3',
    3: 'b4',
    4: 'l1',
    5: 'r1',
    6: 'l2',
    7: 'r2',
    8: 's1',
    9: 's2',
    10: 'l3',
    11: 'r3',
    12: 'up',
    13: 'down',
    14: 'left',
    15: 'right',
    16: 'a1',
    17: 'a2'
  }
  let running = false
  let timer = null
  let lastPayload = ''
  let layoutConfig = null
  let selectedLayout = 'controller'
  let draggingLayout = null
  let currentProfile = 'default.json'
  let comboState = null
  let comboLoading = false
  let facingRight = true
  let facingToggleDown = false
  const basePath = detectBasePath()
  const defaultServerHost = 'localhost:8080'
  const serverInputEndpoint = '/api/input/gamepad'
  const storageKeys = {
    customServer: 'inputCastGamepad.customServer',
    serverHost: 'inputCastGamepad.serverHost'
  }

  loadServerSettings()

  function detectBasePath(){
    const path = location.pathname.replace(/\/+$/, '')
    return path.replace(/\/gamepad$/, '')
  }

  function appPath(path){
    return (basePath || '') + path
  }

  function currentServerURL(){
    return 'http://' + currentServerHost() + appPath(serverInputEndpoint)
  }

  function currentServerHost(){
    if (!customServerInput.checked) return defaultServerHost
    return normalizeServerHost(serverHostInput.value) || defaultServerHost
  }

  function loadServerSettings(){
    const customServer = readStorage(storageKeys.customServer) === 'true'
    const storedHost = normalizeServerHost(readStorage(storageKeys.serverHost) || defaultServerHost)
    customServerInput.checked = customServer
    serverHostInput.value = customServer ? storedHost : defaultServerHost
    syncServerControls()
  }

  function saveServerSettings(){
    writeStorage(storageKeys.customServer, customServerInput.checked ? 'true' : 'false')
    writeStorage(storageKeys.serverHost, currentServerHost())
  }

  function syncServerControls(){
    serverHostInput.disabled = !customServerInput.checked
    serverHostInput.placeholder = customServerInput.checked ? '192.168.0.10 or server-name' : defaultServerHost
    if (!customServerInput.checked) {
      serverHostInput.value = defaultServerHost
    }
  }

  function normalizeServerHost(value){
    value = String(value || '').trim()
    if (!value) return ''
    try {
      const url = new URL(value.includes('://') ? value : 'http://' + value)
      if (url.host) value = url.host
    } catch (error) {
      value = value.replace(/^https?:\/\//i, '')
    }
    value = value.replace(/^https?:\/\//i, '').replace(/^\/+|\/+$/g, '')
    if (value === 'localhost') return defaultServerHost
    if (value.includes(':')) return value
    return value + ':8080'
  }

  function readStorage(key){
    try {
      return window.localStorage.getItem(key)
    } catch (error) {
      return null
    }
  }

  function writeStorage(key, value){
    try {
      window.localStorage.setItem(key, value)
    } catch (error) {
      append('settings save failed: ' + error)
    }
  }

  function assetPath(path){
    const value = String(path || '')
    if (!value || /^(?:[a-z]+:)?\/\//i.test(value) || /^(?:data|blob):/i.test(value)) return value
    if (value.startsWith('/')) return appPath(value)
    return value
  }

  function append(message){
    log.textContent = new Date().toLocaleTimeString() + ' ' + message + '\n' + log.textContent
  }

  function sendState(state){
    const payload = JSON.stringify(state)
    if (payload === lastPayload) return
    lastPayload = payload
    fetch(currentServerURL(), {
      method: 'POST',
      headers: {'Content-Type': 'application/json'},
      body: payload
    }).then(response => {
      if (!response.ok) append('send failed: ' + response.status)
    }).catch(error => {
      append('send error: ' + error)
    })
    append('sent: ' + payload)
  }

  function readGamepads(){
    const gamepad = findGamepad()
    if (!gamepad) {
      status.textContent = 'No gamepad visible. Press a GP2040-CE button or reconnect it.'
      renderGamepads()
      return
    }
    renderGamepads()
    status.textContent = 'Reading: ' + gamepad.id
    const buttons = normalizeButtons(gamepad)
    updateFacingFromButtons(buttons)
    sendState({
      device_id: 'gamepad',
      buttons
    })
  }

  function findGamepad(){
    const gamepads = navigator.getGamepads ? navigator.getGamepads() : []
    return Array.from(gamepads).find(Boolean) || null
  }

  function renderGamepads(){
    if (!navigator.getGamepads) {
      gamepadsPanel.textContent = 'navigator.getGamepads is not available in this browser.'
      return
    }
    const gamepads = Array.from(navigator.getGamepads())
    const visible = gamepads
      .map((gamepad, index) => ({ gamepad, index }))
      .filter(item => item.gamepad)
    if (visible.length === 0) {
      gamepadsPanel.textContent = 'No gamepad slots are populated.'
      return
    }
    gamepadsPanel.innerHTML = visible.map(({ gamepad, index }) => {
      const pressed = gamepad.buttons
        .map((button, buttonIndex) => button.pressed ? buttonIndex : null)
        .filter(buttonIndex => buttonIndex !== null)
        .join(', ')
      const axes = gamepad.axes.map(value => value.toFixed(2)).join(', ')
      return [
        '<div>',
        '<strong>#' + index + '</strong> ' + escapeHTML(gamepad.id),
        '<br>mapping: ' + escapeHTML(gamepad.mapping || '-'),
        '<br>buttons: ' + gamepad.buttons.length + ' pressed: ' + escapeHTML(pressed || '-'),
        '<br>axes: ' + escapeHTML(axes || '-'),
        '</div>'
      ].join('')
    }).join('<hr>')
  }

  function normalizeButtons(gamepad){
    const buttons = {}
    Object.values(buttonMap).forEach(id => { buttons[id] = false })
    gamepad.buttons.forEach((button, index) => {
      const id = buttonMap[index]
      if (id) buttons[id] = !!button.pressed
    })
    if (gamepad.axes && gamepad.axes.length >= 2) {
      buttons.left = buttons.left || gamepad.axes[0] < -0.5
      buttons.right = buttons.right || gamepad.axes[0] > 0.5
      buttons.up = buttons.up || gamepad.axes[1] < -0.5
      buttons.down = buttons.down || gamepad.axes[1] > 0.5
    }
    return buttons
  }

  function updateFacingFromButtons(buttons){
    const pressed = !!(buttons && buttons.up && (buttons.s1 || buttons.ba || buttons.BA || buttons.back || buttons.BACK))
    if (!pressed) {
      facingToggleDown = false
      return
    }
    if (facingToggleDown) return
    facingToggleDown = true
    facingRight = !facingRight
    facingStatus.textContent = facingRight ? 'Facing: 右向き (1P)' : 'Facing: 左向き (2P)'
  }

  function start(){
    if (running) return
    saveServerSettings()
    running = true
    setServerSettingsEnabled(false)
    append('started')
    timer = window.setInterval(readGamepads, 16)
    readGamepads()
    updateVisibilityStatus()
  }

  function stop(){
    running = false
    if (timer) window.clearInterval(timer)
    timer = null
    lastPayload = ''
    setServerSettingsEnabled(true)
    append('stopped')
  }

  function setServerSettingsEnabled(enabled){
    customServerInput.disabled = !enabled
    serverHostInput.disabled = !enabled || !customServerInput.checked
  }

  function updateVisibilityStatus(){
    if (!running || !document.hidden) return
    status.textContent = 'Gamepad client is hidden. Browser Gamepad API may pause or miss inputs; keep this page visible.'
  }

  function loadLayout(){
    fetch(appPath('/api/config'))
      .then(response => {
        currentProfile = response.headers.get('X-Config-Profile') || currentProfile
        return response.json()
      })
      .then(config => {
        layoutConfig = normalizeLayoutConfig(config)
        configName.value = currentProfile
        selectedLayout = selectedLayout || 'controller'
        renderLayoutEditor()
        updateLayoutForm()
        layoutStatus.textContent = 'Loaded current overlay layout.'
        loadProfiles()
      })
      .catch(error => {
        layoutConfig = normalizeLayoutConfig({})
        renderLayoutEditor()
        updateLayoutForm()
        layoutStatus.textContent = 'Load failed, showing fallback layout: ' + error
      })
  }

  function loadProfiles(){
    fetch(appPath('/api/config/profiles'))
      .then(response => response.json())
      .then(data => {
        currentProfile = data.current || currentProfile
        const profiles = Array.isArray(data.profiles) ? data.profiles : []
        configProfile.innerHTML = profiles.map(name => {
          return '<option value="' + escapeHTML(name) + '">' + escapeHTML(name) + '</option>'
        }).join('')
        if (profiles.includes(currentProfile)) {
          configProfile.value = currentProfile
        }
        configName.value = currentProfile
      })
      .catch(error => {
        layoutStatus.textContent = 'Profile list failed: ' + error
      })
  }

  function loadProfile(){
    const name = (configName.value || configProfile.value || '').trim()
    if (!name) return
    fetch(appPath('/api/config/profile'), {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name })
    }).then(response => {
      if (!response.ok) throw new Error(response.status)
      return response.json()
    }).then(data => {
      currentProfile = data.current || name
      configName.value = currentProfile
      layoutStatus.textContent = 'Loaded profile: ' + currentProfile
      loadLayout()
    }).catch(error => {
      layoutStatus.textContent = 'Load profile failed: ' + error
    })
  }

  function normalizeLayoutConfig(config){
    if (!config || typeof config !== 'object') config = {}
    if (!config.controller) {
      config.controller = { x: 905, y: 890, width: 190, height: 125, color: '#5f7e91', image: '' }
    }
    if (!config.controller.color) {
      config.controller.color = '#5f7e91'
    }
    if (!config.controller.image) {
      config.controller.image = ''
    }
    if (!config.combo_audio || typeof config.combo_audio !== 'object') {
      config.combo_audio = { volume: 0.7 }
    }
    if (!Number.isFinite(Number(config.combo_audio.volume))) {
      config.combo_audio.volume = 0.7
    }
    if (!Array.isArray(config.buttons) || config.buttons.length === 0) {
      config.buttons = defaultLayoutButtons()
    }
    return config
  }

  function defaultLayoutButtons(){
    return [
      { id: 'left', label: '←', visible: true, x: 930, y: 948, size: 16 },
      { id: 'down', label: '↓', visible: true, x: 950, y: 948, size: 16 },
      { id: 'right', label: '→', visible: true, x: 970, y: 948, size: 16 },
      { id: 'up', label: '↑', visible: true, x: 970, y: 976, size: 18 },
      { id: 'b1', label: 'A', visible: true, x: 1014, y: 948, size: 16 },
      { id: 'b2', label: 'B', visible: true, x: 1034, y: 940, size: 16 },
      { id: 'b3', label: 'X', visible: true, x: 1054, y: 940, size: 16 },
      { id: 'b4', label: 'Y', visible: true, x: 1074, y: 940, size: 16 },
      { id: 'l1', label: 'LB', visible: true, x: 1014, y: 918, size: 16 },
      { id: 'l2', label: 'LT', visible: true, x: 1034, y: 910, size: 16 },
      { id: 'r1', label: 'RB', visible: true, x: 1054, y: 910, size: 16 },
      { id: 'r2', label: 'RT', visible: true, x: 1074, y: 910, size: 16 },
      { id: 's1', label: 'BACK', visible: true, x: 1054, y: 900, size: 10 },
      { id: 's2', label: 'START', visible: true, x: 1076, y: 900, size: 10 },
      { id: 'l3', label: 'LS', visible: true, x: 950, y: 980, size: 10 },
      { id: 'r3', label: 'RS', visible: true, x: 994, y: 980, size: 10 },
      { id: 'a1', label: 'HOME', visible: true, x: 1034, y: 900, size: 10 },
      { id: 'a2', label: 'TURBO', visible: true, x: 916, y: 904, size: 10 }
    ]
  }

  function renderLayoutEditor(){
    if (!layoutConfig) return
    layoutSvg.innerHTML = ''
    layoutSvg.setAttribute('viewBox', layoutViewBox(layoutConfig))
    fillLayoutTargets()

    const bg = svgEl('rect', {
      x: layoutConfig.controller.x,
      y: layoutConfig.controller.y,
      width: layoutConfig.controller.width,
      height: layoutConfig.controller.height,
      rx: 12,
      class: 'layout-bg' + (selectedLayout === 'controller' ? ' selected' : ''),
      fill: layoutConfig.controller.color || '#5f7e91',
      'data-target': 'controller'
    })
    layoutSvg.appendChild(bg)
    if (layoutConfig.controller.image) {
      const image = svgEl('image', {
        x: layoutConfig.controller.x,
        y: layoutConfig.controller.y,
        width: layoutConfig.controller.width,
        height: layoutConfig.controller.height,
        href: assetPath(layoutConfig.controller.image),
        preserveAspectRatio: 'xMidYMid slice',
        opacity: 0.72,
        'data-target': 'controller'
      })
      layoutSvg.appendChild(image)
    }

    layoutConfig.buttons.forEach(button => {
      const centerX = button.x + button.size / 2
      const centerY = button.y + button.size / 2
      const circle = svgEl('circle', {
        cx: centerX,
        cy: centerY,
        r: button.size / 2,
        class: 'layout-button' + (selectedLayout === button.id ? ' selected' : ''),
        fill: button.visible === false ? 'rgba(70,70,70,0.35)' : (button.color || 'rgba(0,0,0,0.22)'),
        opacity: button.visible === false ? 0.45 : 1,
        'data-target': button.id
      })
      const label = svgEl('text', {
        x: centerX,
        y: centerY,
        class: 'layout-label'
      })
      label.textContent = button.label || button.id
      layoutSvg.appendChild(circle)
      layoutSvg.appendChild(label)
    })
  }

  function fillLayoutTargets(){
    const current = layoutTarget.value || selectedLayout
    const options = ['controller'].concat(layoutConfig.buttons.map(button => button.id))
    layoutTarget.innerHTML = options.map(id => {
      const label = id === 'controller' ? 'background' : id
      return '<option value="' + escapeHTML(id) + '">' + escapeHTML(label) + '</option>'
    }).join('')
    if (options.includes(current)) {
      layoutTarget.value = current
    } else {
      layoutTarget.value = 'controller'
      selectedLayout = 'controller'
    }
  }

  function layoutViewBox(config){
    const items = []
    const c = config.controller
    items.push({ x: c.x, y: c.y, width: c.width, height: c.height })
    config.buttons.forEach(button => {
      items.push({ x: button.x, y: button.y, width: button.size, height: button.size })
    })
    const minX = Math.min(...items.map(item => item.x)) - 24
    const minY = Math.min(...items.map(item => item.y)) - 24
    const maxX = Math.max(...items.map(item => item.x + item.width)) + 24
    const maxY = Math.max(...items.map(item => item.y + item.height)) + 24
    return [minX, minY, Math.max(80, maxX - minX), Math.max(80, maxY - minY)].join(' ')
  }

  function selectedButton(){
    if (!layoutConfig || selectedLayout === 'controller') return null
    return layoutConfig.buttons.find(button => button.id === selectedLayout) || null
  }

  function updateLayoutForm(){
    if (!layoutConfig) return
    const button = selectedButton()
    const target = button || layoutConfig.controller
    layoutTarget.value = selectedLayout
    layoutInputs.visible.disabled = !button
    layoutInputs.visible.checked = button ? button.visible !== false : true
    layoutInputs.label.disabled = !button
    layoutInputs.label.value = button ? (button.label || '') : ''
    layoutInputs.historyLabel.disabled = !button
    layoutInputs.historyLabel.value = button ? (button.history_label || '') : ''
    layoutInputs.historyColor.disabled = !button
    layoutInputs.historyColor.value = toColorInput(button ? (button.history_color || button.color) : '', '#ffffff')
    layoutInputs.color.disabled = !button
    layoutInputs.color.value = toColorInput(button ? button.color : '', '#000000')
    layoutInputs.bgColor.disabled = !!button
    layoutInputs.bgColor.value = toColorInput(layoutConfig.controller.color, '#5f7e91')
    layoutInputs.bgImage.disabled = !!button
    layoutInputs.bgImage.value = button ? '' : (layoutConfig.controller.image || '')
    layoutInputs.bgUpload.disabled = !!button
    const volume = Math.max(0, Math.min(1, Number(layoutConfig.combo_audio && layoutConfig.combo_audio.volume) || 0.7))
    layoutInputs.comboAudioVolume.value = Math.round(volume * 100)
    layoutInputs.comboAudioVolumeLabel.textContent = 'Volume: ' + Math.round(volume * 100) + '%'
    layoutInputs.x.value = target.x
    layoutInputs.y.value = target.y
    layoutInputs.size.disabled = !button
    layoutInputs.width.disabled = !!button
    layoutInputs.height.disabled = !!button
    layoutInputs.size.value = button ? button.size : ''
    layoutInputs.width.value = button ? '' : target.width
    layoutInputs.height.value = button ? '' : target.height
  }

  function applyLayoutForm(){
    if (!layoutConfig) return
    const button = selectedButton()
    const target = button || layoutConfig.controller
    target.x = numberValue(layoutInputs.x, target.x)
    target.y = numberValue(layoutInputs.y, target.y)
    if (button) {
      button.visible = layoutInputs.visible.checked
      button.label = layoutInputs.label.value
      button.history_label = layoutInputs.historyLabel.value.slice(0, 2)
      button.history_color = layoutInputs.historyColor.value
      button.color = layoutInputs.color.value
      target.size = Math.max(1, numberValue(layoutInputs.size, target.size))
    } else {
      layoutConfig.controller.color = layoutInputs.bgColor.value
      layoutConfig.controller.image = layoutInputs.bgImage.value.trim()
      target.width = Math.max(1, numberValue(layoutInputs.width, target.width))
      target.height = Math.max(1, numberValue(layoutInputs.height, target.height))
    }
    layoutConfig.combo_audio = layoutConfig.combo_audio || {}
    layoutConfig.combo_audio.volume = Math.max(0, Math.min(100, numberValue(layoutInputs.comboAudioVolume, 70))) / 100
    layoutInputs.comboAudioVolumeLabel.textContent = 'Volume: ' + Math.round(layoutConfig.combo_audio.volume * 100) + '%'
    renderLayoutEditor()
    updateLayoutForm()
  }

  function numberValue(input, fallback){
    const value = Number(input.value)
    return Number.isFinite(value) ? Math.round(value) : fallback
  }

  function toColorInput(value, fallback){
    if (!value) return fallback
    const trimmed = String(value).trim()
    if (/^#[0-9a-fA-F]{6}$/.test(trimmed)) return trimmed
    const match = trimmed.match(/^rgba?\((\d+),\s*(\d+),\s*(\d+)/)
    if (!match) return fallback
    return '#' + [match[1], match[2], match[3]].map(part => {
      return Math.max(0, Math.min(255, Number(part))).toString(16).padStart(2, '0')
    }).join('')
  }

  function saveLayout(profile){
    if (!layoutConfig) return
    const query = profile ? '?profile=' + encodeURIComponent(profile) : ''
    fetch(appPath('/api/config') + query, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(layoutConfig)
    }).then(response => {
      if (!response.ok) throw new Error(response.status)
      if (profile) {
        currentProfile = withJsonExtension(profile)
        configName.value = currentProfile
      }
      layoutStatus.textContent = 'Saved: ' + currentProfile
      append('layout saved: ' + currentProfile)
      loadProfiles()
    }).catch(error => {
      layoutStatus.textContent = 'Save failed: ' + error
    })
  }

  function withJsonExtension(name){
    const trimmed = String(name || '').trim()
    if (!trimmed) return currentProfile
    return /\.json$/i.test(trimmed) ? trimmed : trimmed + '.json'
  }

  function uploadBackgroundImage(){
    const file = layoutInputs.bgUpload.files && layoutInputs.bgUpload.files[0]
    if (!file || !layoutConfig) return
    const body = new FormData()
    body.append('image', file)
    fetch(appPath('/api/background/upload'), {
      method: 'POST',
      body
    }).then(response => {
      if (!response.ok) throw new Error(response.status)
      return response.json()
    }).then(result => {
      layoutConfig.controller.image = result.path || ''
      renderLayoutEditor()
      updateLayoutForm()
      layoutStatus.textContent = 'Uploaded background image. Press Save to keep it.'
    }).catch(error => {
      layoutStatus.textContent = 'Upload failed: ' + error
    })
  }

  function loadCombos(){
    fetch(appPath('/api/combos'))
      .then(response => {
        if (!response.ok) throw new Error(response.status)
        return response.json()
      })
      .then(data => applyCombos(data))
      .catch(error => {
        comboStatus.textContent = 'Combo load failed: ' + error
        append('combo load failed: ' + error)
      })
  }

  function applyCombos(data){
    comboLoading = true
    comboState = data || null
    const files = comboState && Array.isArray(comboState.files) ? comboState.files : []
    comboFile.innerHTML = files.map(file => optionHTML(file.file, file.title || file.file)).join('')
    if (files.length === 0) {
      comboSet.innerHTML = ''
      comboRecipe.innerHTML = ''
      comboPracticeSet.innerHTML = ''
      comboStatus.textContent = 'No combo YAML loaded.'
      comboLoading = false
      return
    }
    comboFile.value = (comboState.current && comboState.current.file) || files[0].file
    fillComboSets()
    if (comboState.current && comboState.current.set_id) comboSet.value = comboState.current.set_id
    fillPracticeControls()
    comboLoading = false
    updateComboStatus()
  }

  function fillComboSets(){
    const file = selectedComboFile()
    comboSet.innerHTML = file ? file.sets.map(set => optionHTML(set.id, set.name || set.id)).join('') : ''
    if (file && comboState.current && comboState.current.file === file.file) {
      comboSet.value = comboState.current.set_id || comboSet.value
    }
  }

  function fillPracticeControls(){
    const active = (comboState && comboState.activePractice) || {}
    comboMode.value = active.mode || 'focus'
    const recipes = comboRecipeOptions()
    comboRecipe.innerHTML = recipes.map(recipe => optionHTML(recipe.id, recipe.name || recipe.id)).join('')
    if (recipes.some(recipe => recipe.id === active.activeRecipeId)) {
      comboRecipe.value = active.activeRecipeId
    }
    const sets = comboState && Array.isArray(comboState.practiceSets) ? comboState.practiceSets : []
    comboPracticeSet.innerHTML = sets.map(set => optionHTML(set.id, set.name || set.id)).join('')
    if (sets.some(set => set.id === active.activeSetId)) {
      comboPracticeSet.value = active.activeSetId
    }
    applySelectedPracticeSet()
  }

  function activateSelectedCombo(){
    if (comboLoading) return
    const file = comboFile.value
    const setId = comboSet.value
    if (!file || !setId) return
    const body = selectedComboPayload(file, setId)
    comboStatus.textContent = 'Changing combo...'
    fetch(appPath('/api/combos/active'), {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body)
    }).then(response => {
      if (!response.ok) throw new Error(response.status)
      return response.json()
    }).then(() => loadCombos()).catch(error => {
      comboStatus.textContent = 'Combo activate failed: ' + error
      append('combo activate failed: ' + error)
    })
  }

  function selectedComboPayload(file, setId){
    const payload = {
      file,
      set_id: setId,
      mode: comboMode.value || 'focus',
      activeRecipe: comboRecipe.value || '',
      activeSet: comboPracticeSet.value || '',
      activeSetIndex: 0,
      loop: comboLoop.checked,
      advanceOnComplete: comboAdvance.checked
    }
    const practiceSet = selectedPracticeSet()
    if (practiceSet && Array.isArray(practiceSet.recipes)) {
      const index = practiceSet.recipes.indexOf(payload.activeRecipe)
      payload.activeSetIndex = index >= 0 ? index : 0
    }
    return payload
  }

  function uploadCombo(){
    const file = comboUpload.files && comboUpload.files[0]
    if (!file) return
    const body = new FormData()
    body.append('combo', file)
    comboStatus.textContent = 'Uploading combo...'
    fetch(appPath('/api/combos/upload'), {
      method: 'POST',
      body
    }).then(response => {
      if (!response.ok) throw new Error(response.status)
      return response.json()
    }).then(() => {
      comboUpload.value = ''
      loadCombos()
    }).catch(error => {
      comboStatus.textContent = 'Combo upload failed: ' + error
      append('combo upload failed: ' + error)
    })
  }

  function updateComboStatus(){
    const file = selectedComboFile()
    const set = selectedComboSet()
    if (!file || !set) {
      comboStatus.textContent = 'Select combo file and set.'
      return
    }
    comboStatus.textContent = (set.name || set.id) + ' (' + (set.mode || 'command') + ')'
  }

  function selectedComboFile(){
    const files = comboState && Array.isArray(comboState.files) ? comboState.files : []
    return files.find(file => file.file === comboFile.value) || null
  }

  function selectedComboSet(){
    const file = selectedComboFile()
    const sets = file && Array.isArray(file.sets) ? file.sets : []
    return sets.find(set => set.id === comboSet.value) || null
  }

  function selectedPracticeSet(){
    const sets = comboState && Array.isArray(comboState.practiceSets) ? comboState.practiceSets : []
    return sets.find(set => set.id === comboPracticeSet.value) || null
  }

  function applySelectedPracticeSet(){
    const set = selectedPracticeSet()
    if (!set) {
      comboLoop.checked = false
      comboAdvance.checked = false
      return
    }
    comboLoop.checked = !!set.loop
    comboAdvance.checked = !!set.advanceOnComplete
    if (Array.isArray(set.recipes) && set.recipes.length > 0 && !set.recipes.includes(comboRecipe.value)) {
      comboRecipe.value = set.recipes[0]
    }
  }

  function comboRecipeOptions(){
    const set = comboState && comboState.active_set
    return set && Array.isArray(set.combos) ? set.combos : []
  }

  function optionHTML(value, label){
    return '<option value="' + escapeHTML(value) + '">' + escapeHTML(label) + '</option>'
  }

  function svgPoint(event){
    const point = layoutSvg.createSVGPoint()
    point.x = event.clientX
    point.y = event.clientY
    return point.matrixTransform(layoutSvg.getScreenCTM().inverse())
  }

  function svgEl(name, attrs){
    const el = document.createElementNS('http://www.w3.org/2000/svg', name)
    Object.keys(attrs).forEach(key => el.setAttribute(key, attrs[key]))
    return el
  }

  document.getElementById('start').addEventListener('click', event => {
    event.preventDefault()
    start()
  })
  document.getElementById('stop').addEventListener('click', event => {
    event.preventDefault()
    stop()
  })
  document.getElementById('scan').addEventListener('click', event => {
    event.preventDefault()
    renderGamepads()
    const gamepad = findGamepad()
    append(gamepad ? 'scan found: ' + gamepad.id : 'scan found no gamepads')
  })
  customServerInput.addEventListener('change', () => {
    syncServerControls()
    saveServerSettings()
  })
  serverHostInput.addEventListener('change', () => {
    serverHostInput.value = normalizeServerHost(serverHostInput.value)
    saveServerSettings()
  })
  serverHostInput.addEventListener('input', saveServerSettings)
  document.getElementById('hid').addEventListener('click', async event => {
    event.preventDefault()
    if (!navigator.hid) {
      hidStatus.textContent = 'WebHID is not available in this browser.'
      append('WebHID unavailable')
      return
    }
    try {
      const devices = await navigator.hid.requestDevice({ filters: [] })
      if (devices.length === 0) {
        hidStatus.textContent = 'No HID device selected.'
        append('WebHID selected no devices')
        return
      }
      hidStatus.innerHTML = devices.map(device => {
        return escapeHTML(device.productName || '(no product name)') +
          ' vendor=0x' + device.vendorId.toString(16) +
          ' product=0x' + device.productId.toString(16)
      }).join('<br>')
      append('WebHID selected: ' + devices.map(device => device.productName || 'unknown').join(', '))
    } catch (error) {
      hidStatus.textContent = 'WebHID error: ' + error
      append('WebHID error: ' + error)
    }
  })
  configProfile.addEventListener('change', () => {
    configName.value = configProfile.value
  })
  document.getElementById('config-load').addEventListener('click', event => {
    event.preventDefault()
    loadProfile()
  })
  document.getElementById('config-save-as').addEventListener('click', event => {
    event.preventDefault()
    saveLayout(configName.value)
  })
  layoutTarget.addEventListener('change', () => {
    selectedLayout = layoutTarget.value
    renderLayoutEditor()
    updateLayoutForm()
  })
  Object.values(layoutInputs).forEach(input => {
    input.addEventListener('change', applyLayoutForm)
    input.addEventListener('input', applyLayoutForm)
  })
  layoutInputs.bgUpload.addEventListener('change', uploadBackgroundImage)
  document.getElementById('layout-reload').addEventListener('click', event => {
    event.preventDefault()
    loadLayout()
  })
  document.getElementById('layout-save').addEventListener('click', event => {
    event.preventDefault()
    saveLayout()
  })
  comboFile.addEventListener('change', () => {
    fillComboSets()
    activateSelectedCombo()
  })
  comboSet.addEventListener('change', activateSelectedCombo)
  comboMode.addEventListener('change', activateSelectedCombo)
  comboRecipe.addEventListener('change', activateSelectedCombo)
  comboPracticeSet.addEventListener('change', () => {
    applySelectedPracticeSet()
    activateSelectedCombo()
  })
  comboLoop.addEventListener('change', activateSelectedCombo)
  comboAdvance.addEventListener('change', activateSelectedCombo)
  comboUpload.addEventListener('change', uploadCombo)
  document.getElementById('combo-reload').addEventListener('click', event => {
    event.preventDefault()
    loadCombos()
  })
  layoutSvg.addEventListener('pointerdown', event => {
    const targetId = event.target.dataset && event.target.dataset.target
    if (!targetId || !layoutConfig) return
    selectedLayout = targetId
    const point = svgPoint(event)
    const button = selectedButton()
    const target = button || layoutConfig.controller
    draggingLayout = {
      pointerId: event.pointerId,
      startX: point.x,
      startY: point.y,
      targetX: target.x,
      targetY: target.y
    }
    layoutSvg.setPointerCapture(event.pointerId)
    renderLayoutEditor()
    updateLayoutForm()
  })
  layoutSvg.addEventListener('pointermove', event => {
    if (!draggingLayout || draggingLayout.pointerId !== event.pointerId) return
    const point = svgPoint(event)
    const dx = Math.round(point.x - draggingLayout.startX)
    const dy = Math.round(point.y - draggingLayout.startY)
    const button = selectedButton()
    const target = button || layoutConfig.controller
    target.x = draggingLayout.targetX + dx
    target.y = draggingLayout.targetY + dy
    renderLayoutEditor()
    updateLayoutForm()
  })
  layoutSvg.addEventListener('pointerup', event => {
    if (draggingLayout && draggingLayout.pointerId === event.pointerId) {
      draggingLayout = null
      layoutSvg.releasePointerCapture(event.pointerId)
    }
  })
  layoutSvg.addEventListener('pointercancel', () => {
    draggingLayout = null
  })

  window.addEventListener('gamepadconnected', event => {
    status.textContent = 'Gamepad connected: ' + event.gamepad.id
    append('connected: ' + event.gamepad.id)
  })
  window.addEventListener('gamepaddisconnected', event => {
    status.textContent = 'Gamepad disconnected: ' + event.gamepad.id
    append('disconnected: ' + event.gamepad.id)
  })
  window.addEventListener('focus', renderGamepads)
  document.addEventListener('visibilitychange', updateVisibilityStatus)
  window.setInterval(renderGamepads, 1000)
  renderGamepads()
  loadLayout()
  loadCombos()

  function escapeHTML(value){
    return String(value).replace(/[&<>"']/g, ch => ({
      '&': '&amp;',
      '<': '&lt;',
      '>': '&gt;',
      '"': '&quot;',
      "'": '&#39;'
    }[ch]))
  }
})();
