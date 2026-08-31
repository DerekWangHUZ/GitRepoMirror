package main

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"os"
)

const size = 512

var gitOrange = color.RGBA{R: 240, G: 80, B: 51, A: 255}

func main() {
	if err := writePNG("build/appicon.png", renderIcon(size)); err != nil {
		panic(err)
	}
	if err := writeICO("build/windows/icon.ico", []int{16, 20, 24, 32, 40, 48, 64, 128, 256}); err != nil {
		panic(err)
	}
}

func renderIcon(iconSize int) *image.RGBA {
	canvas := image.NewRGBA(image.Rect(0, 0, iconSize, iconSize))
	stroke := iconSize * 18 / 240
	if stroke < 2 {
		stroke = 2
	}
	point := func(value10 int) int { return value10 * iconSize / 240 }

	diamond := [][2]int{{120, 20}, {220, 120}, {120, 220}, {20, 120}, {120, 20}}
	for index := 1; index < len(diamond); index++ {
		drawLine(canvas, point(diamond[index-1][0]), point(diamond[index-1][1]), point(diamond[index][0]), point(diamond[index][1]), stroke, gitOrange)
	}
	drawLine(canvas, point(96), point(96), point(144), point(144), stroke, gitOrange)
	drawLine(canvas, point(85), point(100), point(85), point(150), stroke, gitOrange)
	drawCircle(canvas, point(85), point(85), point(15), stroke, gitOrange)
	drawCircle(canvas, point(155), point(155), point(15), stroke, gitOrange)
	return canvas
}

func writePNG(path string, icon image.Image) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return png.Encode(file, icon)
}

func writeICO(path string, sizes []int) error {
	images := make([][]byte, len(sizes))
	for index, iconSize := range sizes {
		var data bytes.Buffer
		if err := png.Encode(&data, renderIcon(iconSize)); err != nil {
			return err
		}
		images[index] = data.Bytes()
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	if err := binary.Write(file, binary.LittleEndian, uint16(0)); err != nil {
		return err
	}
	if err := binary.Write(file, binary.LittleEndian, uint16(1)); err != nil {
		return err
	}
	if err := binary.Write(file, binary.LittleEndian, uint16(len(images))); err != nil {
		return err
	}

	offset := uint32(6 + 16*len(images))
	for index, data := range images {
		iconSize := sizes[index]
		width, height := byte(iconSize), byte(iconSize)
		if iconSize == 256 {
			width, height = 0, 0
		}
		if _, err := file.Write([]byte{width, height, 0, 0}); err != nil {
			return err
		}
		if err := binary.Write(file, binary.LittleEndian, uint16(1)); err != nil {
			return err
		}
		if err := binary.Write(file, binary.LittleEndian, uint16(32)); err != nil {
			return err
		}
		if err := binary.Write(file, binary.LittleEndian, uint32(len(data))); err != nil {
			return err
		}
		if err := binary.Write(file, binary.LittleEndian, offset); err != nil {
			return err
		}
		offset += uint32(len(data))
	}
	for _, data := range images {
		if _, err := file.Write(data); err != nil {
			return err
		}
	}
	return nil
}

func drawCircle(img *image.RGBA, cx, cy, radius, width int, fill color.RGBA) {
	outer := radius + width/2
	inner := radius - width/2
	if inner < 0 {
		inner = 0
	}
	for y := cy - outer; y <= cy+outer; y++ {
		for x := cx - outer; x <= cx+outer; x++ {
			dx, dy := x-cx, y-cy
			distance := dx*dx + dy*dy
			if distance <= outer*outer && distance >= inner*inner && image.Pt(x, y).In(img.Bounds()) {
				img.SetRGBA(x, y, fill)
			}
		}
	}
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
