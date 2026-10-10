//go:build esp32c3

package main

import (
	"machine"

	"github.com/hybridgroup/tinyplayer"
)

var (
	spi = machine.SPI0
	i2c = machine.I2C0
)

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
