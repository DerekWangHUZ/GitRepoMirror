// 设置视图：迁移自 templates.js 的 settingsView。
// 受控表单：默认值来自后端，编辑后由 App 持有草稿并统一提交。
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { Button } from "@/components/ui/button";

export function SettingsView({ draft, onDraftChange, onSave }) {
  const patch = (changes) => onDraftChange({ ...draft, ...changes });
  return (
    <>
      <div className="grm-page-head">
        <div>
          <h1>设置</h1>
          <p>配置网络、命名与同步行为</p>
        </div>
      </div>
      <form
        className="grm-settings-form"
        id="settingsForm"
        onSubmit={(event) => {
          event.preventDefault();
          onSave();
        }}
      >
        <section className="grm-setting-section">
          <h3>网络</h3>
          <label className="grm-label">
            独立代理地址
            <span>留空使用系统网络。支持 HTTP、HTTPS 和 SOCKS5。</span>
            <Input
              name="proxy"
              className="mt-2"
              value={draft.proxy ?? ""}
              onChange={(event) => patch({ proxy: event.target.value })}
              placeholder="http://127.0.0.1:7890"
            />
          </label>
        </section>
        <section className="grm-setting-section">
          <h3>默认命名</h3>
          <div className="grm-two-col">
            <label className="grm-label">
              名称前缀
              <Input
                name="prefix"
                className="mt-2"
                value={draft.prefix ?? ""}
                onChange={(event) => patch({ prefix: event.target.value })}
                placeholder="mirror-"
              />
            </label>
            <label className="grm-label">
              名称后缀
              <Input
                name="suffix"
                className="mt-2"
                value={draft.suffix ?? ""}
                onChange={(event) => patch({ suffix: event.target.value })}
                placeholder="-backup"
              />
            </label>
          </div>
        </section>
        <section className="grm-setting-section">
          <h3>同步</h3>
          <label className="grm-label" htmlFor="settings-concurrency">
            批量同步并发数
            <span>建议保持 1–2，避免触发托管平台频率限制。</span>
            <Select
              items={[1, 2, 3, 4].map((n) => ({ label: String(n), value: n }))}
              value={draft.concurrency ?? 2}
              onValueChange={(value) =>
                patch({ concurrency: Number(value ?? 2) })
              }
            >
              <SelectTrigger id="settings-concurrency" className="mt-2 w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {[1, 2, 3, 4].map((number) => (
                  <SelectItem key={number} value={number}>
                    {number}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </label>
          <label className="grm-check-row">
            <Switch
              name="syncLfs"
              checked={Boolean(draft.syncLfs)}
              onCheckedChange={(checked) => patch({ syncLfs: checked })}
              aria-label="默认同步 Git LFS 对象"
            />
            <span>
              <strong>默认同步 Git LFS 对象</strong>
              <small>仅在已安装 Git LFS 时生效</small>
            </span>
          </label>
          <label className="grm-check-row">
            <Switch
              name="pruneTargetRefs"
              checked={Boolean(draft.pruneTargetRefs)}
              onCheckedChange={(checked) => patch({ pruneTargetRefs: checked })}
              aria-label="清理目标端多余分支和标签"
            />
            <span>
              <strong>清理目标端多余分支和标签</strong>
              <small>
                完整镜像时删除目标端存在、但源端没有的分支和标签。关闭时保留这些分支；无论开关，同名分支仍会按源端强制覆盖。
              </small>
            </span>
          </label>
        </section>
        <Button type="submit">保存设置</Button>
      </form>
    </>
  );
}
