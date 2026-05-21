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
  const buttonOrder = ['b1', 'b2', 'b3', 'b4', 'l1', 'l2', 'r1', 'r2', 's1', 's2', 'l3', 'r3', 'a1', 'a2']

  function fetchConfig(){
    return fetch('/api/config').then(r=>r.json())
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
      bg.style.backgroundImage = 'url("' + String(c.image).replace(/"/g, '\\"') + '")'
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
        parts.direction ? '<span class="history-token history-direction">' + escapeHTML(parts.direction) + '</span>' : '',
        historyButtonMarkup(parts),
        '</span>',
        '</div>'
      ].join('')
    }).join('')
  }

  function historyButtonMarkup(parts){
    if (parts.buttons.length > 0) {
      return parts.buttons.map(button => {
        return '<span class="history-token history-button" style="background:' + escapeAttribute(button.color) + '">' + escapeHTML(button.label) + '</span>'
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
    if (down && right) return '↘'
    if (down && left) return '↙'
    if (up && right) return '↗'
    if (up && left) return '↖'
    return directionOrder
      .filter(id => buttonsState[id])
      .map(id => directionLabels[id])
      .join('')
  }

  function labelFor(id){
    if (!currentConfig || !Array.isArray(currentConfig.buttons)) return id.toUpperCase()
    const def = currentConfig.buttons.find(b => b.id === id)
    return (def && def.label) || id.toUpperCase()
  }

  function buttonHistoryToken(id){
    const def = buttonDefFor(id)
    const label = (def && def.history_label) || labelFor(id).slice(0, 1) || id.slice(0, 1).toUpperCase()
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
    const ws = new WebSocket(proto + '://' + location.host + '/ws')
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
