// 来源：macUI-SolidGlass 模板本机副本（desktop/src/skin/desktop.js），复制日期 2026-11-22。
// token 禁止手工微调 —— 一切取值改动必须回到模板仓库进行。
//
// 桌面版窗口行为：红绿灯事件委托、双击最大化、外链交给系统默认浏览器。
//
// 网页版的红绿灯只是演示；桌面版里通过 document 级事件委托接管，
// 无论 React 如何重新渲染或切换窄/宽窗布局都不会丢失绑定。
;(function () {
  "use strict"

  var html = document.documentElement

  function runtime() {
    // Wails 运行时是注入的全局对象
    return window.runtime
  }

  /** 标记这是桌面版：CSS 据此不再裁窗口圆角（由系统裁）。 */
  html.classList.add("is-desktop")

  function isElement(node) {
    return !!node && node.nodeType === 1
  }

  function call(rt, name, arg) {
    try {
      if (typeof rt[name] !== "function") return false
      if (typeof arg === "undefined") rt[name]()
      else rt[name](arg)
      return true
    } catch (e) {
      console.error("调用桌面窗口运行时失败:", name, e)
      return false
    }
  }

  function toggleMaximise(rt) {
    if (!call(rt, "WindowToggleMaximise")) {
      call(rt, "WindowMaximise")
    }
  }

  // 1. 红绿灯点击事件委托（capture 模式优先于 React 演示回调）
  document.addEventListener(
    "click",
    function (event) {
      var rt = runtime()
      if (!rt || !isElement(event.target)) return
      var btn = event.target.closest("[data-win], .light--close, .light--min, .light--max")
      if (!btn) return

      event.preventDefault()
      event.stopPropagation()

      var action = btn.getAttribute("data-win")
      if (!action) {
        if (btn.classList.contains("light--close")) action = "close"
        else if (btn.classList.contains("light--min")) action = "min"
        else if (btn.classList.contains("light--max")) action = "max"
      }

      if (action === "close") call(rt, "Quit")
      else if (action === "min") call(rt, "WindowMinimise")
      else if (action === "max") toggleMaximise(rt)
    },
    true
  )

  // 2. 双击标题区 / 拖拽带切换最大化
  document.addEventListener("dblclick", function (event) {
    var rt = runtime()
    if (!rt || !isElement(event.target)) return
    if (event.target.closest("button, a, input, select, textarea, .traffic")) return
    if (!event.target.closest(".sidebar__chrome, .sidebar__header, .drag-strip, .mobile-topbar, .toolbar__heading")) return
    toggleMaximise(rt)
  })

  // 3. 外链交给系统默认浏览器打开，不在应用 WebView 内跳走
  document.addEventListener("click", function (event) {
    var rt = runtime()
    if (!rt || !isElement(event.target)) return
    var link = event.target.closest("a[href]")
    if (!link) return
    var href = link.getAttribute("href") || ""
    if (/^https?:\/\//i.test(href)) {
      event.preventDefault()
      call(rt, "BrowserOpenURL", href)
    }
  })

  // 4. 挂载顶部 22px 拖拽带（确保始终存在）
  function ensureDragStrip() {
    if (!document.body || document.querySelector(".drag-strip")) return
    var strip = document.createElement("div")
    strip.className = "drag-strip"
    strip.setAttribute("aria-hidden", "true")
    document.body.appendChild(strip)
  }

  // 5. is-maximized 同步：resize 防抖查询 + 事件监听
  //    桌面版走 Wails 绑定 WindowIsMaximised（app.go 提供）；
  //    window.runtime 桥不可用时退化到绑定层（window.go）。
  //    每次状态落定都广播 grm:window-maximized，让 React 的红绿灯
  //    maximized 态跟随真实窗口（事件驱动，零轮询）。
  function applyMaximized(value) {
    html.classList.toggle("is-maximized", !!value)
    window.dispatchEvent(new CustomEvent("grm:window-maximized", { detail: !!value }))
  }

  function syncMaximized(rt) {
    var query = null
    if (rt && typeof rt.WindowIsMaximised === "function") {
      query = function () {
        return rt.WindowIsMaximised()
      }
    } else if (window.go && window.go.main && window.go.main.App &&
               typeof window.go.main.App.WindowIsMaximised === "function") {
      query = function () {
        return window.go.main.App.WindowIsMaximised()
      }
    }
    if (!query) return
    try {
      var res = query()
      if (res && typeof res.then === "function") {
        res.then(function (v) {
          applyMaximized(v)
        })
      } else {
        applyMaximized(res)
      }
    } catch {
      /* 查询失败不阻断后续 resize 防抖重试 */
    }
  }

  function bindWindowState() {
    ensureDragStrip()
    var rt = runtime()

    var maxTimer = null
    window.addEventListener("resize", function () {
      if (maxTimer) clearTimeout(maxTimer)
      maxTimer = setTimeout(function () {
        syncMaximized(rt)
      }, 140)
    })
    syncMaximized(rt)

    try {
      rt &&
        rt.EventsOn &&
        rt.EventsOn("window:state-changed", function (state) {
          // state: 1 正常 2 最小化 3 最大化
          applyMaximized(state === 3)
        })
    } catch {
      // EventsOn 不可用时不影响 resize 防抖查询兜底
      console.error("监听窗口状态失败")
    }
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", bindWindowState)
  } else {
    bindWindowState()
  }
})()
