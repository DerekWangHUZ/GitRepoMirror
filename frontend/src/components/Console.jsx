// 底部可折叠实时控制台：迁移自 shell.js 的 console 区块。
import { useEffect, useRef } from "react";
import { Terminal } from "lucide-react";

import { LogLine } from "@/components/LogLine";
import { cn } from "@/lib/utils";

export function Console({ logs, busyCount, collapsed, onToggle }) {
  const bodyRef = useRef(null);
  const items = logs.slice(-200);

  // 新日志到达时贴底滚动（沿用旧 renderer.consoleOutput 的行为）
  useEffect(() => {
    const body = bodyRef.current;
    if (body) body.scrollTop = body.scrollHeight;
  }, [logs]);

  return (
    <section
      className={cn("grm-console", collapsed && "collapsed")}
      id="console"
      aria-label="实时输出"
    >
      <button
        type="button"
        className="grm-console-head"
        id="consoleToggle"
        onClick={onToggle}
      >
        <span className="grm-console-title">
          <Terminal />
          实时输出
        </span>
        <span id="consoleStatus">
          {busyCount ? `${busyCount} 个任务运行中` : "空闲"}
        </span>
        <i>⌃</i>
      </button>
      <div className="grm-console-body" id="consoleBody" ref={bodyRef}>
        {items.length ? (
          items.map((log, index) => <LogLine key={index} log={log} />)
        ) : (
          <div className="grm-log-empty">等待任务开始…</div>
        )}
      </div>
    </section>
  );
}
