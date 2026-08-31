package main

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
)

const size = 512

func main() {
	canvas := image.NewRGBA(image.Rect(0, 0, size, size))
	draw.Draw(canvas, canvas.Bounds(), image.NewUniform(color.RGBA{R: 244, G: 248, B: 252, A: 255}), image.Point{}, draw.Src)
	for y := 32; y < 480; y++ {
		for x := 32; x < 480; x++ {
			if insideRoundedSquare(x, y, 32, 480, 116) {
				t := float64(x+y-64) / 896
				canvas.SetRGBA(x, y, color.RGBA{R: uint8(40 - 35*t), G: uint8(89 + 76*t), B: uint8(199 - 16*t), A: 255})
			}
		}
	}
	white := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	drawLine(canvas, 142, 146, 256, 90, 24, white)
	drawLine(canvas, 256, 90, 370, 146, 24, white)
	drawLine(canvas, 370, 146, 256, 203, 24, white)
	drawLine(canvas, 256, 203, 142, 146, 24, white)
	drawLine(canvas, 142, 203, 256, 259, 24, white)
	drawLine(canvas, 256, 259, 370, 203, 24, white)
	drawLine(canvas, 142, 203, 142, 296, 24, white)
	drawLine(canvas, 142, 296, 256, 353, 24, white)
	drawLine(canvas, 256, 353, 370, 296, 24, white)
	drawLine(canvas, 370, 296, 370, 203, 24, white)
	drawLine(canvas, 142, 296, 142, 366, 24, white)
	drawLine(canvas, 142, 366, 256, 422, 24, white)
	drawLine(canvas, 256, 422, 370, 366, 24, white)
	drawLine(canvas, 370, 366, 370, 296, 24, white)
	file, err := os.Create("build/appicon.png")
	if err != nil {
		panic(err)
	}
	defer file.Close()
	if err := png.Encode(file, canvas); err != nil {
		panic(err)
	}
}

func insideRoundedSquare(x, y, min, max, radius int) bool {
	if x >= min+radius && x < max-radius || y >= min+radius && y < max-radius {
		return true
	}
	cx := min + radius
	if x >= max-radius {
		cx = max - radius - 1
	}
	cy := min + radius
	if y >= max-radius {
		cy = max - radius - 1
	}
	dx, dy := x-cx, y-cy
	return dx*dx+dy*dy <= radius*radius
}

func drawLine(img *image.RGBA, x0, y0, x1, y1, width int, fill color.RGBA) {
	dx, dy := x1-x0, y1-y0
	steps := abs(dx)
	if abs(dy) > steps {
		steps = abs(dy)
	}
	for step := 0; step <= steps; step++ {
		x := x0 + dx*step/steps
		y := y0 + dy*step/steps
		for py := y - width/2; py <= y+width/2; py++ {
			for px := x - width/2; px <= x+width/2; px++ {
				ox, oy := px-x, py-y
				if ox*ox+oy*oy <= width*width/4 && image.Pt(px, py).In(img.Bounds()) {
					img.SetRGBA(px, py, fill)
				}
			}
		}
	}
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
