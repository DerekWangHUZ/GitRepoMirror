// 来源：macUI-SolidGlass 模板本机副本（web/src/macos/traffic-lights.tsx），复制日期 2026-11-22。
// token 禁止手工微调 —— 样式由 skin/macos/chrome.css 的 .traffic/.light 提供。
import { cn } from "@/lib/utils";

/**
 * 窗口控件：红绿灯。
 * 13px 直径、间距 8px、悬停显字形、灯圆心嵌进窗口 32px 圆角弧。
 * 按钮带 `data-win="close|min|max"`：桌面版（Wails）通过 document 事件
 * 委托（skin/desktop.js）接管为真正的系统窗口行为；onClick 桥接
 * Close/Minimise/ToggleMaximise 作为绑定层回退。
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
        data-win="close"
        aria-label="关闭"
        title="关闭"
        onClick={onClose}
      />
      <button
        type="button"
        className="light light--min"
        data-win="min"
        aria-label="最小化"
        title="最小化"
        onClick={onMinimize}
      />
      <button
        type="button"
        className="light light--max"
        data-win="max"
        aria-label={maximized ? "还原" : "最大化"}
        title={maximized ? "还原" : "最大化"}
        onClick={onMaximize}
      />
    </div>
  );
}
