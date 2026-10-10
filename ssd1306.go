package tinypod

import (
	"image/color"
	"runtime"

	"tinygo.org/x/drivers"
	"tinygo.org/x/drivers/ssd1306"
)

const chunkSize = 32

// SSD1306 is an I2C SSD1306 that sends the frame in small pieces and yields
// in between, so audio keeps flowing on boards where I2C blocks the CPU.
type SSD1306 struct {
	*ssd1306.Device
	bus   drivers.I2C
	addr  uint16
	chunk [chunkSize + 1]byte
}

// NewSSD1306 configures an SSD1306 at addr on bus, which must already be
// configured.
func NewSSD1306(bus drivers.I2C, addr uint16, width, height int16) *SSD1306 {
	dev := ssd1306.NewI2C(bus)
	dev.Configure(ssd1306.Config{Width: width, Height: height, Address: addr})
	return &SSD1306{Device: dev, bus: bus, addr: addr}
}

// FillRectangle sets the pixels in the frame buffer. ssd1306.Device clears the
// whole screen for a black fill that reaches the bottom right corner.
func (s *SSD1306) FillRectangle(x, y, w, h int16, c color.RGBA) error {
	for j := y; j < y+h; j++ {
		for i := x; i < x+w; i++ {
			s.SetPixel(i, j, c)
		}
	}
	return nil
}

func (s *SSD1306) Display() error {
	w, h := s.Size()
	for _, c := range [...]byte{ssd1306.COLUMNADDR, 0, byte(w - 1), ssd1306.PAGEADDR, 0, byte(h/8 - 1)} {
		s.Command(c)
	}
	buf := s.GetBuffer()
	s.chunk[0] = 0x40
	for i := 0; i < len(buf); i += chunkSize {
		copy(s.chunk[1:], buf[i:i+chunkSize])
		if err := s.bus.Tx(s.addr, s.chunk[:], nil); err != nil {
			return err
		}
		runtime.Gosched()
	}
	return nil
}
