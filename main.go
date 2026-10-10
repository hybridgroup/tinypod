//go:build esp32c3 || rp2350

// tinypod is an iPod classic style music player for a XIAO board on the Seeed
// XIAO expansion board. It plays WAV files from the microSD card.
package main

import (
	"machine"
	"time"

	"github.com/hybridgroup/tinyplayer"
	"github.com/soypat/fat"
)

func main() {
	time.Sleep(time.Second)

	// The expansion board buzzer is on D3, keep it quiet.
	buzzer := machine.D3
	buzzer.Configure(machine.PinConfig{Mode: machine.PinOutput})
	buzzer.Low()

	i2c.Configure(machine.I2CConfig{SDA: machine.D4, SCL: machine.D5, Frequency: 400 * machine.KHz})
	scr := newScreen()
	scr.message("tinypod", "starting")

	out, err := audioOutput()
	if err != nil {
		failed(scr, "no audio", err)
	}
	player := tinyplayer.New(out)

	var fs fat.FS
	if err := mountCard(&fs); err != nil {
		failed(scr, "no SD card", err)
	}
	scr.message("tinypod", "reading songs")
	lib, err := scan(&fs)
	if err != nil {
		failed(scr, "scan failed", err)
	}
	println("songs:", len(lib.songs), "albums:", len(lib.albums))

	btn := newButton()
	var in input = btn
	if g := newGamepad(btn); g != nil {
		println("gamepad found")
		in = g
	}

	newApp(scr, lib, newDeck(player, &fs), player, in).run()
}

func failed(scr *screen, msg string, err error) {
	scr.message("tinypod", msg, err.Error())
	for {
		println(msg+":", err.Error())
		time.Sleep(time.Second)
	}
}
