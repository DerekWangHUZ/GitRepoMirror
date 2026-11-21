// 单行日志：迁移自 templates.js 的 logLine。
export function LogLine({ log }) {
  return (
    <div className={`grm-log-line ${log.level || "info"}`}>
      <time>{log.time}</time>
      <span>{log.message}</span>
    </div>
  );
}
