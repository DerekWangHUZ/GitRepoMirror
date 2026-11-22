// 日志与进度事件模型（迁移自 src/ui/state.js，逻辑不变；上限 MAX_LOGS 条保留）。

export const MAX_LOGS = 1000;

export function createLogs() {
  return [];
}

export function appendLog(logs, log) {
  logs.push(log);
  if (logs.length > MAX_LOGS) logs.splice(0, logs.length - MAX_LOGS);
}
