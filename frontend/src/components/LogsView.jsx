// 运行日志视图：迁移自 templates.js 的 logsView/logLine。
import { LogLine } from "@/components/LogLine";
import { Button } from "@/components/ui/button";

export function LogsView({ logs, onClear }) {
  return (
    <>
      {/* 工具栏范式：52px leading 标题 + trailing 动作（chrome.css .toolbar） */}
      <div className="toolbar">
        <div className="toolbar__heading">
          <h1 className="toolbar__title">运行日志</h1>
        </div>
        <div className="toolbar__actions">
          <Button variant="outline" onClick={onClear}>
            清空日志
          </Button>
        </div>
      </div>
      <p className="page-subtitle">实时查看 Git 与平台 CLI 输出</p>
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
