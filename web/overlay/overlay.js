(() => {
  const overlay = document.getElementById('overlay')
  let buttons = {}
  let currentState = { buttons: {} }
  let currentConfig = null
  let historyList = null
  let historyMenu = null
  let historyEntries = []
  let copyHistoryEntries = []
  let activeHistory = null
  let historyClearButtonDown = false
  let comboPanel = null
  let comboConfig = null
  let comboProgress = []
  let activePractice = null
  let successFlash = null
  let facing = 'right'
  let facingToggleChordDown = false
  let audioContext = null
  let audioUnlocked = false
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

  function fetchComboConfig(){
    return fetch(appPath('/api/combos')).then(r=>r.json())
  }

  function buildButtons(cfg){
    currentConfig = cfg
    overlay.innerHTML = ''
    buttons = {}
    historyList = null
    comboPanel = null
    if (cfg.overlay && cfg.overlay.width && cfg.overlay.height) {
      overlay.style.width = cfg.overlay.width + 'px'
      overlay.style.height = cfg.overlay.height + 'px'
    }
    const controller = document.createElement('div')
    controller.className = 'controller'
    overlay.appendChild(controller)
    buildControllerBackground(cfg, controller)
    buildHistory(cfg)
    buildComboPanel(cfg)
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
    renderCombos()
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
    history.innerHTML = '<div class=history-list></div>'
    history.addEventListener('click', event => {
      event.stopPropagation()
      showHistoryMenu(event.clientX, event.clientY)
    })
    overlay.appendChild(history)
    historyList = history.querySelector('.history-list')
    ensureHistoryMenu()
  }

  function ensureHistoryMenu(){
    if (historyMenu) return historyMenu
    historyMenu = document.createElement('div')
    historyMenu.className = 'history-menu'
    historyMenu.hidden = true
    historyMenu.innerHTML = [
      '<button type=button class=history-menu-item data-action=copy>コピー</button>',
      '<button type=button class=history-menu-item data-action=clear>履歴をクリア</button>'
    ].join('')
    historyMenu.addEventListener('click', event => {
      event.stopPropagation()
      const action = event.target && event.target.dataset && event.target.dataset.action
      if (action === 'copy') {
        copyHistoryMarkdown()
      } else if (action === 'clear') {
        clearHistory()
      }
      hideHistoryMenu()
    })
    document.body.appendChild(historyMenu)
    return historyMenu
  }

  function showHistoryMenu(clientX, clientY){
    const menu = ensureHistoryMenu()
    menu.hidden = false
    menu.style.left = '0px'
    menu.style.top = '0px'
    const rect = menu.getBoundingClientRect ? menu.getBoundingClientRect() : { width: 0, height: 0 }
    const margin = 8
    const viewportWidth = window.innerWidth || 0
    const viewportHeight = window.innerHeight || 0
    const maxX = viewportWidth > 0 ? Math.max(margin, viewportWidth - rect.width - margin) : clientX
    const maxY = viewportHeight > 0 ? Math.max(margin, viewportHeight - rect.height - margin) : clientY
    const x = Math.min(Math.max(margin, clientX), maxX)
    const y = Math.min(Math.max(margin, clientY), maxY)
    menu.style.left = x + 'px'
    menu.style.top = y + 'px'
  }

  function hideHistoryMenu(){
    if (historyMenu) historyMenu.hidden = true
  }

  function clearHistory(){
    activeHistory = null
    historyEntries = []
    copyHistoryEntries = []
    renderHistory()
  }

  function copyHistoryMarkdown(){
    const markdown = historyMarkdownTable()
    return writeClipboardText(markdown)
  }

  function writeClipboardText(value){
    if (typeof navigator !== 'undefined' && navigator.clipboard && navigator.clipboard.writeText) {
      return navigator.clipboard.writeText(value).catch(() => fallbackWriteClipboardText(value))
    }
    return fallbackWriteClipboardText(value)
  }

  function fallbackWriteClipboardText(value){
    const textarea = document.createElement('textarea')
    textarea.value = value
    textarea.setAttribute('readonly', '')
    textarea.style.position = 'fixed'
    textarea.style.left = '-9999px'
    document.body.appendChild(textarea)
    textarea.select()
    try {
      if (document.execCommand) document.execCommand('copy')
    } finally {
      document.body.removeChild(textarea)
    }
    return Promise.resolve()
  }

  function buildComboPanel(cfg){
    const h = cfg.combo_display || {}
    const hasLayout = !!(h.x || h.y || h.width || h.height)
    const enabled = !cfg.combo_display || h.enabled !== false || !hasLayout
    if (!enabled) return
    const history = cfg.history || {}
    const combo = document.createElement('section')
    combo.className = 'combo-display'
    if (h.show_border === false) {
      combo.classList.add('combo-frameless')
    }
    const defaultX = (history.x || 40) + (history.width || 250) + 24
    combo.style.left = (h.x || defaultX) + 'px'
    combo.style.top = (h.y || history.y || 40) + 'px'
    combo.style.width = (h.width || 420) + 'px'
    combo.style.height = (h.height || 520) + 'px'
    combo.innerHTML = '<div class="combo-header"></div><div class="combo-list"></div><div class="combo-success" aria-hidden="true">SUCCESS</div>'
    overlay.appendChild(combo)
    comboPanel = combo
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
    if (consumeHistoryClearButton(state.buttons || {})) return
    const now = performance.now()
    const key = stateKey(state.buttons || {})
    if (!activeHistory) {
      activeHistory = { key, buttons: cloneButtons(state.buttons || {}), startedAt: now, frames: 1 }
      processComboInput(activeHistory.buttons, 1)
      renderHistory()
      return
    }
    if (activeHistory.key === key) {
      activeHistory.frames = frameCount(activeHistory.startedAt, now)
      renderHistory()
      return
    }
    activeHistory.frames = frameCount(activeHistory.startedAt, now)
    const elapsedFrames = activeHistory.frames
    const completedEntry = {
      frames: activeHistory.frames,
      buttons: cloneButtons(activeHistory.buttons)
    }
    historyEntries.unshift(completedEntry)
    copyHistoryEntries.push(cloneHistoryEntry(completedEntry))
    trimHistory()
    activeHistory = { key, buttons: cloneButtons(state.buttons || {}), startedAt: now, frames: 1 }
    processComboInput(activeHistory.buttons, elapsedFrames)
    renderHistory()
  }

  function consumeHistoryClearButton(buttonsState){
    const clearPressed = !!buttonsState.a1
    if (!clearPressed) {
      historyClearButtonDown = false
      return false
    }
    if (!historyClearButtonDown) clearHistory()
    historyClearButtonDown = true
    return true
  }

  function frameCount(startedAt, now){
    const frames = Math.max(1, Math.floor(((now - startedAt) / 1000) * 60) + 1)
    return Math.min(maxFrameCount, frames)
  }

  function trimHistory(){
    const maxEntries = (currentConfig && currentConfig.history && currentConfig.history.max_entries) || 24
    historyEntries = historyEntries.slice(0, maxEntries)
    copyHistoryEntries = copyHistoryEntries.slice(-copyHistoryMaxEntries())
  }

  function copyHistoryMaxEntries(){
    const value = currentConfig && currentConfig.history && Number(currentConfig.history.copy_max_entries)
    return value > 0 ? Math.floor(value) : 100
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

  function historyRowsChronological(){
    const rows = copyHistoryEntries.map(cloneHistoryEntry)
    if (activeHistory) {
      rows.push({ frames: activeHistory.frames || 1, buttons: cloneButtons(activeHistory.buttons) })
    }
    return rows.slice(-copyHistoryMaxEntries())
  }

  function cloneHistoryEntry(entry){
    return {
      frames: entry.frames,
      buttons: cloneButtons(entry.buttons || {})
    }
  }

  function historyMarkdownTable(){
    return historyCopyTextFromRows(historyRowsChronological())
  }

  function historyCopyTextFromRows(rows){
    return historyCopyDescription().concat([historyMarkdownFromRows(rows)]).join('\n\n')
  }

  function historyCopyDescription(){
    return [
      '入力履歴は上から下へ、古い入力から新しい入力の順に並んでいます。',
      '方向はテンキー表記です。5=ニュートラル、1=左下、2=下、3=右下、4=左、6=右、7=左上、8=上、9=右上です。',
      '方向とボタン、または複数ボタンの同時押しは + でつなぎます。',
      '1F程度でボタンや方向が増減している行は、高速な別入力ではなく、同時押しや方向入力の押下タイミングがわずかにずれたものとして扱ってください。'
    ]
  }

  function historyMarkdownFromRows(rows){
    const lines = [
      '| フレーム | コマンド |',
      '| --- | --- |'
    ]
    ;(Array.isArray(rows) ? rows : []).forEach(entry => {
      lines.push('| ' + markdownCell(entry.frames) + ' | ' + markdownCell(historyCommandText(entry.buttons || {})) + ' |')
    })
    return lines.join('\n')
  }

  function markdownCell(value){
    return String(value == null ? '' : value).replace(/\|/g, '\\|')
  }

  function applyComboConfig(nextConfig){
    comboConfig = nextConfig || null
    activePractice = cloneActivePractice(comboConfig && comboConfig.activePractice)
    resetComboProgress()
    renderCombos()
  }

  function resetComboProgress(){
    const combos = activeCombos()
    comboProgress = combos.map(() => initialComboProgress())
  }

  function initialComboProgress(){
    return { index: 0, frames: 0, complete: false, flashedAt: 0, completedAt: 0, failedAt: 0, failLocked: false, armAfterRelease: false, feedback: null, feedbackAt: 0 }
  }

  function renderCombos(){
    if (!comboPanel) return
    const header = comboPanel.querySelector('.combo-header')
    const list = comboPanel.querySelector('.combo-list')
    const set = comboConfig && comboConfig.active_set
    if (!set) {
      header.textContent = 'Combos'
      list.innerHTML = '<div class="combo-empty">No combo set selected</div>'
      return
    }
    const practiceLabel = activePracticeLabel()
    header.textContent = ''
    const combos = activeCombos()
    list.innerHTML = combos.map((combo, comboIndex) => {
      const progress = comboProgress[comboIndex] || { index: 0, complete: false }
      const steps = Array.isArray(combo.steps) ? combo.steps : []
      const expanded = expandedComboSteps(steps)
      const completeClass = progress.complete ? ' combo-complete' : ''
      const stepHTML = steps.map((step, stepIndex) => richComboStepHTML(step, expanded, progress, stepIndex)).join('')
      const name = combo.name || combo.notation || practiceLabel || set.name || set.id || 'Combo'
      return [
        '<div class="combo-row' + completeClass + '">',
        '<div class="combo-name">' + escapeHTML(name) + '</div>',
        '<div class="combo-steps">' + stepHTML + '</div>',
        '</div>'
      ].join('')
    }).join('')
    renderSuccessFlash()
  }

  function richComboStepHTML(step, expanded, progress, stepIndex){
    const displayState = comboDisplayStepState(expanded, progress, stepIndex)
    const stateClass = displayState ? ' ' + displayState : ''
    const marker = displayState === 'current' ? '<span class="combo-cursor">≫</span>' : '<span class="combo-cursor"></span>'
    const label = comboRichStepLabel(step, progress, stepIndex)
    const notation = comboStepNotation(step)
    return [
      '<div class="combo-step' + stateClass + '">',
      marker,
      '<span class="combo-step-body">',
      '<span class="combo-step-label">' + escapeHTML(label) + '</span>',
      notation ? '<span class="combo-step-notation">' + escapeHTML(notation) + '</span>' : '',
      '</span>',
      '</div>'
    ].join('')
  }

  function activeCombos(){
    const set = comboConfig && comboConfig.active_set
    const combos = set && Array.isArray(set.combos) ? set.combos : []
    if (combos.length === 0) return []
    const active = activePractice || cloneActivePractice(comboConfig && comboConfig.activePractice)
    const activeRecipeID = active && active.activeRecipeId
    if (!activeRecipeID) return [combos[0]]
    const found = combos.find(combo => combo.id === activeRecipeID)
    return found ? [found] : [combos[0]]
  }

  function cloneActivePractice(source){
    if (!source) return null
    return {
      mode: source.mode || 'focus',
      activeRecipeId: source.activeRecipeId || '',
      activeSetId: source.activeSetId || '',
      activeSetIndex: Number(source.activeSetIndex || 0)
    }
  }

  function activePracticeLabel(){
    if (!activePractice || activePractice.mode !== 'playlist') return ''
    const set = practiceSetByID(activePractice.activeSetId)
    if (!set || !Array.isArray(set.recipes) || set.recipes.length === 0) return ''
    return (set.name || activePractice.activeSetId || 'Practice') + ' ' + (activePractice.activeSetIndex + 1) + '/' + set.recipes.length
  }

  function practiceSetByID(id){
    const sets = comboConfig && Array.isArray(comboConfig.practiceSets) ? comboConfig.practiceSets : []
    return sets.find(set => set.id === id) || null
  }

  function comboStepLabel(step){
    if (!step) return ''
    if (typeof step === 'string') return step
    const resolved = resolveComboStep(step)
    if (resolved && resolved !== step) return comboStepLabel(resolved)
    if (step.label) return step.label
    if (step.notation) return step.notation
    const direction = directionToNumpad(step.direction)
    const buttons = Array.isArray(step.buttons) ? step.buttons.join('+') : ''
    return (direction || '5') + buttons
  }

  function comboStepNotation(step){
    if (!step || typeof step === 'string') return ''
    const resolved = resolveComboStep(step)
    if (resolved && resolved !== step) return comboStepNotation(resolved)
    if (step.notation) return step.notation
    const direction = directionToNumpad(step.direction)
    const buttons = Array.isArray(step.buttons) ? step.buttons.join('+') : ''
    return buttons ? (direction || '5') + buttons : ''
  }

  function comboRichStepLabel(step, progress, stepIndex){
    const label = comboStepLabel(step)
    if (!progress || !progress.feedback || progress.feedback.displayIndex !== stepIndex) return label
    if (progress.feedback.type === 'miss') return label
    const suffix = progress.feedback.type === 'early' ? 'Too Early -' + progress.feedback.frames + 'F' : 'Too Late +' + progress.feedback.frames + 'F'
    return label + '  ' + suffix
  }

  function comboStepText(step, state, progress, stepIndex){
    const label = comboStepLabel(step)
    if (progress && progress.feedback && progress.feedback.displayIndex === stepIndex) {
      if (progress.feedback.type === 'miss') return '[×] ' + label
      const suffix = progress.feedback.type === 'early' ? ' Too Early -' + progress.feedback.frames + 'F' : ' Too Late +' + progress.feedback.frames + 'F'
      return '[×] ' + label + suffix
    }
    if (state === 'done') return '[✓] ' + label
    if (state === 'current') return '[→] ' + label
    return '[ ] ' + label
  }

  function processComboInput(buttonsState, elapsedFrames){
    if (updateFacingFromButtons(buttonsState)) {
      resetComboProgress()
      renderCombos()
      return
    }
    const set = comboConfig && comboConfig.active_set
    if (!set || !Array.isArray(set.combos)) return
    if (comboResetButtonPressed(buttonsState)) {
      resetComboProgress()
      renderCombos()
      return
    }
    activeCombos().forEach((combo, comboIndex) => {
      const steps = Array.isArray(combo.steps) ? combo.steps : []
      const expanded = expandedComboSteps(steps)
      if (expanded.length === 0) return
      const progress = comboProgress[comboIndex] || { index: 0, frames: 0, complete: false, flashedAt: 0 }
      if (progress.failLocked) {
        comboProgress[comboIndex] = progress
        return
      }
      if (progress.armAfterRelease) {
        if (hasAnyInput(buttonsState)) {
          comboProgress[comboIndex] = progress
          return
        }
        progress.armAfterRelease = false
        progress.frames = 0
        comboProgress[comboIndex] = progress
        return
      }
      if (progress.complete) {
        progress.flashedAt += elapsedFrames
        if (progress.flashedAt > 120) {
          Object.assign(progress, initialComboProgress())
        }
      }
      if (!progress.complete) {
        const firstStep = expanded[0] && expanded[0].matchStep
        if (progress.feedback && progress.index === 0) {
          progress.feedbackAt = Number(progress.feedbackAt || 0) + elapsedFrames
          if (comboStepMatches(firstStep, buttonsState, set.mode)) {
            progress.feedback = null
            progress.feedbackAt = 0
          } else if (progress.feedbackAt > 30) {
            progress.feedback = null
            progress.feedbackAt = 0
          }
        }
        progress.frames += elapsedFrames
        const currentStep = expanded[progress.index] && expanded[progress.index].matchStep
        const currentMatches = comboStepMatches(currentStep, buttonsState, set.mode)
        const firstMatches = progress.index > 0 && comboStepMatches(firstStep, buttonsState, set.mode)
        if (firstMatches && !currentMatches) {
          progress.index = 1
          progress.frames = 0
          progress.feedback = null
          progress.feedbackAt = 0
          comboProgress[comboIndex] = progress
          return
        }
        const timing = comboTimingForProgress(expanded, progress)
        if (timing.state === 'late') {
          failComboProgress(progress, expanded, progress.index, 'late', timing.window, buttonsState)
        } else if (timing.state === 'early') {
          if (currentMatches) {
            failComboProgress(progress, expanded, progress.index, 'early', timing.window, buttonsState)
          }
          comboProgress[comboIndex] = progress
          return
        } else if (currentMatches) {
          progress.feedback = null
          progress.feedbackAt = 0
          const matchedExpandedIndex = progress.index
          progress.index += 1
          if (comboDisplayStepCompleted(expanded, matchedExpandedIndex, progress.index)) {
            playComboSound('step')
          }
          progress.frames = 0
          dispatchRecipeProgress(combo, progress, expanded)
          if (progress.index >= expanded.length) {
            progress.complete = true
            progress.flashedAt = 0
            progress.completedAt = performance.now()
            showSuccessFlash()
            playComboSound('complete')
            dispatchRecipeCompleted(combo)
            if (handleRecipeCompleted()) return
          }
        } else if (progress.index > 0) {
          if (firstMatches) {
            progress.index = 1
            progress.frames = 0
            progress.feedback = null
            progress.feedbackAt = 0
          } else if (comboProgressIsStale(expanded, progress)) {
            failComboProgress(progress, expanded, progress.index, 'miss', null, buttonsState)
          }
        }
      }
      comboProgress[comboIndex] = progress
    })
    renderCombos()
  }

  function comboResetButtonPressed(buttonsState){
    if (!buttonsState) return false
    if (facingToggleChordPressed(buttonsState)) return false
    if (backButtonPressed(buttonsState)) return true
    return false
  }

  function updateFacingFromButtons(buttonsState){
    const pressed = facingToggleChordPressed(buttonsState)
    if (!pressed) {
      facingToggleChordDown = false
      return false
    }
    if (facingToggleChordDown) return false
    facingToggleChordDown = true
    facing = facing === 'right' ? 'left' : 'right'
    dispatchComboEvent('facing_changed', {
      facing,
      label: facingLabel()
    })
    return true
  }

  function facingToggleChordPressed(buttonsState){
    return !!(buttonsState && buttonsState.up && backButtonPressed(buttonsState))
  }

  function backButtonPressed(buttonsState){
    if (!buttonsState) return false
    if (buttonsState.s1 || buttonsState.ba || buttonsState.BA || buttonsState.back || buttonsState.BACK) return true
    if (!currentConfig || !Array.isArray(currentConfig.buttons)) return false
    return currentConfig.buttons.some(button => {
      const id = button && button.id
      if (!id || !buttonsState[id]) return false
      const label = normalizeAlias((button.history_label || button.label || id))
      return label === 'BA' || label === 'BACK'
    })
  }

  function facingLabel(){
    return facing === 'left' ? '左向き (2P)' : '右向き (1P)'
  }

  function failComboProgress(progress, expanded, expandedIndex, type, window, buttonsState){
    applyTimingFeedback(progress, expanded, expandedIndex, type, window, buttonsState)
    progress.index = 0
    progress.frames = 0
    progress.complete = false
    progress.flashedAt = 0
    progress.failedAt = performance.now()
    progress.failLocked = true
    progress.feedbackAt = 0
  }

  function resetFailedProgress(now){
    let changed = false
    comboProgress.forEach(progress => {
      if (progress && progress.failLocked && progress.failedAt !== undefined && now - progress.failedAt > 900) {
        Object.assign(progress, initialComboProgress())
        changed = true
      }
    })
    return changed
  }

  function comboProgressIsStale(expanded, progress){
    const previous = expanded && expanded[progress.index - 1]
    const current = expanded && expanded[progress.index]
    const window = previous && previous.inputWindow
    if (window && Number(window.end || 0) > 0) {
      return progress.frames > Number(window.end || 0)
    }
    if (previous && current && previous.move && current.move) {
      const timing = canTransitionByCancelWindow(previous.move, current.move, progress.frames)
      return timing && timing.state === 'late'
    }
    return progress.frames > 90
  }

  function handleRecipeCompleted(){
    if (!activePractice || activePractice.mode !== 'playlist') return false
    const set = practiceSetByID(activePractice.activeSetId)
    if (!set || set.advanceOnComplete !== true || !Array.isArray(set.recipes) || set.recipes.length === 0) return false
    let nextIndex = activePractice.activeSetIndex + 1
    if (nextIndex >= set.recipes.length) {
      if (!set.loop) return false
      nextIndex = 0
    }
    activePractice.activeSetIndex = nextIndex
    activePractice.activeRecipeId = set.recipes[nextIndex]
    resetComboProgress()
    comboProgress.forEach(progress => {
      progress.armAfterRelease = true
    })
    dispatchActiveRecipeChanged()
    return true
  }

  function applyTimingFeedback(progress, expanded, expandedIndex, type, feedbackWindow, buttonsState){
    const part = expanded[Math.max(0, expandedIndex)]
    const window = feedbackWindow
    let frames = 0
    if (window) {
      frames = type === 'early' ? Math.max(0, Number(window.start || 0) - progress.frames) : Math.max(0, progress.frames - Number(window.end || 0))
    }
    const issue = type === 'miss' ? comboInputIssue(part && part.matchStep, buttonsState) : { reason: type }
    progress.feedback = Object.assign({ displayIndex: part ? part.displayIndex : 0, type, frames }, issue)
  }

  function comboInputIssue(step, buttonsState){
    if (!step) return { reason: 'command_missing' }
    const expectedDirection = step.direction || '5'
    const actualDirection = currentNumpadDirection(buttonsState || {})
    const buttons = Array.isArray(step.buttons) ? step.buttons : []
    const directionOK = expectedDirection === actualDirection || (!step.direction && actualDirection === '5')
    const missingButtons = buttons.filter(id => !comboButtonPressed(id, buttonsState || {}))
    const anyExpectedButtonPressed = buttons.some(id => comboButtonPressed(id, buttonsState || {}))
    if (!directionOK && anyExpectedButtonPressed) {
      return { reason: 'button_lead', expectedDirection, actualDirection, missingButtons }
    }
    if (!directionOK) {
      return { reason: 'direction_missing', expectedDirection, actualDirection, missingButtons }
    }
    if (missingButtons.length > 0) {
      return { reason: 'button_missing', expectedDirection, actualDirection, missingButtons }
    }
    return { reason: 'command_missing', expectedDirection, actualDirection, missingButtons }
  }

  function playComboSound(type){
    if (!ensureAudioContext()) return
    try {
      if (audioContext.state === 'suspended') return
      const now = audioContext.currentTime
      if (type === 'complete') {
        playTone(now, 880, 0.07, 0.08)
        playTone(now + 0.08, 1320, 0.16, 0.11)
        return
      }
      playTone(now, 740, 0.045, 0.07)
      playTone(now + 0.045, 980, 0.07, 0.06)
    } catch (error) {
      audioContext = null
    }
  }

  function showSuccessFlash(){
    successFlash = { startedAt: performance.now() }
    renderSuccessFlash()
  }

  function renderSuccessFlash(){
    if (!comboPanel) return
    const el = comboPanel.querySelector('.combo-success')
    if (!el) return
    const visible = successFlash && performance.now() - successFlash.startedAt <= 1500
    el.classList.toggle('show', !!visible)
    if (!visible && successFlash) {
      successFlash = null
    }
  }

  function ensureAudioContext(){
    const Ctx = window.AudioContext || window.webkitAudioContext
    if (!Ctx) return false
    if (!audioContext) audioContext = new Ctx()
    return true
  }

  function unlockAudio(){
    if (audioUnlocked || !ensureAudioContext()) return
    const finish = () => {
      if (!audioContext || audioContext.state !== 'running') return
      audioUnlocked = true
      playSilentTone()
      removeAudioUnlockListeners()
    }
    try {
      if (audioContext.state === 'suspended' && audioContext.resume) {
        audioContext.resume().then(finish).catch(() => {})
        return
      }
      finish()
    } catch (error) {
      audioContext = null
    }
  }

  function playSilentTone(){
    const now = audioContext.currentTime
    const oscillator = audioContext.createOscillator()
    const gain = audioContext.createGain()
    gain.gain.setValueAtTime(0.0001, now)
    gain.gain.setValueAtTime(0.0001, now + 0.02)
    oscillator.connect(gain)
    gain.connect(audioContext.destination)
    oscillator.start(now)
    oscillator.stop(now + 0.02)
  }

  function addAudioUnlockListeners(){
    ;['pointerdown', 'keydown', 'touchstart'].forEach(type => {
      window.addEventListener(type, unlockAudio, { passive: true })
    })
  }

  function removeAudioUnlockListeners(){
    ;['pointerdown', 'keydown', 'touchstart'].forEach(type => {
      window.removeEventListener(type, unlockAudio)
    })
  }

  function playTone(start, frequency, duration, gainValue){
    const oscillator = audioContext.createOscillator()
    const gain = audioContext.createGain()
    oscillator.type = 'sine'
    oscillator.frequency.setValueAtTime(frequency, start)
    const volume = comboAudioVolume()
    if (volume <= 0) return
    gain.gain.setValueAtTime(0.0001, start)
    gain.gain.exponentialRampToValueAtTime(gainValue * volume, start + 0.01)
    gain.gain.exponentialRampToValueAtTime(0.0001, start + duration)
    oscillator.connect(gain)
    gain.connect(audioContext.destination)
    oscillator.start(start)
    oscillator.stop(start + duration + 0.02)
  }

  function comboAudioVolume(){
    const raw = currentConfig && currentConfig.combo_audio && Number(currentConfig.combo_audio.volume)
    if (!Number.isFinite(raw)) return 0.7
    return Math.max(0, Math.min(1, raw))
  }

  function dispatchRecipeProgress(combo, progress, expanded){
    const currentStep = Math.min(progress.index, expanded.length)
    const nextPart = expanded[progress.index]
    const matchedPart = expanded[progress.index - 1]
    dispatchComboEvent('recipe_progress', {
      id: combo.id || '',
      name: combo.name || '',
      currentStep,
      totalSteps: expanded.length,
      matchedLabel: matchedPart ? comboStepLabel((combo.steps || [])[matchedPart.displayIndex]) : '',
      nextLabel: nextPart ? comboStepLabel((combo.steps || [])[nextPart.displayIndex]) : '',
      frameNo: Math.round(performance.now() / (1000 / 60))
    })
  }

  function dispatchRecipeCompleted(combo){
    dispatchComboEvent('recipe_completed', {
      id: combo.id || '',
      name: combo.name || '',
      frameNo: Math.round(performance.now() / (1000 / 60))
    })
  }

  function dispatchActiveRecipeChanged(){
    if (!activePractice) return
    dispatchComboEvent('active_recipe_changed', {
      mode: activePractice.mode || 'focus',
      activeSetId: activePractice.activeSetId || '',
      activeSetIndex: activePractice.activeSetIndex || 0,
      activeRecipeId: activePractice.activeRecipeId || ''
    })
  }

  function dispatchComboEvent(type, detail){
    const payload = Object.assign({ type }, detail || {})
    window.dispatchEvent(new CustomEvent(type, { detail: payload }))
  }

  function expandedComboSteps(steps){
    const expanded = []
    steps.forEach((step, displayIndex) => {
      const parts = expandComboStep(step)
      parts.forEach(part => expanded.push({
        displayIndex,
        matchStep: part.matchStep,
        inputWindow: part.inputWindow || null,
        move: part.move,
        sourceStep: part.sourceStep || step
      }))
    })
    return expanded
  }

  function expandComboStep(step){
    if (!step) return []
    const resolved = resolveComboStep(step)
    const notation = resolved && resolved.notation
    if (notation) {
      return expandNotationStep(notation, resolved)
    }
    return [{ matchStep: normalizeComboStep(resolved), inputWindow: null, move: resolved.move, sourceStep: resolved }]
  }

  function resolveComboStep(step){
    if (!step || typeof step === 'string') return step
    const move = moveByID(step.move)
    if (!move) return step
    const command = commandByID(move.command || move.commandID || step.command)
    return Object.assign({}, step, {
      move,
      command: step.command || move.command || move.commandID || '',
      notation: step.notation || move.input || (command && command.notation) || move.notation || '',
      label: step.label || move.name || (command && command.name) || step.label || ''
    })
  }

  function moveByID(id){
    const moves = comboConfig && Array.isArray(comboConfig.moves) ? comboConfig.moves : []
    return moves.find(move => move.id === id) || null
  }

  function commandByID(id){
    const commands = comboConfig && Array.isArray(comboConfig.commands) ? comboConfig.commands : []
    return commands.find(command => command.id === id) || null
  }

  function expandNotationStep(value, sourceStep){
    const text = String(value || '').trim()
    const match = text.match(/^([1-9]+)?(.*)$/)
    const directions = match && match[1] ? match[1].split('') : []
    const buttons = normalizeNotationButtons(match && match[2] ? match[2] : '')
    if (directions.length === 0) {
      return [{ matchStep: { direction: null, buttons }, inputWindow: null, move: sourceStep && sourceStep.move, sourceStep }]
    }
    return directions.map((direction, index) => ({
      matchStep: {
        direction: normalizeDirection(direction),
        buttons: index === directions.length - 1 ? buttons : []
      },
      inputWindow: index === directions.length - 1 ? null : gapInputWindow(sourceStep),
      move: index === directions.length - 1 ? sourceStep && sourceStep.move : null,
      sourceStep
    }))
  }

  function comboDisplayStepState(expanded, progress, displayIndex){
    if (progress && progress.feedback && progress.feedback.displayIndex === displayIndex) return 'failed'
    if (progress && progress.failLocked) return ''
    if (progress.complete) return 'done'
    const indexes = expanded.map((part, index) => part.displayIndex === displayIndex ? index : -1).filter(index => index >= 0)
    if (indexes.length === 0) return ''
    const first = indexes[0]
    const last = indexes[indexes.length - 1]
    if (progress.index > last) return 'done'
    if (progress.index >= first && progress.index <= last) return 'current'
    return ''
  }

  function comboDisplayStepCompleted(expanded, matchedExpandedIndex, nextExpandedIndex){
    const matched = expanded && expanded[matchedExpandedIndex]
    if (!matched) return false
    const next = expanded[nextExpandedIndex]
    return !next || next.displayIndex !== matched.displayIndex
  }

  function comboTimingForProgress(expanded, progress){
    if (!expanded || progress.index <= 0) return { state: 'ok' }
    const previous = expanded[progress.index - 1]
    const current = expanded[progress.index]
    const window = previous && previous.inputWindow
    if (window) return timingForWindow(window, progress.frames)
    const cancelTiming = comboCancelTiming(previous, current, progress.frames)
    if (cancelTiming) return cancelTiming
    return { state: 'ok' }
  }

  function timingForWindow(window, frames){
    const start = Number(window.start || 0)
    const end = Number(window.end || 0)
    if (frames < start) return { state: 'early', window }
    if (end > 0 && frames > end) return { state: 'late', window }
    return { state: 'ok', window }
  }

  function comboCancelTiming(previous, current, frames){
    if (!previous || !current) return null
    if (previous.displayIndex === current.displayIndex) return null
    if (!previous.move || !current.move) return null
    return canTransitionByCancelWindow(previous.move, current.move, frames)
  }

  function canTransitionByCancelWindow(fromMove, toMove, elapsed){
    const windows = Array.isArray(fromMove && fromMove.cancelWindows) ? fromMove.cancelWindows : []
    const candidates = windows.filter(window => cancelWindowTargetsMove(window, toMove))
    if (candidates.length === 0) return { state: 'ok' }
    const matching = candidates.find(window => elapsed >= Number(window.start || 0) && elapsed <= Number(window.end || 0))
    if (matching) return { state: 'ok', ok: true, window: matching }
    const closest = closestCancelWindow(candidates, elapsed)
    if (!closest) return { state: 'ok' }
    if (elapsed < Number(closest.start || 0)) return { state: 'early', ok: false, window: closest }
    return { state: 'late', ok: false, window: closest }
  }

  function cancelWindowTargetsMove(window, move){
    if (!window || !move) return false
    const targets = Array.isArray(window.targets) ? window.targets : []
    if (targets.includes(move.id)) return true
    return intersects(window.targetTags || [], move.tags || [])
  }

  function closestCancelWindow(windows, elapsed){
    let best = null
    let bestDistance = Infinity
    windows.forEach(window => {
      const start = Number(window.start || 0)
      const end = Number(window.end || 0)
      const distance = elapsed < start ? start - elapsed : elapsed > end ? elapsed - end : 0
      if (distance < bestDistance) {
        best = window
        bestDistance = distance
      }
    })
    return best
  }

  function intersects(left, right){
    const rightSet = new Set((Array.isArray(right) ? right : []).map(value => String(value)))
    return (Array.isArray(left) ? left : []).some(value => rightSet.has(String(value)))
  }

  function gapInputWindow(step){
    if (!step || typeof step === 'string') return { start: 0, end: 8 }
    const value = Number(step.gap_window_frames || step.gapWindowFrames || step.maxGapFrames || 0)
    return { start: 0, end: value > 0 ? value : 8 }
  }

  function normalizeComboStep(step){
    if (!step) return { direction: null, buttons: [] }
    if (typeof step === 'string') return parseNotationStep(step)
    if (step.notation) return parseNotationStep(step.notation)
    return {
      direction: normalizeDirection(step.direction),
      buttons: normalizeStepButtons(step.buttons || [])
    }
  }

  function parseNotationStep(value){
    const text = String(value || '').trim()
    const match = text.match(/^([1-9])?(.*)$/)
    const direction = normalizeDirection(match && match[1] ? match[1] : '')
    const buttonsText = (match && match[2] ? match[2] : '').trim()
    return { direction, buttons: normalizeNotationButtons(buttonsText) }
  }

  function normalizeNotationButtons(value){
    const text = String(value || '').trim()
    if (!text) return []
    if (text.includes('+') || text.includes(',')) {
      return normalizeStepButtons(text.split(/[+,]/))
    }
    const aliases = buttonAliases().sort((a, b) => b.alias.length - a.alias.length)
    const out = []
    let rest = normalizeAlias(text)
    while (rest) {
      const found = aliases.find(item => rest.startsWith(item.alias))
      if (!found) {
        out.push(rest)
        break
      }
      out.push(found.id)
      rest = rest.slice(found.alias.length)
    }
    return normalizeStepButtons(out)
  }

  function normalizeStepButtons(values){
    const aliases = buttonAliasMap()
    return values.map(value => {
      const key = normalizeAlias(value)
      return aliases[key] || String(value || '').trim()
    }).filter(Boolean)
  }

  function buttonAliasMap(){
    const out = {}
    if (currentConfig && Array.isArray(currentConfig.buttons)) {
      currentConfig.buttons.forEach(button => {
        ;[button.id, button.label, button.history_label].forEach(alias => {
          const key = normalizeAlias(alias)
          if (key) out[key] = button.id
        })
      })
    }
    Object.assign(out, defaultButtonAliases())
    return out
  }

  function defaultButtonAliases(){
    return {
      LP: 'b3',
      MP: 'b4',
      HP: 'r1',
      LK: 'b1',
      MK: 'b2',
      HK: 'r2',
      P: 'b3|b4|r1',
      PUNCH: 'b3|b4|r1',
      K: 'b1|b2|r2',
      KICK: 'b1|b2|r2'
    }
  }

  function buttonAliases(){
    const aliases = []
    const seen = {}
    const map = buttonAliasMap()
    Object.keys(map).forEach(alias => {
      if (!seen[alias]) {
        seen[alias] = true
        aliases.push({ alias, id: map[alias] })
      }
    })
    return aliases
  }

  function normalizeAlias(value){
    return String(value || '').trim().toUpperCase().replace(/\s+/g, '')
  }

  function comboStepMatches(step, buttonsState, mode){
    if (!step) return false
    if ((mode || 'command') !== 'normal') {
      const currentDirection = currentNumpadDirection(buttonsState)
      if ((step.direction || '5') !== currentDirection) return false
    } else if (step.direction && step.direction !== '5' && step.direction !== currentNumpadDirection(buttonsState)) {
      return false
    }
    if (!step.buttons || step.buttons.length === 0) {
      return step.direction && step.direction !== '5' ? true : !hasAnyInput(buttonsState)
    }
    return step.buttons.every(id => comboButtonPressed(id, buttonsState))
  }

  function comboButtonPressed(id, buttonsState){
    const value = String(id || '')
    if (value.includes('|')) {
      return value.split('|').some(part => !!buttonsState[part])
    }
    return !!buttonsState[value]
  }

  function currentNumpadDirection(buttonsState){
    const up = !!buttonsState.up
    const down = !!buttonsState.down
    const left = !!buttonsState.left
    const right = !!buttonsState.right
    if (down && left && !up && !right) return relativeDirection('1')
    if (down && !left && !right) return '2'
    if (down && right && !up && !left) return relativeDirection('3')
    if (left && !up && !down && !right) return relativeDirection('4')
    if (right && !up && !down && !left) return relativeDirection('6')
    if (up && left && !down && !right) return relativeDirection('7')
    if (up && !left && !right) return '8'
    if (up && right && !down && !left) return relativeDirection('9')
    return '5'
  }

  function relativeDirection(direction){
    if (facing !== 'left') return direction
    return ({ '1': '3', '3': '1', '4': '6', '6': '4', '7': '9', '9': '7' })[direction] || direction
  }

  function normalizeDirection(value){
    const text = normalizeAlias(value)
    const direct = {
      '': null,
      '5': '5',
      'N': '5',
      'NEUTRAL': '5',
      '1': '1',
      'DOWNLEFT': '1',
      'DOWN-LEFT': '1',
      '2': '2',
      'DOWN': '2',
      '3': '3',
      'DOWNRIGHT': '3',
      'DOWN-RIGHT': '3',
      '4': '4',
      'LEFT': '4',
      '6': '6',
      'RIGHT': '6',
      '7': '7',
      'UPLEFT': '7',
      'UP-LEFT': '7',
      '8': '8',
      'UP': '8',
      '9': '9',
      'UPRIGHT': '9',
      'UP-RIGHT': '9'
    }
    return Object.prototype.hasOwnProperty.call(direct, text) ? direct[text] : text
  }

  function directionToNumpad(value){
    return normalizeDirection(value) || ''
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

  function historyCommandText(buttonsState){
    const direction = historyNumpadDirection(buttonsState || {})
    const labels = buttonOrder
      .filter(id => buttonsState && buttonsState[id])
      .map(id => buttonCommandLabel(id))
    if (labels.length === 0) return direction
    if (direction === '5') return labels.join('+')
    return [direction].concat(labels).join('+')
  }

  function historyNumpadDirection(buttonsState){
    const up = !!buttonsState.up
    const down = !!buttonsState.down
    const left = !!buttonsState.left
    const right = !!buttonsState.right
    if (down && left && !up && !right) return '1'
    if (down && !left && !right) return '2'
    if (down && right && !up && !left) return '3'
    if (left && !up && !down && !right) return '4'
    if (right && !up && !down && !left) return '6'
    if (up && left && !down && !right) return '7'
    if (up && !left && !right) return '8'
    if (up && right && !down && !left) return '9'
    return '5'
  }

  function buttonCommandLabel(id){
    const def = buttonDefFor(id)
    return (def && (def.history_label || def.label)) || labelFor(id) || id
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
      fetchComboConfig().then(cfg=>{ applyComboConfig(cfg) }).catch(()=>{})
    }
    ws.onmessage = e => {
      try{
        const data = JSON.parse(e.data)
        if (data.type === 'config' && data.config) {
          buildButtons(data.config)
        } else if (data.type === 'combo_config' && data.combo_config) {
          applyComboConfig(data.combo_config)
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

  window.inputCastComboTest = {
    canTransitionByCancelWindow,
    cancelWindowTargetsMove,
    comboButtonPressed,
    comboInputIssue,
    comboStepMatches,
    expandNotationStep,
    comboDisplayStepCompleted,
    normalizeComboStep,
    normalizeNotationButtons,
    parseNotationStep,
    timingForWindow,
    comboAudioVolume,
    intersects,
    historyCommandText,
    copyHistoryMaxEntries,
    historyCopyTextFromRows,
    historyMarkdownFromRows,
    clearHistory,
    updateHistoryFromState,
    processComboInput,
    unlockAudio,
    updateFacingFromButtons,
    resetFailedProgress,
    setCurrentConfig(config){
      currentConfig = config || null
    },
    setComboConfig(config){
      applyComboConfig(config || null)
    },
    setFacing(value){
      facing = value === 'left' ? 'left' : 'right'
      facingToggleChordDown = false
    },
    getFacing(){
      return facing
    },
    getComboProgress(){
      return comboProgress.map(progress => Object.assign({}, progress, {
        feedback: progress.feedback ? Object.assign({}, progress.feedback) : null
      }))
    },
    getHistoryRows(){
      return historyRowsChronological()
    }
  }

  // initial load
  fetchConfig().then(cfg=>{ buildButtons(cfg); return fetchComboConfig() }).then(cfg=>{ applyComboConfig(cfg); connect() }).catch(e=>{ console.error(e); connect() })
  window.addEventListener('resize', () => {
    hideHistoryMenu()
    fitOverlay()
  })
  if (document.addEventListener) {
    document.addEventListener('click', hideHistoryMenu)
    document.addEventListener('keydown', event => {
      if (event.key === 'Escape') hideHistoryMenu()
    })
  }
  addAudioUnlockListeners()
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
      if (comboProgress.some(progress => progress && progress.complete)) {
        const now = performance.now()
        let changed = false
        comboProgress.forEach(progress => {
          if (progress && progress.complete && progress.completedAt && now - progress.completedAt > 2000) {
            Object.assign(progress, initialComboProgress())
            changed = true
          }
        })
        if (changed || comboProgress.some(progress => progress && progress.complete && progress.completedAt && now - progress.completedAt <= 2100)) {
          renderCombos()
        }
      }
      if (comboProgress.some(progress => progress && progress.failLocked)) {
        const now = performance.now()
        if (resetFailedProgress(now)) {
          renderCombos()
        }
      }
      if (successFlash) {
        renderSuccessFlash()
      }
      rafId = window.requestAnimationFrame(tick)
    }
    rafId = window.requestAnimationFrame(tick)
  }

})();
