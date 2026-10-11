//go:build esp32c3

package main

import (
	"machine"

	"github.com/hybridgroup/tinyplayer"
	"github.com/hybridgroup/tinypod"
)

var (
	spi = machine.SPI0
	i2c = machine.I2C0
)

const readAhead = true

func audioOutput() (tinyplayer.Output, error) {
	err := machine.I2S0.Configure(machine.I2SConfig{
		SCK:            machine.D6,
		WS:             machine.D7,
		SDO:            machine.D0,
		SDI:            machine.NoPin,
		Mode:           machine.I2SModeSource,
		AudioFrequency: 44100,
		Stereo:         true,
	})
	return &machine.I2S0, err
}

// led returns nothing because the xiao-esp32c3 has no RGB LED.
func led() (tinypod.WS2812, int) {
	return nil, 0
}
