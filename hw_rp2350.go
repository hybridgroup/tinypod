//go:build rp2350

package main

import (
	"machine"

	"github.com/hybridgroup/tinyplayer"
	pio "github.com/tinygo-org/pio/rp2-pio"
	"github.com/tinygo-org/pio/rp2-pio/piolib"
)

var (
	spi = machine.SPI0
	i2c = machine.I2C1
)

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
