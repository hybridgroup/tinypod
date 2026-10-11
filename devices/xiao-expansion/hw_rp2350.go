//go:build rp2350

package main

import (
	"machine"

	"github.com/hybridgroup/tinyplayer"
	"github.com/hybridgroup/tinypod"
	pio "github.com/tinygo-org/pio/rp2-pio"
	"github.com/tinygo-org/pio/rp2-pio/piolib"
	"tinygo.org/x/drivers/ws2812"
)

var (
	spi = machine.SPI0
	i2c = machine.I2C1
)

// piolib I2S queues one DMA buffer at a time, so a read ahead goroutine that
// holds the CPU in an SPI transfer makes the next buffer start late.
const readAhead = false

// audioOutput uses BCK on D6 and LCK on D7, which must be the pin after BCK.
func audioOutput() (tinyplayer.Output, error) {
	sm, err := pio.PIO0.ClaimStateMachine()
	if err != nil {
		return nil, err
	}
	i2s, err := piolib.NewI2S(sm, machine.D0, machine.D6)
	if err != nil {
		return nil, err
	}
	return i2s, i2s.SetSampleFrequency(44100)
}

// led is the RGB LED on the board. Call it after audioOutput so I2S gets its
// PIO state machine first.
func led() (tinypod.WS2812, int) {
	power := machine.NEOPIXEL_POWER
	power.Configure(machine.PinConfig{Mode: machine.PinOutput})
	power.High()
	return ws2812.NewWS2812(machine.WS2812), 1
}
