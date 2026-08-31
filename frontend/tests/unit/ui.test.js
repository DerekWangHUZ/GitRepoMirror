import { describe, expect, it } from "vitest";
import { appendLog, createState } from "../../src/ui/state.js";
import { createRenderer } from "../../src/ui/renderer.js";
import {
  environmentRow,
  repositoryCard,
  settingsView,
  targetFields,
} from "../../src/ui/templates.js";
import { escapeHTML, formatTime, platformLabel } from "../../src/ui/utils.js";

describe("UI contracts", () => {
  it("escapes untrusted values and preserves platform labels", () => {
    expect(escapeHTML(`<script a="b">'&`)).toBe(
      "&lt;script a=&quot;b&quot;&gt;&#39;&amp;",
    );
    expect(platformLabel("github")).toBe("GitHub");
    expect(platformLabel("custom")).toBe("Git");
    expect(formatTime("0001-01-01T00:00:00Z")).toBe("尚未同步");
  });

  it("caps logs at one thousand entries", () => {
    const state = createState();
    for (let index = 0; index < 1005; index += 1)
      appendLog(state, { message: String(index) });
    expect(state.logs).toHaveLength(1000);
    expect(state.logs[0].message).toBe("5");
  });

  it("renders the empty log state during initial data rendering", () => {
    document.body.innerHTML = `
      <span id="repoCount"></span>
      <div id="repositoriesView" class="view"></div>
      <div id="logsView" class="view hidden"></div>
      <div id="settingsView" class="view hidden"></div>
    `;
    const renderer = createRenderer(createState());

    renderer.data();

    expect(document.querySelector("#logsView").textContent).toContain(
      "命令输出将在这里显示",
    );
  });

  it.each(["healthy", "syncing", "failed"])(
    "renders the %s repository state",
    (status) => {
      const html = repositoryCard(
        {
          id: "repo",
          status,
          mode: "mirror",
          source: { platform: "github", displayName: "<source>" },
          target: { platform: "gitlab", displayName: "target" },
          lastError: status === "failed" ? "<failure>" : "",
        },
        new Set(),
      );
      expect(html).toContain(`repo-card ${status}`);
      expect(html).toContain("&lt;source&gt;");
      if (status === "failed") expect(html).toContain("&lt;failure&gt;");
    },
  );

  it("renders environment authentication details", () => {
    expect(
      environmentRow(
        "GitHub CLI",
        { installed: true, authenticated: false, authState: "invalid" },
        "github",
      ),
    ).toContain("凭据失效");
  });

  it("switches target fields between managed and URL modes", () => {
    const managed = targetFields("github", {
      github: { installed: true, authenticated: true, login: "owner" },
    });
    expect(managed.managed).toBe(true);
    expect(managed.html).toContain("targetNamespace");
    const fallback = targetFields("gitlab", {
      gitlab: { installed: false, authenticated: false },
    });
    expect(fallback.managed).toBe(false);
    expect(fallback.html).toContain("targetUrl");
  });

  it("preserves settings defaults and values", () => {
    const html = settingsView({
      concurrency: 3,
      syncLfs: true,
      proxy: "http://127.0.0.1:7890",
      prefix: "mirror-",
    });
    expect(html).toContain("http://127.0.0.1:7890");
    expect(html).toContain("<option selected>3</option>");
    expect(html).toContain('name="syncLfs" checked');
  });
});
