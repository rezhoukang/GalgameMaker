// 纯剧情状态机：只有经过端口才能修改数值；恢复页面不重放效果。
function createStoryRuntime(nodes, variables, first) {
  var definitions = Object.create(null);
  variables.forEach(function(v) { definitions[String(v.id)] = v; });
  function defaults() {
    var values = {};
    variables.forEach(function(v) { values[String(v.id)] = v.initialValue; });
    return values;
  }
  function validValues(values) {
    return values && typeof values === 'object' && !Array.isArray(values) &&
      Object.keys(values).length === variables.length && variables.every(function(v) { return Number.isSafeInteger(values[String(v.id)]); });
  }
  function initial() { return { currentNode: String(first), values: defaults(), history: [] }; }
  function condition(c, values) {
    var n = values[String(c.variableId)];
    if (!definitions[String(c.variableId)] || !Number.isSafeInteger(n) || !Number.isSafeInteger(c.value)) return false;
    switch (c.operator) {
      case '>=': return n >= c.value;
      case '<=': return n <= c.value;
      case '==': return n === c.value;
      case '!=': return n !== c.value;
      case '>': return n > c.value;
      case '<': return n < c.value;
      default: return false;
    }
  }
  function eligible(port, values) {
    if (port.conditionMode !== 'all' && port.conditionMode !== 'any') return false;
    if (!port.conditions.length) return true;
    return port.conditionMode === 'any' ? port.conditions.some(function(c) { return condition(c,values); }) : port.conditions.every(function(c) { return condition(c,values); });
  }
  function transition(state, portId) {
    if (!state || !nodes[String(state.currentNode)] || !validValues(state.values)) throw new Error('剧情状态无效');
    var port = nodes[String(state.currentNode)].options.find(function(p) { return p.id === portId; });
    if (!port || !nodes[String(port.target)] || !eligible(port,state.values)) throw new Error('当前选项条件不满足');
    var values = Object.assign({}, state.values);
    var seen = Object.create(null);
    port.effects.forEach(function(e) {
      var key = String(e.variableId);
      if (!definitions[key] || seen[key] || !Number.isSafeInteger(e.delta)) throw new Error('变量效果无效');
      seen[key] = true;
      if (e.operation !== 'add' && e.operation !== 'clear') throw new Error('变量操作无效');
      if (e.operation === 'clear' && e.delta !== 0) throw new Error('清空操作的变化量必须为 0');
      // 清空由播放器在经过端口时执行，等价于 delta = -当前值。
      var next = e.operation === 'clear' ? 0 : values[key] + e.delta;
      if (!Number.isSafeInteger(next)) throw new Error('变量数值超出安全整数范围');
      values[key] = next;
    });
    return { currentNode: String(port.target), values: values, history: state.history.concat([{from: String(state.currentNode), portId: port.id, target: String(port.target)}]) };
  }
  function validState(state) {
    if (!state || !nodes[String(state.currentNode)] || !validValues(state.values) || !Array.isArray(state.history)) return false;
    // 按路径重算并核对数值，拒绝损坏、断裂或条件不成立的存档。
    var replay = initial();
    try {
      for (var i = 0; i < state.history.length; i++) {
        var step = state.history[i];
        if (!step || step.from !== replay.currentNode) return false;
        // 验证只需要当前状态，避免反复复制越来越长的历史。
        replay.history = [];
        replay = transition(replay,step.portId);
        if (step.target !== replay.currentNode) return false;
      }
    } catch (_) { return false; }
    return replay.currentNode === String(state.currentNode) && variables.every(function(v) { return replay.values[String(v.id)] === state.values[String(v.id)]; });
  }
  return { initial: initial, eligible: eligible, transition: transition, validState: validState };
}
