import { afterAll, describe, expect, it } from "vitest";
import { cleanup, render, screen } from "@testing-library/react";

import { appendLog, createLogs } from "@/lib/logs";
import {
  environmentRowModel,
  formatTime,
  loginButtonLabel,
  managedPlatforms,
  platformLabel,
  targetFieldsModel,
} from "@/lib/utils";
import { LogLine } from "@/components/LogLine";
import { LogsView } from "@/components/LogsView";
import { PlatformIcon } from "@/components/PlatformIcon";
import { RepositoriesView } from "@/components/RepositoriesView";
import { TrafficLights } from "@/components/TrafficLights";

describe("UI contracts", () => {
  afterAll(cleanup);

  it("preserves platform labels", () => {
    expect(platformLabel("github")).toBe("GitHub");
    expect(platformLabel("gitee")).toBe("Gitee");
    expect(platformLabel("codeberg")).toBe("Codeberg");
    expect(platformLabel("azuredevops")).toBe("Azure DevOps");
    expect(platformLabel("custom")).toBe("Git");
    expect(formatTime("0001-01-01T00:00:00Z")).toBe("尚未同步");
  });

  it("exposes every managed platform with a CLI name", () => {
    const platforms = Object.fromEntries(
      managedPlatforms().map((platform) => [platform.id, platform.cli]),
    );
    expect(platforms).toEqual({
      github: "gh",
      gitlab: "glab",
      gitee: "gitee",
      codeberg: "fj",
      azuredevops: "az",
    });
  });

  it("caps logs at one thousand entries", () => {
    const logs = createLogs();
    for (let index = 0; index < 1005; index += 1)
      appendLog(logs, { message: String(index) });
    expect(logs).toHaveLength(1000);
    expect(logs[0].message).toBe("5");
  });

  it("renders the empty log state", () => {
    render(<LogsView logs={[]} onClear={() => {}} />);
    expect(screen.getByText("命令输出将在这里显示")).toBeTruthy();
  });

  it("renders log levels as class names", () => {
    const { container } = render(
      <>
        <LogLine
          log={{ time: "08:00:00", level: "command", message: "git fetch" }}
        />
        <LogLine log={{ time: "08:00:01", level: "stderr", message: "boom" }} />
      </>,
    );
    expect(container.querySelector(".grm-log-line.command")).toBeTruthy();
    expect(container.querySelector(".grm-log-line.stderr")).toBeTruthy();
  });

  it.each(["healthy", "syncing", "failed"])(
    "renders the %s repository state",
    (status) => {
      render(
        <RepositoriesView
          repositories={[
            {
              id: "repo",
              status,
              mode: "mirror",
              source: { platform: "github", displayName: "<source>" },
              target: { platform: "gitlab", displayName: "target" },
              lastError: status === "failed" ? "<failure>" : "",
            },
          ]}
          busy={new Set()}
          onSyncAll={() => {}}
          onNewMirror={() => {}}
          onSyncOne={() => {}}
          onRemove={() => {}}
          onOpenURL={() => {}}
        />,
      );
      // React 文本注入不可执行（children 不是 innerHTML），直接断言文本内容。
      expect(screen.getByText("<source>")).toBeTruthy();
      expect(document.querySelector(".grm-repo-card")).toBeTruthy();
      expect(document.querySelector(`.grm-status.${status}`)).toBeTruthy();
      if (status === "failed")
        expect(screen.getByText("<failure>")).toBeTruthy();
      cleanup();
    },
  );

  it("renders environment authentication details", () => {
    expect(
      environmentRowModel(
        "GitHub CLI",
        { installed: true, authenticated: false, authState: "invalid" },
        "github",
      ).detail,
    ).toBe("凭据失效");
    expect(
      loginButtonLabel({ installed: true, authState: "missing" }, "gitlab"),
    ).toBe("重新登录 GitLab");
  });

  it("disables batch sync while any repository operation is busy", () => {
    render(
      <RepositoriesView
        repositories={[
          {
            id: "repo",
            status: "healthy",
            mode: "mirror",
            source: { platform: "generic", displayName: "source" },
            target: { platform: "generic", displayName: "target" },
          },
        ]}
        busy={new Set(["repo"])}
        onSyncAll={() => {}}
        onNewMirror={() => {}}
        onSyncOne={() => {}}
        onRemove={() => {}}
        onOpenURL={() => {}}
      />,
    );
    // S8 工具栏范式：动作集中在 .toolbar__actions（原 grm-head-actions）。
    const syncAllButton = screen
      .getAllByRole("button", { name: /全部同步/ })
      .find((button) => button.closest(".toolbar__actions"));
    expect(syncAllButton).toBeTruthy();
    expect(syncAllButton.disabled).toBe(true);
    const syncOneButton = screen
      .getAllByRole("button", { name: /立即同步|同步中/ })
      .find((button) => button.closest(".grm-repo-actions"));
    expect(syncOneButton).toBeTruthy();
    expect(syncOneButton.disabled).toBe(true);
    cleanup();
  });

  it("aligns the shell contract: toolbar title + page subtitle + section title", () => {
    render(
      <RepositoriesView
        repositories={[]}
        busy={new Set()}
        onSyncAll={() => {}}
        onNewMirror={() => {}}
        onSyncOne={() => {}}
        onRemove={() => {}}
        onOpenURL={() => {}}
      />,
    );
    expect(document.querySelector(".toolbar__title")).toBeTruthy();
    expect(document.querySelector(".toolbar__actions")).toBeTruthy();
    expect(document.querySelector(".page-subtitle")).toBeTruthy();
    cleanup();
    render(
      <RepositoriesView
        repositories={[
          {
            id: "repo",
            status: "healthy",
            mode: "mirror",
            source: { platform: "github", displayName: "source" },
            target: { platform: "gitlab", displayName: "target" },
          },
        ]}
        busy={new Set()}
        onSyncAll={() => {}}
        onNewMirror={() => {}}
        onSyncOne={() => {}}
        onRemove={() => {}}
        onOpenURL={() => {}}
      />,
    );
    // 仓库分节用 section-title（S8 页面范式）。
    expect(document.querySelector(".section-title")).toBeTruthy();
    cleanup();
  });

  it("switches target fields between managed and URL modes", () => {
    const managed = targetFieldsModel("github", {
      github: { installed: true, authenticated: true, login: "owner" },
    });
    expect(managed.managed).toBe(true);
    expect(managed.mode).toBe("managed");
    expect(managed.defaultNamespace).toBe("owner");

    const fallback = targetFieldsModel("gitlab", {
      gitlab: { installed: false, authenticated: false },
    });
    expect(fallback.managed).toBe(false);
    expect(fallback.mode).toBe("url");
    expect(fallback.message).toContain("glab 缺失或未登录");
  });

  it("adapts target fields for the new platforms", () => {
    const gitee = targetFieldsModel("gitee", {
      gitee: { installed: true, authenticated: true, login: "gitee-user" },
    });
    expect(gitee.managed).toBe(true);
    expect(gitee.defaultNamespace).toBe("gitee-user");
    expect(gitee.fixedVisibility).toBe(false);

    const azure = targetFieldsModel("azuredevops", {
      azuredevops: {
        installed: true,
        authenticated: true,
        login: "user@example.com",
      },
    });
    expect(azure.managed).toBe(true);
    expect(azure.namespaceHint).toBe("组织/项目");
    expect(azure.fixedVisibility).toBe(true);

    const codebergFallback = targetFieldsModel("codeberg", {
      codeberg: { installed: false, authenticated: false },
    });
    expect(codebergFallback.managed).toBe(false);
    expect(codebergFallback.message).toContain("fj 缺失或未登录");
  });

  it("renders the traffic lights with window actions", () => {
    const calls = [];
    render(
      <TrafficLights
        onClose={() => calls.push("close")}
        onMinimize={() => calls.push("min")}
        onMaximize={() => calls.push("max")}
      />,
    );
    screen.getByRole("button", { name: "关闭" }).click();
    screen.getByRole("button", { name: "最小化" }).click();
    screen.getByRole("button", { name: "最大化" }).click();
    expect(calls).toEqual(["close", "min", "max"]);
  });

  it("exposes data-win handles for the desktop event delegation", () => {
    render(
      <TrafficLights
        onClose={() => {}}
        onMinimize={() => {}}
        onMaximize={() => {}}
        maximized
      />,
    );
    // S5：桌面接线（skin/desktop.js）按 data-win 委托接管红绿灯。
    expect(document.querySelector('[data-win="close"]')).toBeTruthy();
    expect(document.querySelector('[data-win="min"]')).toBeTruthy();
    expect(document.querySelector('[data-win="max"]')).toBeTruthy();
    // 最大化时绿色灯给出还原语义。
    expect(screen.getByRole("button", { name: "还原" })).toBeTruthy();
  });

  it("covers the 8 Apple accent options with graphite included", async () => {
    const { ACCENT_OPTIONS } = await import("@/lib/theme");
    const ids = ACCENT_OPTIONS.map((option) => option.id);
    expect(ids).toEqual([
      "blue",
      "purple",
      "pink",
      "red",
      "orange",
      "yellow",
      "green",
      "graphite",
    ]);
  });

  it("renders platform icons as inert svg", () => {
    const { container } = render(<PlatformIcon platform="github" />);
    const svg = container.querySelector("svg");
    expect(svg).toBeTruthy();
    expect(svg.getAttribute("fill")).toBe("currentColor");
  });
});
