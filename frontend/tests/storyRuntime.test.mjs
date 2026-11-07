import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { runInNewContext } from 'node:vm'
import test from 'node:test'

const source = readFileSync(new URL('../../backend/internal/service/templates/story-runtime.js', import.meta.url), 'utf8')
const template = readFileSync(new URL('../../backend/internal/service/templates/player.html', import.meta.url), 'utf8')
const createRuntime = runInNewContext(source + '\ncreateStoryRuntime')
const vars = [{id:1,initialValue:3}, {id:2,initialValue:0}]
const port = (extra = {}) => ({id:10,target:1,isShow:true,label:'再来一轮',conditionMode:'all',conditions:[],effects:[],...extra})
const graph = p => ({1:{hash:'root',name:'开场',html:'a.html',options:[p]}})
const copy = value => JSON.parse(JSON.stringify(value))

test('all/any comparisons and unconditional ports', () => {
  const r = createRuntime(graph(port()), vars, 1)
  for (const [operator, value, expected] of [['>=',3,true],['<=',2,false],['==',3,true],['!=',3,false],['>',2,true],['<',4,true]]) {
    assert.equal(r.eligible(port({conditions:[{variableId:1,operator,value}]}),r.initial().values),expected)
  }
  const conditions = [{variableId:1,operator:'>',value:1},{variableId:2,operator:'>',value:1}]
  assert.equal(r.eligible(port({conditions}), r.initial().values), false)
  assert.equal(r.eligible(port({conditions,conditionMode:'any'}), r.initial().values), true)
  assert.equal(r.eligible(port({conditionMode:'any'}), r.initial().values), true)
})

test('conditions precede clear and add effects, and unconfigured variables are retained', () => {
  const p = port({conditions:[{variableId:1,operator:'>=',value:3}],effects:[{variableId:1,set:1,delta:0},{variableId:2,set:0,delta:1}]})
  const r = createRuntime(graph(p), vars, 1)
  const original = r.initial()
  const next = r.transition(original,10)
  assert.deepEqual(copy(next.values), {'1':0,'2':1})
  assert.deepEqual(copy(original.values), {'1':3,'2':0})
  assert.equal(r.validState(next),true)
  assert.throws(() => r.transition(next,10), /条件/)
  const corrupt = copy(next); corrupt.values['2'] = 9
  assert.equal(r.validState(corrupt), false)
  corrupt.values['2'] = 1; corrupt.history[0].target = 'missing'
  assert.equal(r.validState(corrupt),false)
  p.conditions = []
  assert.deepEqual(copy(r.transition(next,10).values),{'1':0,'2':2})
})

test('overflow fails atomically and long looping history validates', () => {
  const p = port({effects:[{variableId:1,set:0,delta:1},{variableId:2,set:0,delta:Number.MAX_SAFE_INTEGER}]})
  const r = createRuntime(graph(p),vars,1)
  const state = r.transition(r.initial(),10)
  const before = copy(state)
  assert.throws(() => r.transition(state,10),/安全整数/)
  assert.deepEqual(copy(state),before)
  p.effects = [{variableId:2,set:0,delta:1}]
  let loop = r.initial()
  for(let i=0;i<1000;i++) loop = r.transition(loop,10)
  assert.equal(r.validState(loop),true)
  assert.equal(loop.values[2],1000)
})

test('clear uses the value at runtime, retains unrelated variables and rejects invalid effects atomically', () => {
  const p = port({effects:[{variableId:1,set:1,delta:0}]})
  for (const n of [0,3,8,-5,Number.MAX_SAFE_INTEGER,-Number.MAX_SAFE_INTEGER]) {
    const r = createRuntime(graph(p),[{id:1,initialValue:n},{id:2,initialValue:9}],1)
    assert.deepEqual(copy(r.transition(r.initial(),10).values),{'1':0,'2':9})
  }
  const r = createRuntime(graph(p),vars,1)
  const state = r.initial()
  p.effects.push({variableId:2,set:2,delta:0})
  assert.throws(() => r.transition(state,10),/set/)
  assert.deepEqual(copy(state.values),{'1':3,'2':0})
  p.effects = [{variableId:1,set:-1,delta:1}]
  assert.throws(() => r.transition(state,10),/set/)
})

test('set assigns any safe target value and does not add to the current value', () => {
  for (const target of [2,0,-3,Number.MAX_SAFE_INTEGER,-Number.MAX_SAFE_INTEGER]) {
    const p = port({effects:[{variableId:1,set:1,delta:target}]})
    const r = createRuntime(graph(p),[{id:1,initialValue:Number.MAX_SAFE_INTEGER}],1)
    const next = r.transition(r.initial(),10)
    assert.equal(next.values[1],target)
    assert.equal(r.validState(next),true)
    assert.equal(r.transition(next,10).values[1],target)
  }
  const p = port({effects:[{variableId:1,set:1,delta:Number.MAX_SAFE_INTEGER+1}]})
  const r = createRuntime(graph(p),vars,1)
  assert.throws(() => r.transition(r.initial(),10),/效果/)
})

function player(p, storage = new Map(), revision = 'v1') {
  const elements = new Map()
  const windowEvents = {}
  function element() {
    return {style:{},textContent:'',children:[],handlers:{},contentWindow:{postMessage(){}},
      addEventListener(name, fn){this.handlers[name]=fn},setAttribute(){},remove(){},
      appendChild(child){this.children.push(child)},replaceChildren(){this.children=[]}}
  }
  const document = {getElementById(id){if(!elements.has(id)) elements.set(id,element()); return elements.get(id)},createElement:element,addEventListener(){}}
  document.getElementById('story-data').textContent=JSON.stringify(graph(p))
  document.getElementById('story-meta').textContent=JSON.stringify({variables:vars,projectKey:'test',revision})
  const ctx = {document,window:{addEventListener(name,fn){windowEvents[name]=fn}},history:{replaceState(){}},confirm:()=>true,
    localStorage:{getItem:key=>storage.get(key),setItem:(key,value)=>storage.set(key,value)}}
  const script = template.slice(template.indexOf('<script>')+8,template.indexOf('</script>',template.indexOf('<script>')))
    .replace('__STORY_RUNTIME__',source).replace('__FIRST_NODE__','1')
  runInNewContext(script,ctx)
  return {ctx,storage,document,windowEvents,click:id=>document.getElementById(id).handlers.click()}
}

test('player guards duplicate/stale clicks and resumes without replaying effects', () => {
  const p = port({effects:[{variableId:1,set:1,delta:0},{variableId:2,set:0,delta:1}]})
  const game=player(p)
  game.click('start')
  game.ctx.afterContent()
  const oldButton=game.document.getElementById('opts').children[0]
  oldButton.onclick();oldButton.onclick()
  assert.equal(game.ctx.state.values[2],1)
  game.ctx.afterContent();oldButton.onclick()
  assert.equal(game.ctx.state.values[2],1)
  const resumed=player(p,game.storage)
  assert.ok(resumed.ctx.saved)
  resumed.click('continue')
  assert.equal(resumed.ctx.state.values[1],0)
  assert.equal(resumed.ctx.state.values[2],1)
  assert.equal(resumed.ctx.state.history.length,1)
  resumed.windowEvents.popstate();resumed.windowEvents.hashchange()
  assert.equal(resumed.ctx.state.values[2],1)
  resumed.ctx.afterContent()
  resumed.document.getElementById('opts').children[0].onclick()
  assert.equal(resumed.ctx.state.values[2],2)
  resumed.click('restart')
  assert.deepEqual(copy(resumed.ctx.state.values),{'1':3,'2':0})
  assert.equal(resumed.ctx.state.history.length,0)
})

test('player rejects mismatched or damaged saves, and storage failure does not stop play', () => {
  const game=player(port());game.click('start')
  assert.equal(player(port(),game.storage,'v2').ctx.saved,null)
  game.storage.set('galgame-play-v1-test','broken json')
  assert.equal(player(port(),game.storage).ctx.saved,null)
  game.ctx.localStorage.setItem=()=>{throw new Error('full')}
  game.click('restart')
  assert.equal(game.ctx.started,true)
  assert.match(game.document.getElementById('save-notice').textContent,/无法保存/)
})
