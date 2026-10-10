//go:build esp32c3 || rp2350

package main

import (
	"image/color"
	"runtime"

	"tinygo.org/x/drivers/ssd1306"
	"tinygo.org/x/tinyfont"
	"tinygo.org/x/tinyfont/proggy"
)

const (
	oledAddr  = 0x3C
	width     = 128
	charWidth = 6
)

var (
	white = color.RGBA{255, 255, 255, 255}
	black = color.RGBA{0, 0, 0, 255}
	font  = &proggy.TinySZ8pt7b
)

type screen struct {
	dev   *ssd1306.Device
	chunk [width + 1]byte
}

func newScreen() *screen {
	dev := ssd1306.NewI2C(i2c)
	dev.Configure(ssd1306.Config{Width: width, Height: 64, Address: oledAddr})
	return &screen{dev: dev}
}

func (s *screen) clear() {
	s.dev.ClearBuffer()
}

// text draws str with its baseline at y, cut to fit before maxX.
func (s *screen) text(x, y int16, str string, c color.RGBA, maxX int16) {
	if n := int(maxX-x) / charWidth; len(str) > n {
		str = str[:max(n, 0)]
	}
	tinyfont.WriteLine(s.dev, font, x, y, str, c)
}

func (s *screen) rect(x, y, w, h int16, c color.RGBA) {
	s.dev.FillRectangle(x, y, w, h, c)
}

// show sends the frame one 128 byte page at a time and yields in between,
// so audio keeps flowing on boards where I2C blocks the CPU.
func (s *screen) show() {
	for _, c := range [...]byte{ssd1306.COLUMNADDR, 0, width - 1, ssd1306.PAGEADDR, 0, 7} {
		s.dev.Command(c)
	}
	buf := s.dev.GetBuffer()
	s.chunk[0] = 0x40
	for page := 0; page < 8; page++ {
		copy(s.chunk[1:], buf[page*width:])
		i2c.Tx(oledAddr, s.chunk[:], nil)
		runtime.Gosched()
	}
}

func (s *screen) message(lines ...string) {
	s.clear()
	for i, l := range lines {
		s.text(2, int16(14+i*12), l, white, width)
	}
	s.show()
}
