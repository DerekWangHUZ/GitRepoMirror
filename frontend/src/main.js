import "./style.css";
import { backend, EventsOn } from "./backend.js";
import { createActions } from "./ui/actions.js";
import { createRenderer } from "./ui/renderer.js";
import { shellTemplate } from "./ui/shell.js";
import { appendLog, createState } from "./ui/state.js";

const app = document.querySelector("#app");
const state = createState();
app.innerHTML = shellTemplate();

const renderer = createRenderer(state);
const actions = createActions(state, backend, renderer);
actions.bind();

EventsOn("log", (log) => {
  appendLog(state, log);
  renderer.consoleOutput();
  if (state.view === "logs") renderer.logs();
});
EventsOn("progress", renderer.progress);
EventsOn("repositories-changed", actions.refreshData);

await Promise.all([actions.refreshData(), actions.checkEnvironment()]);
renderer.consoleOutput();
