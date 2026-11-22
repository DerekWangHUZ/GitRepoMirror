import { StrictMode } from "react";
import { createRoot } from "react-dom/client";

import App from "./App";
import "./index.css";

// 桌面版窗口接线（模板 desktop/src/skin/desktop.js）：is-desktop、
// 红绿灯事件委托、22px 拖拽带、双击最大化、is-maximized 同步、外链浏览器。
// Vite 下用 ?url 引入，让浏览器直接执行这份独立脚本，不进打包管道。
import desktopSkinUrl from "./skin/desktop.js?url";
const desktopSkin = document.createElement("script");
desktopSkin.src = desktopSkinUrl;
desktopSkin.type = "module";
document.head.appendChild(desktopSkin);

const root = document.getElementById("app");
if (!root) throw new Error("找不到 #app 挂载点");

createRoot(root).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
