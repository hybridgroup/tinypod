package tinypod

import (
	"time"

	"github.com/hybridgroup/tinyplayer/flashdisk"
	"github.com/hybridgroup/tinyplayer/storage"
	"github.com/soypat/fat"
	"tinygo.org/x/drivers"
	"tinygo.org/x/drivers/sd"
)

// Volume is the FAT volume that holds the songs.
type Volume struct {
	fs    *fat.FS
	flash *storage.Volume
	disk  *flashdisk.Disk

	last    uint32
	changed time.Time
}

// MountSD mounts an SD card read only. Configure spi at 100 to 400 kHz first,
// see the sd package Init docs. fast is called after Init to raise the clock.
func MountSD(spi drivers.SPI, cs func(bool), fast func() error) (*Volume, error) {
	cs(true)
	card := sd.NewSPICard(spi, cs)
	if err := card.Init(); err != nil {
		return nil, err
	}
	if fast != nil {
		if err := fast(); err != nil {
			return nil, err
		}
	}
	v := &Volume{fs: &fat.FS{}}
	if err := v.fs.Mount(card, 512, fat.ModeRead); err != nil {
		return nil, err
	}
	return v, nil
}

// MountFlash mounts a FAT volume in flash, and formats it if there is none.
// Show the disk as a USB drive with msc.Port so the user can copy songs to
// it. tinypod reads the songs again after the copy.
func MountFlash(disk *flashdisk.Disk) (*Volume, error) {
	vol, _, err := storage.Mount(disk)
	if err != nil {
		return nil, err
	}
	return &Volume{fs: &vol.FS, flash: vol, disk: disk, last: disk.Changed()}, nil
}

// USB reports if songs are copied to the volume over USB.
func (v *Volume) USB() bool {
	return v.disk != nil
}

const settleTime = time.Second

// settled reports true once after the USB host wrote to the disk and then
// stopped for a second.
func (v *Volume) settled() bool {
	if v.disk == nil {
		return false
	}
	t := now()
	if c := v.disk.Changed(); c != v.last {
		v.last = c
		v.changed = t
		return false
	}
	if v.changed.IsZero() || t.Sub(v.changed) < settleTime {
		return false
	}
	v.changed = time.Time{}
	return true
}

// reload writes cached sectors and mounts the volume again.
func (v *Volume) reload() error {
	if err := v.disk.Sync(); err != nil {
		return err
	}
	return v.flash.Remount()
}
