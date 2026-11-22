// 来源：macUI-SolidGlass 模板本机副本（desktop/window_shape.go），复制日期 2026-11-22。
// token 禁止手工微调 —— 一切取值改动必须回到模板仓库进行。
// （本仓适配：日志输出改用标准 log，模板的 logLine 由宿主自行接线。）

//go:build windows

package main

import (
	"log"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// 窗口外形：让窗口圆角与侧栏、顶栏的圆角同心。
//
// 侧栏面板：左边距/上边距 12px、圆角 20px → 圆心在 (32,32)
// 顶栏（窄窗）：外边距 8px、圆角 24px      → 圆心在 (32,32)
// 因此窗口自身也要 32px 圆角，三段圆弧才共用同一个圆心。
//
// DWM 只提供 8px / 4px 两档系统圆角（DWMWA_WINDOW_CORNER_PREFERENCE），
// 所以这里用窗口区域（SetWindowRgn）自己裁一个 32px 圆角矩形：
// 系统投影与 1px 边框都还在，代价是区域是 1 位掩码、边缘没有抗锯齿。
// 最大化时清掉区域（系统也不会给最大化窗口圆角）。

const (
	cornerRadiusDIP = 32.0
	wmSize          = 0x0005
	wmDpiChanged    = 0x02E0
	gwlWndProc      = ^uintptr(3) // GWLP_WNDPROC = -4
)

var (
	gdi32  = windows.NewLazySystemDLL("gdi32.dll")
	user32 = windows.NewLazySystemDLL("user32.dll")

	procCreateRoundRectRgn = gdi32.NewProc("CreateRoundRectRgn")
	procDeleteObject       = gdi32.NewProc("DeleteObject")

	procGetWindowRect     = user32.NewProc("GetWindowRect")
	procSetWindowRgn      = user32.NewProc("SetWindowRgn")
	procSetWindowLongPtrW = user32.NewProc("SetWindowLongPtrW")
	procCallWindowProcW   = user32.NewProc("CallWindowProcW")
	procGetDpiForWindow   = user32.NewProc("GetDpiForWindow")
	procIsZoomed          = user32.NewProc("IsZoomed")

	procEnumWindows              = user32.NewProc("EnumWindows")
	procGetWindowThreadProcessID = user32.NewProc("GetWindowThreadProcessId")
	procIsWindowVisible          = user32.NewProc("IsWindowVisible")
	procGetWindowTextW           = user32.NewProc("GetWindowTextW")

	shapeHWND   uintptr
	prevWndProc uintptr
)

type winRect struct {
	Left, Top, Right, Bottom int32
}

// mainWindowHandle 找出本进程那个可见、有标题的顶层窗口。
func mainWindowHandle() uintptr {
	pid := windows.GetCurrentProcessId()
	var found uintptr
	cb := syscall.NewCallback(func(hwnd uintptr, _ uintptr) uintptr {
		var wpid uint32
		procGetWindowThreadProcessID.Call(hwnd, uintptr(unsafe.Pointer(&wpid)))
		if wpid != pid {
			return 1
		}
		if vis, _, _ := procIsWindowVisible.Call(hwnd); vis == 0 {
			return 1
		}
		var buf [64]uint16
		n, _, _ := procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
		if n == 0 {
			return 1
		}
		found = hwnd
		return 0 // 找到即停
	})
	procEnumWindows.Call(cb, 0)
	return found
}

// applyWindowShape 按当前尺寸与 DPI 重新设置窗口圆角区域。
func applyWindowShape(hwnd uintptr) {
	if hwnd == 0 {
		return
	}

	// 最大化时系统不给圆角，这里同样清掉区域
	if zoomed, _, _ := procIsZoomed.Call(hwnd); zoomed != 0 {
		procSetWindowRgn.Call(hwnd, 0, 1)
		return
	}

	var rect winRect
	if ok, _, _ := procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&rect))); ok == 0 {
		return
	}
	w := int(rect.Right - rect.Left)
	h := int(rect.Bottom - rect.Top)
	if w <= 0 || h <= 0 {
		return
	}

	dpi, _, _ := procGetDpiForWindow.Call(hwnd)
	if dpi == 0 {
		dpi = 96
	}
	diameter := int(cornerRadiusDIP*float64(dpi)/96.0+0.5) * 2

	// +1：区域右/下边界是开区间，尺寸要多给 1px 才不会切掉一列像素
	rgn, _, _ := procCreateRoundRectRgn.Call(0, 0, uintptr(w+1), uintptr(h+1),
		uintptr(diameter), uintptr(diameter))
	if rgn == 0 {
		return
	}
	// 成功后区域归系统所有，不能再 DeleteObject
	if ok, _, _ := procSetWindowRgn.Call(hwnd, rgn, 1); ok == 0 {
		procDeleteObject.Call(rgn)
	}
}

// installWindowShape 接管窗口过程，在尺寸变化时重新裁剪；只安装一次。
func installWindowShape(hwnd uintptr) {
	shapeHWND = hwnd
	if prevWndProc == 0 {
		cb := syscall.NewCallback(func(h, msg, wparam, lparam uintptr) uintptr {
			ret, _, _ := procCallWindowProcW.Call(prevWndProc, h, msg, wparam, lparam)
			if msg == wmSize || msg == wmDpiChanged {
				applyWindowShape(h)
			}
			return ret
		})
		prev, _, err := procSetWindowLongPtrW.Call(hwnd, gwlWndProc, cb)
		if prev == 0 || (err != nil && err != syscall.Errno(0)) {
			log.Println("窗口外形：接管窗口过程失败，缩放时圆角可能不同步")
		}
		prevWndProc = prev
	}
	applyWindowShape(hwnd)
}

// installWindowShapeWhenReady 在窗口出现后立刻裁剪，避免先看到系统方角。
// 只轮询很短一段时间（窗口通常几十毫秒内就出现），跑一次即结束。
func installWindowShapeWhenReady() {
	go func() {
		for i := 0; i < 200; i++ {
			if hwnd := mainWindowHandle(); hwnd != 0 {
				installWindowShape(hwnd)
				return
			}
			time.Sleep(20 * time.Millisecond)
		}
		log.Println("窗口外形：等待主窗口超时，未应用圆角裁剪")
	}()
}

// shapeReapply 供非 Windows 空实现占位（本仓只在 Windows 构建里生效）。
func shapeReapply() {
	if shapeHWND != 0 {
		applyWindowShape(shapeHWND)
	}
}
