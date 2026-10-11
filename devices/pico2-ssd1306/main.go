//go:build pico2

// tinypod on a Pico 2 with an SSD1306, four buttons and an I2S DAC. Songs are
// in flash, which shows up as a USB drive.
package main

import (
	"machine"
	"machine/usb/msc"
	"time"

	"github.com/hybridgroup/tinyplayer/flashdisk"
	"github.com/hybridgroup/tinypod"
	pio "github.com/tinygo-org/pio/rp2-pio"
	"github.com/tinygo-org/pio/rp2-pio/piolib"
	"tinygo.org/x/drivers/ws2812"
)

const (
	diskSize = 3 << 20
	ledCount = 22
)

func main() {
	// MSC replaces the USB descriptor, so it must be set up before the host
	// enumerates, see TinyGo src/machine/usb.go ConfigureUSBEndpoint. Mount
	// first so a new volume is formatted before the host looks at it.
	var vol *tinypod.Volume
	disk, diskErr := flashdisk.Tail(machine.Flash, diskSize)
	if diskErr == nil {
		m := msc.Port(disk)
		m.SetVendorID("TinyGo")
		m.SetProductID("tinypod")
		vol, diskErr = tinypod.MountFlash(disk)
	}
	machine.USBDev.Configure(machine.UARTConfig{})
	time.Sleep(time.Second)

	i2c := machine.I2C1
	i2c.Configure(machine.I2CConfig{SDA: machine.GPIO6, SCL: machine.GPIO7, Frequency: 400 * machine.KHz})
	disp := tinypod.NewSSD1306(i2c, 0x3C, 128, 64)
	theme := tinypod.MonoTheme()
	tinypod.Splash(disp, theme)

	if diskErr != nil {
		tinypod.Fail(disp, theme, "no flash volume", diskErr)
	}

	out, err := audioOutput()
	if err != nil {
		tinypod.Fail(disp, theme, "no audio", err)
	}
	leds := ws2812.NewWS2812(machine.GPIO15)

	err = tinypod.Run(tinypod.Config{
		Display:  disp,
		Theme:    theme,
		Output:   out,
		Storage:  vol,
		LED:      leds,
		LEDCount: ledCount,
		Input: tinypod.Buttons(
			tinypod.Button{Pressed: button(machine.GPIO10), Press: tinypod.Up, Repeat: true},
			tinypod.Button{Pressed: button(machine.GPIO11), Press: tinypod.Down, Repeat: true},
			tinypod.Button{Pressed: button(machine.GPIO12), Press: tinypod.Select, Hold: tinypod.PlayPause},
			tinypod.Button{Pressed: button(machine.GPIO13), Press: tinypod.Back, Hold: tinypod.Next},
		),
	})
	tinypod.Fail(disp, theme, "scan failed", err)
}

// button is a button from pin to GND.
func button(pin machine.Pin) func() bool {
	pin.Configure(machine.PinConfig{Mode: machine.PinInputPullup})
	return func() bool { return !pin.Get() }
}

// audioOutput uses DIN on GPIO2, BCK on GPIO3 and LCK on GPIO4.
func audioOutput() (*piolib.I2S, error) {
	sm, err := pio.PIO0.ClaimStateMachine()
	if err != nil {
		return nil, err
	}
	i2s, err := piolib.NewI2S(sm, machine.GPIO2, machine.GPIO3)
	if err != nil {
		return nil, err
	}
	return i2s, i2s.SetSampleFrequency(44100)
}
