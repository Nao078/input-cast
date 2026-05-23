(() => {
  const overlay = document.getElementById('overlay')
  let buttons = {}
  let currentState = { buttons: {} }
  let currentConfig = null
  let historyList = null
  let historyEntries = []
  let activeHistory = null
  let rafId = null
  const maxFrameCount = 99
  const directionOrder = ['up', 'down', 'left', 'right']
  const directionLabels = { up: '↑', down: '↓', left: '←', right: '→' }
  const historyDirections = {
    up: { label: '↑', x: 1, y: 0 },
    down: { label: '↓', x: 1, y: 2 },
    left: { label: '←', x: 0, y: 1 },
    right: { label: '→', x: 2, y: 1 },
    'up-left': { label: '↖', x: 0, y: 0 },
    'up-right': { label: '↗', x: 2, y: 0 },
    'down-left': { label: '↙', x: 0, y: 2 },
    'down-right': { label: '↘', x: 2, y: 2 }
  }
  const buttonOrder = ['b1', 'b2', 'b3', 'b4', 'l1', 'l2', 'r1', 'r2', 's1', 's2', 'l3', 'r3', 'a1', 'a2']
  const basePath = detectBasePath()

  function detectBasePath(){
    const path = location.pathname.replace(/\/+$/, '')
    return path.replace(/\/(?:overlay|preview)$/, '')
  }

  function appPath(path){
    return (basePath || '') + path
  }

  function assetPath(path){
    const value = String(path || '')
    if (!value || /^(?:[a-z]+:)?\/\//i.test(value) || /^(?:data|blob):/i.test(value)) return value
    if (value.startsWith('/')) return appPath(value)
    return value
  }

  function fetchConfig(){
    return fetch(appPath('/api/config')).then(r=>r.json())
  }

  function buildButtons(cfg){
    currentConfig = cfg
    overlay.innerHTML = ''
    buttons = {}
    historyList = null
    if (cfg.overlay && cfg.overlay.width && cfg.overlay.height) {
      overlay.style.width = cfg.overlay.width + 'px'
      overlay.style.height = cfg.overlay.height + 'px'
    }
    const controller = document.createElement('div')
    controller.className = 'controller'
    overlay.appendChild(controller)
    buildControllerBackground(cfg, controller)
    buildHistory(cfg)
    cfg.buttons.forEach(b=>{
      if (b.visible === false) return
      const el = document.createElement('div')
      el.className = 'button'
      el.id = 'btn-'+b.id
      el.dataset.id = b.id
      el.textContent = b.label || b.id
      el.style.left = b.x+'px'
      el.style.top = b.y+'px'
      el.style.width = b.size+'px'
      el.style.height = b.size+'px'
      el.style.setProperty('--button-size', b.size+'px')
      if (b.color) el.style.setProperty('--button-color', b.color)
      if (b.pressed_color) el.style.setProperty('--button-pressed-color', b.pressed_color)
      if (b.text_color) el.style.setProperty('--button-text-color', b.text_color)
      if (b.opacity) el.style.opacity = String(b.opacity)
      controller.appendChild(el)
      buttons[b.id] = el
    })
    applyState(currentState)
    renderHistory()
    fitOverlay()
  }

  function buildControllerBackground(cfg, controller){
    const c = cfg.controller || { x: 905, y: 890, width: 190, height: 125 }
    const bg = document.createElement('div')
    bg.className = 'controller-bg'
    bg.style.left = c.x + 'px'
    bg.style.top = c.y + 'px'
    bg.style.width = c.width + 'px'
    bg.style.height = c.height + 'px'
    if (c.color) bg.style.background = c.color
    if (c.image) {
      bg.style.backgroundImage = 'url("' + assetPath(c.image).replace(/"/g, '\\"') + '")'
      bg.style.backgroundSize = 'cover'
      bg.style.backgroundPosition = 'center'
    }
    controller.appendChild(bg)
  }

  function buildHistory(cfg){
    const enabled = !cfg.history || cfg.history.enabled !== false
    if (!enabled) return
    const history = document.createElement('section')
    history.className = 'history'
    const h = cfg.history || {}
    if (h.show_border === false) {
      history.classList.add('history-frameless')
    }
    history.style.left = (h.x || 40) + 'px'
    history.style.top = (h.y || 40) + 'px'
    history.style.width = (h.width || 250) + 'px'
    history.style.height = (h.height || 1000) + 'px'
    history.innerHTML = '<div class="history-list"></div>'
    overlay.appendChild(history)
    historyList = history.querySelector('.history-list')
  }

  function fitOverlay(){
    const cfg = currentConfig
    if (!cfg || !cfg.overlay || !cfg.overlay.width || !cfg.overlay.height) return
    const scaleX = window.innerWidth / cfg.overlay.width
    const scaleY = window.innerHeight / cfg.overlay.height
    const scale = Math.min(scaleX, scaleY)
    overlay.style.transform = 'scale(' + scale + ')'
  }

  function applyState(state){
    if(!state || !state.buttons) return
    const incomingKeys = Object.keys(state.buttons)
    const totalButtons = Object.keys(buttons).length
    // if incoming contains full set, treat as full update
    if(incomingKeys.length === totalButtons){
      Object.keys(buttons).forEach(id=>{
        const el = buttons[id]
        const pressed = !!state.buttons[id]
        el.classList.toggle('pressed', pressed)
      })
    } else {
      // partial update: only change specified keys
      incomingKeys.forEach(id=>{
        const el = buttons[id]
        if(!el) return
        const pressed = !!state.buttons[id]
        el.classList.toggle('pressed', pressed)
      })
    }
  }

  function updateHistoryFromState(state){
    const now = performance.now()
    const key = stateKey(state.buttons || {})
    if (!activeHistory) {
      activeHistory = { key, buttons: cloneButtons(state.buttons || {}), startedAt: now, frames: 1 }
      renderHistory()
      return
    }
    if (activeHistory.key === key) {
      activeHistory.frames = frameCount(activeHistory.startedAt, now)
      renderHistory()
      return
    }
    activeHistory.frames = frameCount(activeHistory.startedAt, now)
    historyEntries.unshift({
      frames: activeHistory.frames,
      buttons: cloneButtons(activeHistory.buttons)
    })
    trimHistory()
    activeHistory = { key, buttons: cloneButtons(state.buttons || {}), startedAt: now, frames: 1 }
    renderHistory()
  }

  function frameCount(startedAt, now){
    const frames = Math.max(1, Math.floor(((now - startedAt) / 1000) * 60) + 1)
    return Math.min(maxFrameCount, frames)
  }

  function trimHistory(){
    const maxEntries = (currentConfig && currentConfig.history && currentConfig.history.max_entries) || 24
    historyEntries = historyEntries.slice(0, maxEntries)
  }

  function renderHistory(){
    if (!historyList) return
    const rows = []
    if (activeHistory) {
      rows.push({ frames: activeHistory.frames || 1, buttons: activeHistory.buttons, live: true })
    }
    rows.push(...historyEntries)
    historyList.innerHTML = rows.map(entry => {
      const parts = formatHistory(entry.buttons)
      return [
        '<div class="history-row' + (entry.live ? ' live' : '') + '">',
        '<span class="history-frame">' + escapeHTML(String(entry.frames)) + '</span>',
        '<span class="history-input">',
        historyDirectionMarkup(parts.direction),
        historyButtonMarkup(parts),
        '</span>',
        '</div>'
      ].join('')
    }).join('')
  }

  function historyDirectionMarkup(direction){
    if (!direction) return ''
    const label = escapeHTML(direction.label || '')
    if (direction.x === undefined || direction.y === undefined) {
      return '<span class="history-token history-direction history-direction-text">' + label + '</span>'
    }
    const style = '--direction-bg-x:-' + (Number(direction.x || 0) * 28) + 'px;--direction-bg-y:-' + (Number(direction.y || 0) * 28) + 'px'
    return '<span class="history-token history-direction" style="' + style + '" aria-label="' + label + '" title="' + label + '"></span>'
  }

  function historyButtonMarkup(parts){
    if (parts.buttons.length > 0) {
      return parts.buttons.map(button => {
        const label = String(button.label || '')
        const lengthClass = label.length > 1 ? ' history-button-compact' : ''
        return '<span class="history-token history-button' + lengthClass + '" style="background:' + escapeAttribute(button.color) + '">' + escapeHTML(label) + '</span>'
      }).join('')
    }
    if (!parts.direction) {
      return '<span class="history-token history-neutral">N</span>'
    }
    return ''
  }

  function formatHistory(buttonsState){
    const direction = formatDirections(buttonsState)
    const buttons = buttonOrder
      .filter(id => buttonsState[id])
      .map(id => buttonHistoryToken(id))
    return { direction, buttons }
  }

  function formatDirections(buttonsState){
    const up = !!buttonsState.up
    const down = !!buttonsState.down
    const left = !!buttonsState.left
    const right = !!buttonsState.right
    if (down && right && !up && !left) return historyDirections['down-right']
    if (down && left && !up && !right) return historyDirections['down-left']
    if (up && right && !down && !left) return historyDirections['up-right']
    if (up && left && !down && !right) return historyDirections['up-left']
    const directions = directionOrder
      .filter(id => buttonsState[id])
    if (directions.length === 1) return historyDirections[directions[0]]
    const label = directions.map(id => directionLabels[id]).join('')
    return label ? { label } : null
  }

  function labelFor(id){
    if (!currentConfig || !Array.isArray(currentConfig.buttons)) return id.toUpperCase()
    const def = currentConfig.buttons.find(b => b.id === id)
    return (def && def.label) || id.toUpperCase()
  }

  function buttonHistoryToken(id){
    const def = buttonDefFor(id)
    const label = ((def && def.history_label) || labelFor(id) || id).slice(0, 2)
    const color = (def && (def.history_color || def.color)) || 'rgba(255,255,255,0.24)'
    return { label, color }
  }

  function buttonDefFor(id){
    if (!currentConfig || !Array.isArray(currentConfig.buttons)) return null
    return currentConfig.buttons.find(b => b.id === id) || null
  }

  function stateKey(buttonsState){
    const ids = [...directionOrder, ...buttonOrder]
    return ids.map(id => buttonsState[id] ? '1' : '0').join('')
  }

  function cloneButtons(buttonsState){
    return Object.assign({}, buttonsState)
  }

  function hasAnyInput(buttonsState){
    return Object.keys(buttonsState).some(id => buttonsState[id])
  }

  function escapeHTML(value){
    return String(value).replace(/[&<>"']/g, ch => ({
      '&': '&amp;',
      '<': '&lt;',
      '>': '&gt;',
      '"': '&quot;',
      "'": '&#39;'
    }[ch]))
  }

  function escapeAttribute(value){
    return escapeHTML(String(value || ''))
  }

  function connect(){
    const proto = (location.protocol === 'https:') ? 'wss' : 'ws'
    const ws = new WebSocket(proto + '://' + location.host + appPath('/ws'))
    ws.onopen = () => {
      // ensure we have latest config on reconnect
      fetchConfig().then(cfg=>{ buildButtons(cfg) }).catch(()=>{})
    }
    ws.onmessage = e => {
      try{
        const data = JSON.parse(e.data)
        if (data.type === 'config' && data.config) {
          buildButtons(data.config)
        } else if (data.type === 'input') {
          if(!currentState.buttons) currentState.buttons = {}
          if (isFullInputState(data.buttons || {})) {
            currentState = { buttons: cloneButtons(data.buttons || {}) }
          } else {
            Object.assign(currentState.buttons, data.buttons || {})
          }
          applyState(currentState)
          updateHistoryFromState(currentState)
        } else {
          if(data.buttons){
            currentState = { buttons: data.buttons }
            applyState(currentState)
            updateHistoryFromState(currentState)
          }
        }
      }catch(e){}
    }
    ws.onclose = ()=>{setTimeout(connect,1000)}
  }

  function isFullInputState(buttonsState){
    const ids = [...directionOrder, ...buttonOrder]
    return ids.every(id => Object.prototype.hasOwnProperty.call(buttonsState, id))
  }

  // initial load
  fetchConfig().then(cfg=>{ buildButtons(cfg); connect() }).catch(e=>{ console.error(e); connect() })
  window.addEventListener('resize', fitOverlay)
  startHistoryTicker()

  function startHistoryTicker(){
    if (rafId) return
    const tick = () => {
      if (activeHistory) {
        const nextFrames = frameCount(activeHistory.startedAt, performance.now())
        if (nextFrames !== activeHistory.frames) {
          activeHistory.frames = nextFrames
          renderHistory()
        }
      }
      rafId = window.requestAnimationFrame(tick)
    }
    rafId = window.requestAnimationFrame(tick)
  }

})();
