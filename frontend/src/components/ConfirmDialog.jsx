// 轻量确认框：迁移自 actions.js 的 confirmDialog（改用 Solid Glass Dialog）。
// 应用内自绘，绝不触发 WebView2 原生 confirm（标题会是 "wails.localhost"）。
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

export function ConfirmDialog({ request, onResolve }) {
  // request: { title, message, confirmText, cancelText, danger } | null
  const open = Boolean(request);
  const done = (value) => {
    request?.resolve?.(value);
    onResolve?.();
  };
  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!next) done(false);
      }}
    >
      <DialogContent className="grm-confirm-content" showCloseButton={false}>
        <DialogHeader>
          <DialogTitle>{request?.title}</DialogTitle>
          <DialogDescription className="whitespace-pre-line">
            {request?.message}
          </DialogDescription>
        </DialogHeader>
        <div className="grm-dialog-footer">
          <Button variant="outline" onClick={() => done(false)}>
            {request?.cancelText || "取消"}
          </Button>
          <Button
            autoFocus
            variant={request?.danger ? "destructive" : "default"}
            onClick={() => done(true)}
          >
            {request?.confirmText || "确认"}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}
