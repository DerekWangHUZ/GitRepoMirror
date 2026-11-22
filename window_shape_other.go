//go:build !windows

package main

// 非 Windows 平台占位：窗口圆角裁剪只在 Windows 上生效
// （模板 desktop/window_shape.go 的交叉编译回退）。

func installWindowShapeWhenReady() {}

func shapeReapply() {}
