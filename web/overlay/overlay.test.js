const assert = require('node:assert/strict')
const fs = require('node:fs')
const path = require('node:path')
const vm = require('node:vm')

function element(){
  return {
    style: { setProperty(){} },
    className: '',
    innerHTML: '',
    textContent: '',
    dataset: {},
    appendChild(){},
    querySelector(){ return element() },
    classList: { add(){}, toggle(){} },
    setAttribute(){},
    addEventListener(){}
  }
}

const windowObject = {
  location: { pathname: '/overlay', protocol: 'http:', host: 'localhost:8080' },
  addEventListener(){},
  dispatchEvent(){},
  requestAnimationFrame(){ return 1 }
}
windowObject.window = windowObject

const context = {
  console,
  setTimeout(){},
  performance: { now(){ return 0 } },
  CustomEvent: function CustomEvent(type, init){ this.type = type; this.detail = init && init.detail },
  WebSocket: function WebSocket(){
    this.onopen = null
    this.onmessage = null
    this.onclose = null
  },
  fetch(url){
    const payload = String(url).includes('/api/combos')
      ? { current: {}, files: [] }
      : { controller: {}, buttons: [] }
    return Promise.resolve({ json: () => Promise.resolve(payload) })
  },
  document: {
    getElementById(){ return element() },
    createElement(){ return element() }
  },
  window: windowObject,
  location: windowObject.location
}

vm.createContext(context)
vm.runInContext(fs.readFileSync(path.join(__dirname, 'overlay.js'), 'utf8'), context)

const hook = context.window.inputCastComboTest

hook.setCurrentConfig({
  buttons: [
    { id: 'b1', label: 'LK', history_label: 'LK' },
    { id: 'b2', label: 'MK', history_label: 'MK' },
    { id: 'b3', label: 'LP', history_label: 'LP' },
    { id: 'b4', label: 'MP', history_label: 'MP' },
    { id: 'r1', label: 'HP', history_label: 'HP' },
    { id: 'r2', label: 'HK', history_label: 'HK' }
  ]
})

{
  hook.setCurrentConfig({ combo_audio: { volume: 0.25 }, buttons: [] })
  assert.equal(hook.comboAudioVolume(), 0.25)
  hook.setCurrentConfig({ combo_audio: { volume: 2 }, buttons: [] })
  assert.equal(hook.comboAudioVolume(), 1)
  hook.setCurrentConfig({ combo_audio: { volume: -1 }, buttons: [] })
  assert.equal(hook.comboAudioVolume(), 0)
}

hook.setCurrentConfig({
  buttons: [
    { id: 'b1', label: 'LK', history_label: 'LK' },
    { id: 'b2', label: 'MK', history_label: 'MK' },
    { id: 'b3', label: 'LP', history_label: 'LP' },
    { id: 'b4', label: 'MP', history_label: 'MP' },
    { id: 'r1', label: 'HP', history_label: 'HP' },
    { id: 'r2', label: 'HK', history_label: 'HK' }
  ]
})

{
  const from = { id: 'jab', cancelWindows: [{ start: 4, end: 8, targets: ['target'] }] }
  assert.equal(hook.canTransitionByCancelWindow(from, { id: 'target' }, 6).state, 'ok')
}

{
  const from = { id: 'jab', cancelWindows: [{ start: 4, end: 8, targetTags: ['special'] }] }
  assert.equal(hook.canTransitionByCancelWindow(from, { id: 'fireball', tags: ['special'] }, 6).state, 'ok')
}

{
  const from = { id: 'jab', cancelWindows: [{ start: 4, end: 8, targets: ['target'] }] }
  const early = hook.canTransitionByCancelWindow(from, { id: 'target' }, 2)
  const late = hook.canTransitionByCancelWindow(from, { id: 'target' }, 10)
  assert.equal(early.state, 'early')
  assert.equal(late.state, 'late')
}

{
  assert.equal(JSON.stringify(hook.parseNotationStep('2MK')), JSON.stringify({ direction: '2', buttons: ['b2'] }))
  const commandParts = hook.expandNotationStep('236P', { maxGapFrames: 12 })
  assert.equal(commandParts.length, 3)
  assert.equal(commandParts[0].matchStep.direction, '2')
  assert.equal(commandParts[1].matchStep.direction, '3')
  assert.equal(commandParts[2].matchStep.direction, '6')
  assert.equal(JSON.stringify(commandParts[2].matchStep.buttons), JSON.stringify(['b3|b4|r1']))
  assert.equal(hook.comboStepMatches({ direction: '2', buttons: ['b2'] }, { down: true, b2: true }, 'command'), true)
  assert.equal(hook.comboStepMatches({ direction: '6', buttons: ['b3|b4|r1'] }, { right: true, r1: true }, 'command'), true)
  assert.equal(hook.comboInputIssue({ direction: '6', buttons: ['r1'] }, { down: true, r1: true }).reason, 'button_lead')
  assert.equal(hook.comboInputIssue({ direction: '2', buttons: ['b2'] }, { down: true }).reason, 'button_missing')
  assert.equal(hook.comboInputIssue({ direction: '2', buttons: ['b2'] }, { right: true }).reason, 'direction_missing')
}

{
  hook.setFacing('right')
  assert.equal(hook.updateFacingFromButtons({ up: true, s1: true }), true)
  assert.equal(hook.getFacing(), 'left')
  assert.equal(hook.updateFacingFromButtons({ up: true, s1: true }), false)
  assert.equal(hook.getFacing(), 'left')
  assert.equal(hook.updateFacingFromButtons({}), false)
  assert.equal(hook.updateFacingFromButtons({ up: true, s1: true }), true)
  assert.equal(hook.getFacing(), 'right')
}

{
  hook.setFacing('left')
  assert.equal(hook.comboStepMatches({ direction: '6', buttons: ['b3'] }, { left: true, b3: true }, 'command'), true)
  assert.equal(hook.comboStepMatches({ direction: '6', buttons: ['b3'] }, { right: true, b3: true }, 'command'), false)
  assert.equal(hook.comboStepMatches({ direction: '4', buttons: ['b3'] }, { right: true, b3: true }, 'command'), true)
  hook.setFacing('right')
}

{
  const commandParts = hook.expandNotationStep('236P', { maxGapFrames: 8 })
  assert.equal(hook.comboDisplayStepCompleted(commandParts, 0, 1), false)
  assert.equal(hook.comboDisplayStepCompleted(commandParts, 1, 2), false)
  assert.equal(hook.comboDisplayStepCompleted(commandParts, 2, 3), true)
}

{
  hook.setComboConfig({
    active_set: {
      id: 'routes',
      mode: 'command',
      combos: [{
        id: 'light',
        name: 'Light',
        steps: [
          { move: 'ryu_5lp', notation: '5LP' },
          { move: 'ryu_2lk', notation: '2LK' }
        ]
      }]
    },
    activePractice: { mode: 'focus', activeRecipeId: 'light' },
    moves: [
      { id: 'ryu_5lp', input: '5LP', cancelWindows: [{ start: 4, end: 8, targets: ['ryu_2lk'] }] },
      { id: 'ryu_2lk', input: '2LK' }
    ]
  })
  hook.processComboInput({ b3: true }, 1)
  hook.processComboInput({ down: true, b1: true }, 12)
  let progress = hook.getComboProgress()[0]
  assert.equal(progress.index, 0)
  assert.equal(progress.feedback.type, 'late')
  assert.equal(progress.failLocked, true)

  hook.processComboInput({ b3: true }, 1)
  progress = hook.getComboProgress()[0]
  assert.equal(progress.index, 0)
  assert.equal(progress.failLocked, true)

  assert.equal(hook.resetFailedProgress(progress.failedAt + 901), true)
  progress = hook.getComboProgress()[0]
  assert.equal(progress.feedback, null)
  assert.equal(progress.failLocked, false)

  hook.processComboInput({ b3: true }, 1)
  progress = hook.getComboProgress()[0]
  assert.equal(progress.index, 1)

  hook.processComboInput({ down: true, b1: true }, 5)
  progress = hook.getComboProgress()[0]
  assert.equal(progress.complete, true)
}

{
  hook.setComboConfig({
    active_set: {
      id: 'routes',
      mode: 'command',
      combos: [{
        id: 'light',
        name: 'Light',
        steps: [
          { move: 'ryu_5lp', notation: '5LP' },
          { move: 'ryu_2lk', notation: '2LK' }
        ]
      }]
    },
    activePractice: { mode: 'focus', activeRecipeId: 'light' },
    moves: [
      { id: 'ryu_5lp', input: '5LP', cancelWindows: [{ start: 4, end: 8, targets: ['ryu_2lk'] }] },
      { id: 'ryu_2lk', input: '2LK' }
    ]
  })
  hook.processComboInput({ b3: true }, 1)
  hook.processComboInput({ b3: true }, 12)
  let progress = hook.getComboProgress()[0]
  assert.equal(progress.index, 1)
  assert.equal(progress.feedback, null)

  hook.processComboInput({ down: true, b1: true }, 5)
  progress = hook.getComboProgress()[0]
  assert.equal(progress.complete, true)
}

{
  hook.setComboConfig({
    active_set: {
      id: 'routes',
      mode: 'command',
      combos: [
        { id: 'first', name: 'First', steps: [{ notation: '5LP' }] },
        { id: 'second', name: 'Second', steps: [{ notation: '5LK' }] }
      ]
    },
    activePractice: { mode: 'playlist', activeSetId: 'playlist', activeSetIndex: 0, activeRecipeId: 'first' },
    practiceSets: [{ id: 'playlist', recipes: ['first', 'second'], loop: true, advanceOnComplete: false }]
  })
  hook.processComboInput({ b3: true }, 1)
  let progress = hook.getComboProgress()[0]
  assert.equal(progress.complete, true)

  hook.setComboConfig({
    active_set: {
      id: 'routes',
      mode: 'command',
      combos: [
        { id: 'first', name: 'First', steps: [{ notation: '5LP' }] },
        { id: 'second', name: 'Second', steps: [{ notation: '5LK' }] }
      ]
    },
    activePractice: { mode: 'playlist', activeSetId: 'playlist', activeSetIndex: 0, activeRecipeId: 'first' },
    practiceSets: [{ id: 'playlist', recipes: ['first', 'second'], loop: true, advanceOnComplete: true }]
  })
  hook.processComboInput({ b3: true }, 1)
  progress = hook.getComboProgress()[0]
  assert.equal(progress.index, 0)
  assert.equal(progress.complete, false)
}

{
  hook.setComboConfig({
    active_set: {
      id: 'routes',
      mode: 'command',
      combos: [
        { id: 'fireball', name: 'Fireball', steps: [{ notation: '236P' }] },
        { id: 'jab', name: 'Jab', steps: [{ notation: '5LP' }] }
      ]
    },
    activePractice: { mode: 'playlist', activeSetId: 'playlist', activeSetIndex: 0, activeRecipeId: 'fireball' },
    practiceSets: [{ id: 'playlist', recipes: ['fireball', 'jab'], loop: false, advanceOnComplete: true }]
  })
  hook.processComboInput({ down: true }, 1)
  hook.processComboInput({ down: true, right: true }, 1)
  hook.processComboInput({ right: true, b3: true }, 1)
  let progress = hook.getComboProgress()[0]
  assert.equal(progress.armAfterRelease, true)
  assert.equal(progress.index, 0)

  hook.processComboInput({ b3: true }, 1)
  progress = hook.getComboProgress()[0]
  assert.equal(progress.index, 0)
  assert.equal(progress.armAfterRelease, true)

  hook.processComboInput({}, 1)
  progress = hook.getComboProgress()[0]
  assert.equal(progress.index, 0)
  assert.equal(progress.armAfterRelease, false)

  hook.processComboInput({ b3: true }, 1)
  progress = hook.getComboProgress()[0]
  assert.equal(progress.complete, true)
}
