// 来源：macUI-SolidGlass 模板本机副本（web/src/macos/traffic-lights.tsx），复制日期 2026-11-22。
// token 禁止手工微调 —— 样式由 skin/macos/chrome.css 的 .traffic/.light 提供。
import { cn } from "@/lib/utils";

/**
 * 窗口控件：红绿灯。
 * 13px 直径、间距 8px、悬停显字形。
 * GitRepoMirror：onClick 直接桥接 Wails 的 Close/Minimise/ToggleMaximise。
 */
export function TrafficLights({
  onClose,
  onMinimize,
  onMaximize,
  maximized,
  className,
}) {
  return (
    <div className={cn("traffic", className)} data-wails-no-drag>
      <button
        type="button"
        className="light light--close"
        aria-label="关闭"
        title="关闭"
        onClick={onClose}
      />
      <button
        type="button"
        className="light light--min"
        aria-label="最小化"
        title="最小化"
        onClick={onMinimize}
      />
      <button
        type="button"
        className="light light--max"
        aria-label={maximized ? "还原" : "最大化"}
        title={maximized ? "还原" : "最大化"}
        onClick={onMaximize}
      />
    </div>
  );
}
