// Package tinypod is an iPod classic style music player. It works with any
// display, an SD card or flash for the songs, and any set of buttons. A small
// main for each device sets up the hardware and calls Run.
package tinypod

import (
	"time"

	"github.com/hybridgroup/tinyplayer"
)

// Config is the hardware that tinypod uses.
type Config struct {
	Display Display
	Theme   Theme
	Output  tinyplayer.Output
	Storage Storage
	Input   Input

	// LED is optional. The LEDs light up in time with the music.
	LED      WS2812
	LEDCount int

	// ReadAhead reads songs into a 32 KB buffer in a separate goroutine.
	// It helps when the display or the card blocks the CPU, as on the
	// ESP32-C3. Leave it off with piolib I2S, which queues one DMA buffer.
	ReadAhead bool
}

// Run scans the songs and runs the player. It shows the splash screen while it
// scans, unless the device main called Splash already. It returns only if the
// scan fails.
func Run(c Config) error {
	if splashAt.IsZero() {
		Splash(c.Display, c.Theme)
	}
	lib, err := newLibrary(c.Storage)
	if err != nil {
		return err
	}
	waitSplash()
	println("songs:", len(lib.songs), "albums:", len(lib.albums))
	out := c.Output
	var l *light
	if c.LED != nil {
		m := &meter{Output: out}
		out = m
		l = newLight(c.LED, c.LEDCount, m)
	}
	d := newDeck(out, lib, c.ReadAhead)
	a := newApp(newScreen(c.Display, c.Theme), c.Input, lib, d)
	a.light = l
	a.run()
	return nil
}

// Fail shows the error and never returns.
func Fail(d Display, t Theme, msg string, err error) {
	Message(d, t, "tinypod", msg, err.Error())
	for {
		println(msg+":", err.Error())
		time.Sleep(time.Second)
	}
}
