package tinypod

import (
	"image/color"
	"sync/atomic"

	"github.com/hybridgroup/tinyplayer"
)

// WS2812 is a strip of one or more WS2812 LEDs, such as ws2812.Device.
type WS2812 interface {
	WriteColors(buf []color.RGBA) error
}

// meterStride checks every 8th frame, so the meter adds little time to the
// audio path. That is enough to follow the level for the LEDs.
const meterStride = 8

// meter passes audio to the output and keeps the loudest sample since the
// last call to take.
type meter struct {
	tinyplayer.Output
	peak atomic.Uint32
}

func (m *meter) WriteStereo(b []uint32) (int, error) {
	var p uint32
	for i := 0; i < len(b); i += meterStride {
		f := b[i]
		p = max(p, abs16(int16(f)), abs16(int16(f>>16)))
	}
	for {
		old := m.peak.Load()
		if p <= old || m.peak.CompareAndSwap(old, p) {
			break
		}
	}
	return m.Output.WriteStereo(b)
}

// take returns the peak from 0 to 32768 and resets it.
func (m *meter) take() uint32 {
	return m.peak.Swap(0)
}

func abs16(v int16) uint32 {
	if v < 0 {
		return uint32(-int32(v))
	}
	return uint32(v)
}

const (
	lightMax   = 128
	lightDecay = 7
)

// light shows the music level on the LEDs. The LEDs get brighter with the
// level, or fill up like a level meter. The color slowly changes.
type light struct {
	led   WS2812
	bar   bool
	m     *meter
	buf   []color.RGBA
	level int
	hue   uint8
	off   bool
}

func newLight(led WS2812, n int, bar bool, m *meter) *light {
	return &light{led: led, bar: bar, m: m, buf: make([]color.RGBA, max(n, 1))}
}

// update follows the level when on is true, and fades out when it is false.
func (l *light) update(on bool) {
	peak := l.m.take()
	if !on {
		peak = 0
	}
	if v := int(peak >> 7); v > l.level {
		l.level = min(v, 255)
	} else {
		l.level = l.level * lightDecay / 8
	}
	if l.level == 0 {
		if !l.off {
			clear(l.buf)
			l.led.WriteColors(l.buf)
			l.off = true
		}
		return
	}
	l.off = false
	l.hue++
	c := wheel(l.hue)
	n := len(l.buf)
	for i := range l.buf {
		b := l.level * lightMax / 255
		if l.bar {
			b = min(max(l.level*n-i*255, 0), 255) * lightMax / 255
		}
		l.buf[i] = color.RGBA{uint8(int(c.R) * b / 255), uint8(int(c.G) * b / 255), uint8(int(c.B) * b / 255), 255}
	}
	l.led.WriteColors(l.buf)
}

// wheel goes from red to green to blue and back to red.
func wheel(h uint8) color.RGBA {
	switch {
	case h < 85:
		return color.RGBA{255 - h*3, h * 3, 0, 255}
	case h < 170:
		h -= 85
		return color.RGBA{0, 255 - h*3, h * 3, 255}
	}
	h -= 170
	return color.RGBA{h * 3, 0, 255 - h*3, 255}
}
