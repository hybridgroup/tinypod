//go:build esp32c3 || rp2350

// tinypod on a XIAO board plugged into the Seeed XIAO expansion board. Songs
// are on the microSD card. Input is an Adafruit Mini I2C Gamepad QT, or the
// board button on D1 alone.
package main

import (
	"machine"
	"time"

	"github.com/hybridgroup/tinypod"
)

func main() {
	time.Sleep(time.Second)

	// The expansion board buzzer is on D3, keep it quiet.
	buzzer := machine.D3
	buzzer.Configure(machine.PinConfig{Mode: machine.PinOutput})
	buzzer.Low()

	i2c.Configure(machine.I2CConfig{SDA: machine.D4, SCL: machine.D5, Frequency: 400 * machine.KHz})
	disp := tinypod.NewSSD1306(i2c, 0x3C, 128, 64)
	theme := tinypod.MonoTheme()
	tinypod.Splash(disp, theme)

	out, err := audioOutput()
	if err != nil {
		tinypod.Fail(disp, theme, "no audio", err)
	}

	vol, err := mountCard()
	if err != nil {
		tinypod.Fail(disp, theme, "no SD card", err)
	}

	btn := machine.D1
	btn.Configure(machine.PinConfig{Mode: machine.PinInputPullup})
	pressed := func() bool { return !btn.Get() }
	in := tinypod.HoldButton(pressed)
	if gp := tinypod.Gamepad(i2c, tinypod.DefaultGamepadMap); gp != nil {
		println("gamepad found")
		in = tinypod.Inputs(gp, tinypod.Buttons(tinypod.Button{Pressed: pressed, Press: tinypod.Back}))
	}

	err = tinypod.Run(tinypod.Config{
		Display:   disp,
		Theme:     theme,
		Output:    out,
		Storage:   vol,
		Input:     in,
		ReadAhead: readAhead,
	})
	tinypod.Fail(disp, theme, "scan failed", err)
}

// mountCard uses the expansion board microSD slot. Init runs at 400 kHz, see
// the sd package Init docs.
func mountCard() (*tinypod.Volume, error) {
	cfg := machine.SPIConfig{
		SCK:       machine.D8,
		SDO:       machine.D10,
		SDI:       machine.D9,
		Frequency: 400 * machine.KHz,
	}
	if err := spi.Configure(cfg); err != nil {
		return nil, err
	}
	cs := machine.D2
	cs.Configure(machine.PinConfig{Mode: machine.PinOutput})
	return tinypod.MountSD(spi, cs.Set, func() error {
		cfg.Frequency = 20 * machine.MHz
		return spi.Configure(cfg)
	})
}
