// 创建镜像 Modal：迁移自 shell.js 的 mirrorForm + templates.js 的 targetFields。
// 受控表单；目标平台切换托管/URL 两种模式（targetFieldsModel）。
// 打开即重置：父组件用 key={String(open)} 重新挂载本组件实现，
// 避免在 effect 里同步 setState（react-hooks lint 红线）。
import { useMemo, useState } from "react";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { cn, PLATFORMS, targetFieldsModel } from "@/lib/utils";

const MODE_ITEMS = [
  { label: "完整镜像 · 所有分支与 Tag", value: "mirror" },
  { label: "浅克隆 · 默认分支最新版本", value: "shallow" },
];

export function CreateMirrorDialog({
  open,
  onOpenChange,
  env,
  defaults,
  progress,
  submitting,
  onSubmit,
  onLoginPlatform,
}) {
  const [source, setSource] = useState("");
  const [targetPlatform, setTargetPlatform] = useState("github");
  const [mode, setMode] = useState("mirror");
  const [prefix, setPrefix] = useState(defaults?.prefix || "");
  const [suffix, setSuffix] = useState(defaults?.suffix || "");
  const [syncLfs, setSyncLfs] = useState(Boolean(defaults?.syncLfs));
  // managed 模式：命名空间 / 自定义名称 / 可见性 / 冲突策略
  // url 模式：完整推送 URL
  const [targetNamespace, setTargetNamespace] = useState(null);
  const [targetName, setTargetName] = useState("");
  const [visibility, setVisibility] = useState("private");
  const [conflictPolicy, setConflictPolicy] = useState("error");
  const [targetUrl, setTargetUrl] = useState("");

  const target = useMemo(
    () => targetFieldsModel(targetPlatform, env),
    [targetPlatform, env],
  );

  // 托管平台的默认命名空间 = 平台 CLI 登录名；用户编辑后以用户值为准。
  const effectiveNamespace =
    targetNamespace ?? (target.managed ? target.defaultNamespace || "" : "");
  const submit = (event) => {
    event.preventDefault();
    const request = {
      source,
      targetPlatform,
      mode,
      prefix,
      suffix,
      syncLfs,
      visibility,
      conflictPolicy,
    };
    if (target.managed) {
      request.targetNamespace = effectiveNamespace;
      request.targetName = targetName;
    } else {
      request.targetUrl = targetUrl;
    }
    onSubmit(request);
  };

  const platformItems = PLATFORMS.map((platform) => ({
    label: platform.label,
    value: platform.id,
  }));
  const stepState = (index) => {
    const step = progress?.step || 0;
    if (!open) return "";
    if (step && index + 1 < step) return "done";
    if (step && index + 1 === step) return progress?.status || "active";
    return index === 0 ? "active" : "";
  };

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!next && submitting) return;
        onOpenChange(next);
      }}
    >
      <DialogContent className="grm-dialog-content">
        <DialogHeader>
          <DialogTitle>创建仓库镜像</DialogTitle>
          <DialogDescription>
            在代码托管平台之间建立独立镜像。
          </DialogDescription>
        </DialogHeader>
        <form id="mirrorForm" onSubmit={submit}>
          <label className="grm-label">
            源仓库 <span>GitHub 可用 owner/repo，其他平台使用完整 URL</span>
            <Input
              name="source"
              required
              className="mt-2"
              value={source}
              onChange={(event) => setSource(event.target.value)}
              placeholder="https://gitlab.com/team/project.git"
              autoComplete="off"
              autoFocus
            />
          </label>
          <div className="grm-two-col mt-4">
            <label className="grm-label" htmlFor="mirror-target-platform">
              目标平台
              <Select
                items={platformItems}
                value={targetPlatform}
                onValueChange={(value) => setTargetPlatform(value ?? "github")}
              >
                <SelectTrigger
                  id="mirror-target-platform"
                  className="mt-2 w-full"
                >
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {PLATFORMS.map((platform) => (
                    <SelectItem key={platform.id} value={platform.id}>
                      {platform.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </label>
            <label className="grm-label" htmlFor="mirror-mode">
              备份模式
              <Select
                items={MODE_ITEMS}
                value={mode}
                onValueChange={(value) => setMode(value ?? "mirror")}
              >
                <SelectTrigger id="mirror-mode" className="mt-2 w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {MODE_ITEMS.map((item) => (
                    <SelectItem key={item.value} value={item.value}>
                      {item.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </label>
          </div>

          <div id="targetFields" className="mt-4">
            {target.managed ? (
              <>
                <div className="grm-two-col">
                  <label className="grm-label">
                    目标命名空间 <span>{target.namespaceHint}</span>
                    <Input
                      name="targetNamespace"
                      required
                      className="mt-2"
                      value={effectiveNamespace}
                      onChange={(event) =>
                        setTargetNamespace(event.target.value)
                      }
                      placeholder="team/platform"
                    />
                  </label>
                  <label className="grm-label">
                    自定义目标名称 <span>留空则使用源仓库名</span>
                    <Input
                      name="targetName"
                      className="mt-2"
                      value={targetName}
                      onChange={(event) => setTargetName(event.target.value)}
                      placeholder="my-repository-mirror"
                    />
                  </label>
                </div>
                <div className="grm-two-col mt-4">
                  <label className="grm-label" htmlFor="mirror-visibility">
                    目标可见性
                    {target.fixedVisibility ? (
                      <Input value="跟随项目" disabled className="mt-2" />
                    ) : (
                      <Select
                        items={[
                          { label: "Private", value: "private" },
                          { label: "Internal", value: "internal" },
                          { label: "Public", value: "public" },
                        ]}
                        value={visibility}
                        onValueChange={(value) =>
                          setVisibility(value ?? "private")
                        }
                      >
                        <SelectTrigger
                          id="mirror-visibility"
                          className="mt-2 w-full"
                        >
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="private">Private</SelectItem>
                          <SelectItem value="internal">Internal</SelectItem>
                          <SelectItem value="public">Public</SelectItem>
                        </SelectContent>
                      </Select>
                    )}
                  </label>
                  <label className="grm-label" htmlFor="mirror-conflict">
                    同名仓库处理
                    <Select
                      items={[
                        { label: "询问并停止", value: "error" },
                        { label: "关联已有仓库", value: "associate" },
                        { label: "自动添加数字后缀", value: "rename" },
                      ]}
                      value={conflictPolicy}
                      onValueChange={(value) =>
                        setConflictPolicy(value ?? "error")
                      }
                    >
                      <SelectTrigger
                        id="mirror-conflict"
                        className="mt-2 w-full"
                      >
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="error">询问并停止</SelectItem>
                        <SelectItem value="associate">关联已有仓库</SelectItem>
                        <SelectItem value="rename">自动添加数字后缀</SelectItem>
                      </SelectContent>
                    </Select>
                  </label>
                </div>
              </>
            ) : (
              <>
                <div className="grm-target-hint warning">
                  <span>{target.message}</span>
                  {target.canLoginToTarget && (
                    <Button
                      type="button"
                      size="xs"
                      className="mt-2"
                      onClick={() => onLoginPlatform?.(targetPlatform)}
                    >
                      重新登录
                    </Button>
                  )}
                </div>
                <label className="grm-label mt-4">
                  目标仓库 URL <span>支持 HTTPS、SSH 或 SCP 地址</span>
                  <Input
                    name="targetUrl"
                    required
                    className="mt-2"
                    value={targetUrl}
                    onChange={(event) => setTargetUrl(event.target.value)}
                    placeholder="git@code.example:team/project.git"
                    autoComplete="off"
                  />
                </label>
              </>
            )}
          </div>

          <div className="grm-two-col mt-4">
            <label className="grm-label">
              名称前缀
              <Input
                name="prefix"
                className="mt-2"
                value={prefix}
                onChange={(event) => setPrefix(event.target.value)}
                placeholder="mirror-"
              />
            </label>
            <label className="grm-label">
              名称后缀
              <Input
                name="suffix"
                className="mt-2"
                value={suffix}
                onChange={(event) => setSuffix(event.target.value)}
                placeholder="-backup"
              />
            </label>
          </div>
          <label className="grm-check-row mt-4">
            <Switch
              checked={syncLfs}
              onCheckedChange={setSyncLfs}
              aria-label="同步 Git LFS 对象"
            />
            <span>
              <strong>同步 Git LFS 对象</strong>
              <small>需要本机已安装 Git LFS，可能显著增加耗时和流量</small>
            </span>
          </label>

          <div className="grm-steps" id="modalSteps">
            {["1 准备目标库", "2 拉取源代码", "3 推送镜像", "4 清理完成"].map(
              (label, index) => (
                <span key={label} className={cn(stepState(index))}>
                  {label}
                </span>
              ),
            )}
          </div>
          <div className="grm-dialog-footer">
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
              disabled={submitting}
            >
              取消
            </Button>
            <Button type="submit" disabled={submitting}>
              {submitting ? "正在备份…" : "开始备份"}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  );
}
