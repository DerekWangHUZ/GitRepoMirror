// 运行日志视图：迁移自 templates.js 的 logsView/logLine。
import { LogLine } from "@/components/LogLine";
import { Button } from "@/components/ui/button";

export function LogsView({ logs, onClear }) {
  return (
    <>
      <div className="grm-page-head">
        <div>
          <h1>运行日志</h1>
          <p>实时查看 Git 与平台 CLI 输出</p>
        </div>
        <Button variant="outline" onClick={onClear}>
          清空日志
        </Button>
      </div>
      <div className="grm-log-panel">
        {logs.length ? (
          logs.map((log, index) => <LogLine key={index} log={log} />)
        ) : (
          <div className="grm-log-empty">命令输出将在这里显示</div>
        )}
      </div>
    </>
  );
}
