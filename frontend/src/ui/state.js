export const createState = () => ({
  data: { repositories: [], settings: { concurrency: 2 } },
  env: {},
  logs: [],
  progress: {},
  busy: new Set(),
  view: "repositories",
});

export function appendLog(state, log) {
  state.logs.push(log);
  if (state.logs.length > 1000) state.logs.splice(0, state.logs.length - 1000);
}
