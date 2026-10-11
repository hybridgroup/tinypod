//go:build gopher_badge

// tinypod on the gopher badge. It uses the 320x240 color screen, the six
// buttons and the speaker. Songs are in flash, which shows up as a USB drive.
package main

import (
	"machine"
	"machine/usb/msc"
	"time"

	"github.com/hybridgroup/tinyplayer/flashdisk"
	"github.com/hybridgroup/tinyplayer/pwm"
	"github.com/hybridgroup/tinypod"
	"tinygo.org/x/drivers/st7789"
	"tinygo.org/x/drivers/ws2812"
)

const diskSize = 6 << 20

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

	machine.SPI0.Configure(machine.SPIConfig{Frequency: 16 * machine.MHz, Mode: 0})
	disp := st7789.New(machine.SPI0, machine.TFT_RST, machine.TFT_WRX, machine.TFT_CS, machine.TFT_BACKLIGHT)
	disp.Configure(st7789.Config{Rotation: st7789.ROTATION_270, Height: 320})
	theme := tinypod.ColorTheme()
	tinypod.Splash(&disp, theme)

	if diskErr != nil {
		tinypod.Fail(&disp, theme, "no flash volume", diskErr)
	}

	enable := machine.SPEAKER_ENABLE
	enable.Configure(machine.PinConfig{Mode: machine.PinOutput})
	enable.High()
	p, err := pwm.New(machine.SPEAKER, machine.NoPin, 11)
	if err != nil {
		tinypod.Fail(&disp, theme, "no audio", err)
	}
	out := &speaker{PWM: p, enable: enable}

	err = tinypod.Run(tinypod.Config{
		Display:  &disp,
		Theme:    theme,
		Output:   out,
		Storage:  vol,
		LED:      ws2812.NewWS2812(machine.WS2812),
		LEDCount: 2,
		Input: tinypod.Buttons(
			tinypod.Button{Pressed: button(machine.BUTTON_UP), Press: tinypod.Up, Repeat: true},
			tinypod.Button{Pressed: button(machine.BUTTON_DOWN), Press: tinypod.Down, Repeat: true},
			tinypod.Button{Pressed: button(machine.BUTTON_LEFT), Press: tinypod.Left},
			tinypod.Button{Pressed: button(machine.BUTTON_RIGHT), Press: tinypod.Right},
			tinypod.Button{Pressed: button(machine.BUTTON_A), Press: tinypod.Select, Hold: tinypod.PlayPause},
			tinypod.Button{Pressed: button(machine.BUTTON_B), Press: tinypod.Back},
		),
	})
	tinypod.Fail(&disp, theme, "scan failed", err)
}

// speaker turns the amplifier off with the output, so it does not hiss while
// nothing plays.
type speaker struct {
	*pwm.PWM
	enable machine.Pin
}

func (s *speaker) Enable(on bool) {
	s.PWM.Enable(on)
	s.enable.Set(on)
}

// button is a badge button, which reads low when pressed.
func button(pin machine.Pin) func() bool {
	pin.Configure(machine.PinConfig{Mode: machine.PinInputPullup})
	return func() bool { return !pin.Get() }
}
