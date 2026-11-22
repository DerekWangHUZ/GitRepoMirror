// 来源：macUI-SolidGlass 模板本机副本（web/src/macos/scrollbar.tsx），复制日期 2026-11-22。
// token 禁止手工微调 —— 一切取值改动必须回到模板仓库进行。
// （去 TS 化：类型标注移除，逻辑逐行保留。）
import { useEffect, useRef, useState } from "react";

import { cn } from "@/lib/utils";

/**
 * 全局自绘 macOS 悬浮滚动条。
 *
 * 原生 WebView2 滚动条会在窗口最右缘切出一道灰白矩形轨道，
 * 并在右上 / 右下 32px 圆角处被裁切。这里改用透明轨道 + 胶囊滑块，
 * 上下各避让 44px 圆角区域，同时支持 1:1 拖拽与点击轨道跳转。
 */
export function MacScrollbar({ targetRef, route }) {
  const trackRef = useRef(null);
  const thumbRef = useRef(null);
  const dragRef = useRef(null);
  const [visible, setVisible] = useState(false);
  const [dragging, setDragging] = useState(false);

  useEffect(() => {
    const scroller = targetRef.current;
    if (!scroller) return;

    let raf = 0;
    const sync = () => {
      const track = trackRef.current;
      const thumb = thumbRef.current;
      if (!track || !thumb) return;

      const view = scroller.clientHeight;
      const total = scroller.scrollHeight;
      if (total <= view + 1) {
        setVisible(false);
        return;
      }

      setVisible(true);
      const trackH = track.clientHeight;
      if (trackH <= 0) return;

      const thumbH = Math.max(32, Math.round((trackH * view) / total));
      const maxScroll = Math.max(1, total - view);
      const offset = (scroller.scrollTop / maxScroll) * Math.max(0, trackH - thumbH);

      thumb.style.height = `${thumbH}px`;
      thumb.style.transform = `translateY(${offset}px)`;
    };

    const scheduleSync = () => {
      window.cancelAnimationFrame(raf);
      raf = window.requestAnimationFrame(sync);
    };

    scroller.addEventListener("scroll", sync, { passive: true });
    window.addEventListener("resize", scheduleSync);

    const ro = new ResizeObserver(scheduleSync);
    ro.observe(scroller);
    const inner = scroller.firstElementChild;
    if (inner) ro.observe(inner);

    const mo = new MutationObserver(scheduleSync);
    if (inner) {
      mo.observe(inner, { childList: true, subtree: true });
    }

    scheduleSync();
    const t = window.setTimeout(sync, 250);

    return () => {
      window.cancelAnimationFrame(raf);
      window.clearTimeout(t);
      scroller.removeEventListener("scroll", sync);
      window.removeEventListener("resize", scheduleSync);
      ro.disconnect();
      mo.disconnect();
    };
  }, [targetRef, route]);

  const onThumbPointerDown = (event) => {
    const scroller = targetRef.current;
    if (!scroller) return;
    event.preventDefault();
    event.stopPropagation();
    event.currentTarget.setPointerCapture(event.pointerId);
    dragRef.current = { startY: event.clientY, startScroll: scroller.scrollTop };
    setDragging(true);
  };

  const onThumbPointerMove = (event) => {
    const drag = dragRef.current;
    const scroller = targetRef.current;
    const track = trackRef.current;
    const thumb = thumbRef.current;
    if (!drag || !scroller || !track || !thumb) return;

    const span = Math.max(1, track.clientHeight - thumb.offsetHeight);
    const maxScroll = Math.max(0, scroller.scrollHeight - scroller.clientHeight);
    scroller.scrollTop = drag.startScroll + (event.clientY - drag.startY) * (maxScroll / span);
  };

  const onThumbPointerUp = (event) => {
    if (!dragRef.current) return;
    dragRef.current = null;
    setDragging(false);
    if (event.currentTarget.hasPointerCapture(event.pointerId)) {
      event.currentTarget.releasePointerCapture(event.pointerId);
    }
  };

  const onTrackClick = (event) => {
    const scroller = targetRef.current;
    const track = trackRef.current;
    const thumb = thumbRef.current;
    if (!scroller || !track || !thumb || event.target === thumb) return;

    const rect = track.getBoundingClientRect();
    const clickY = event.clientY - rect.top;
    const thumbH = thumb.offsetHeight;
    const span = Math.max(1, track.clientHeight - thumbH);
    const maxScroll = Math.max(0, scroller.scrollHeight - scroller.clientHeight);
    const ratio = Math.max(0, Math.min(1, (clickY - thumbH / 2) / span));
    scroller.scrollTo({ top: ratio * maxScroll, behavior: "smooth" });
  };

  return (
    <div
      ref={trackRef}
      aria-hidden="true"
      onClick={onTrackClick}
      className={cn("mac-scrollbar", visible && "is-on", dragging && "is-dragging")}
    >
      <div
        ref={thumbRef}
        className="mac-scrollbar__thumb"
        onPointerDown={onThumbPointerDown}
        onPointerMove={onThumbPointerMove}
        onPointerUp={onThumbPointerUp}
        onPointerCancel={onThumbPointerUp}
      />
    </div>
  );
}
