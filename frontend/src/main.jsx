import { StrictMode } from "react";
import { createRoot } from "react-dom/client";

import App from "./App";
import "./index.css";

const root = document.getElementById("app");
if (!root) throw new Error("找不到 #app 挂载点");

createRoot(root).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
