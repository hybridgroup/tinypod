//go:build esp32c3 || rp2350

package main

import (
	"machine"

	"github.com/soypat/fat"
	"tinygo.org/x/drivers/sd"
)

// mountCard mounts the FAT volume on the expansion board microSD slot read
// only. Init must run at 100 to 400 kHz, see the sd package Init docs.
func mountCard(fs *fat.FS) error {
	cfg := machine.SPIConfig{
		SCK:       machine.D8,
		SDO:       machine.D10,
		SDI:       machine.D9,
		Frequency: 400 * machine.KHz,
	}
	if err := spi.Configure(cfg); err != nil {
		return err
	}
	cs := machine.D2
	cs.Configure(machine.PinConfig{Mode: machine.PinOutput})
	cs.High()
	card := sd.NewSPICard(spi, cs.Set)
	if err := card.Init(); err != nil {
		return err
	}
	cfg.Frequency = 20 * machine.MHz
	if err := spi.Configure(cfg); err != nil {
		return err
	}
	return fs.Mount(card, 512, fat.ModeRead)
}

