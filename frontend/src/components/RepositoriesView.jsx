// 镜像仓库视图：迁移自 templates.js 的 repositoryView/repositoryCard/emptyState。
import { ExternalLink, Plus, RefreshCw } from "lucide-react";

import { PlatformIcon } from "@/components/PlatformIcon";
import { Button } from "@/components/ui/button";
import { cn, formatTime, platformLabel, STATUS_LABELS } from "@/lib/utils";

export function RepositoriesView({
  repositories,
  busy,
  onSyncAll,
  onNewMirror,
  onSyncOne,
  onRemove,
  onOpenURL,
}) {
  return (
    <>
      <div className="grm-page-head">
        <div>
          <h1>镜像仓库</h1>
          <p>在不同 Git 托管平台之间保存独立副本</p>
        </div>
        <div className="grm-head-actions">
          <Button
            variant="outline"
            disabled={!repositories.length || busy.size > 0}
            onClick={onSyncAll}
          >
            <RefreshCw />
            全部同步
          </Button>
          <Button onClick={onNewMirror}>
            <Plus />
            新建镜像
          </Button>
        </div>
      </div>
      {repositories.length ? (
        <div className="grm-repo-grid">
          {repositories.map((repo) => (
            <RepositoryCard
              key={repo.id}
              repo={repo}
              busy={busy}
              onSyncOne={onSyncOne}
              onRemove={onRemove}
              onOpenURL={onOpenURL}
            />
          ))}
        </div>
      ) : (
        <EmptyState onNewMirror={onNewMirror} />
      )}
    </>
  );
}

function RepositoryCard({ repo, busy, onSyncOne, onRemove, onOpenURL }) {
  const status = repo.status || "healthy";
  const isBusy = busy.has(repo.id) || status === "syncing";
  return (
    <article className={cn("grm-repo-card", status === "failed" && "failed")}>
      <div className="grm-repo-top">
        <div className="grm-repo-icon" data-platform={repo.source.platform}>
          <PlatformIcon platform={repo.source.platform} size={23} />
        </div>
        <div className="grm-repo-title">
          <h3>{repo.source.displayName}</h3>
          {repo.target.webUrl ? (
            <button
              type="button"
              className="grm-repo-target"
              onClick={() => onOpenURL(repo.target.webUrl)}
            >
              {repo.target.displayName}
              <ExternalLink />
            </button>
          ) : (
            <span className="grm-repo-target">{repo.target.displayName}</span>
          )}
        </div>
        <span className={cn("grm-status", status)}>
          <i className="grm-dot" />
          {STATUS_LABELS[status] || status}
        </span>
      </div>
      <div className="grm-repo-meta">
        <span>
          平台
          <b>
            {platformLabel(repo.source.platform)} →{" "}
            {platformLabel(repo.target.platform)}
          </b>
        </span>
        <span>
          模式<b>{repo.mode === "shallow" ? "浅克隆" : "完整镜像"}</b>
        </span>
        <span>
          上次同步<b>{formatTime(repo.lastSync)}</b>
        </span>
      </div>
      {repo.lastError ? (
        <div className="grm-error-line" title={repo.lastError}>
          {repo.lastError}
        </div>
      ) : null}
      <div className="grm-repo-actions">
        <Button
          variant="outline"
          className="grm-sync-button"
          disabled={isBusy}
          onClick={() => onSyncOne(repo.id)}
        >
          <RefreshCw />
          {isBusy ? "同步中…" : "立即同步"}
        </Button>
        <Button
          variant="ghost"
          size="icon"
          aria-label="解除绑定"
          title="解除绑定"
          onClick={() => onRemove(repo.id)}
        >
          <TrashGlyph />
        </Button>
      </div>
    </article>
  );
}

function TrashGlyph() {
  return (
    <svg
      viewBox="0 0 24 24"
      width={16}
      height={16}
      aria-hidden="true"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.8"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      <path d="M4 7h16M9 7V4h6v3M7 7l1 13h8l1-13M10 11v5M14 11v5" />
    </svg>
  );
}

function EmptyState({ onNewMirror }) {
  return (
    <div className="grm-empty">
      <div className="grm-empty-visual">
        <div>
          <PlatformIcon platform="git" size={42} />
        </div>
        <span>
          <RefreshCw />
        </span>
      </div>
      <h2>还没有镜像仓库</h2>
      <p>创建第一个仓库镜像，保存分支、Tag 和提交历史。</p>
      <Button onClick={onNewMirror}>
        <Plus />
        创建第一个镜像
      </Button>
    </div>
  );
}
