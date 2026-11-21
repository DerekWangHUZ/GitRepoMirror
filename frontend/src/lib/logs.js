// 日志与进度事件模型（迁移自 src/ui/state.js，逻辑不变；上限 1000 条保留）。

export function createLogs() {
  return [];
}

export function appendLog(logs, log) {
  logs.push(log);
  if (logs.length > 1000) logs.splice(0, logs.length - 1000);
}
