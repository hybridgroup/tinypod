package tinypod

import (
	"io"
	"slices"
	"strings"
	"time"

	"github.com/hybridgroup/tinyplayer/flashdisk"
	"github.com/hybridgroup/tinyplayer/storage"
	"github.com/soypat/fat"
	"tinygo.org/x/drivers"
	"tinygo.org/x/drivers/sd"
)

// Storage holds the songs. The Library finds songs with List and the Deck
// reads them with Open.
type Storage interface {
	// List returns the sorted WAV paths and folder names in dir. The root is "".
	List(dir string) (songs, dirs []string, err error)
	// Open opens a song. Only one song is open at a time.
	Open(path string) (io.ReadCloser, error)
	// Changed reports true once after new songs were copied to the storage.
	Changed() bool
	// Reload makes the storage ready to read again after Changed.
	Reload() error
	// USB reports if songs are copied to the storage over USB.
	USB() bool
}

// Volume is a FAT volume on an SD card or in flash.
type Volume struct {
	fs    *fat.FS
	flash *storage.Volume
	disk  *flashdisk.Disk
	file  fat.File

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

func (v *Volume) List(dir string) (songs, dirs []string, err error) {
	var d fat.Dir
	path := dir
	if path == "" {
		path = "/"
	}
	if err := v.fs.OpenDir(&d, path); err != nil {
		return nil, nil, err
	}
	defer d.Close()
	err = d.ForEachFile(func(fi *fat.FileInfo) error {
		name := fi.Name()
		switch {
		case strings.HasPrefix(name, "."):
		case fi.IsDir():
			dirs = append(dirs, name)
		case strings.HasSuffix(strings.ToLower(name), ".wav"):
			songs = append(songs, dir+"/"+name)
		}
		return nil
	})
	slices.Sort(songs)
	slices.Sort(dirs)
	return songs, dirs, err
}

// Open reuses one fat.File so playing a song does not allocate.
func (v *Volume) Open(path string) (io.ReadCloser, error) {
	if err := v.fs.OpenFile(&v.file, path, fat.ModeRead); err != nil {
		return nil, err
	}
	return &v.file, nil
}

func (v *Volume) USB() bool {
	return v.disk != nil
}

const settleTime = time.Second

// Changed reports true once after the USB host wrote to the disk and then
// stopped for a second.
func (v *Volume) Changed() bool {
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

// Reload writes cached sectors and mounts the volume again.
func (v *Volume) Reload() error {
	if v.disk == nil {
		return nil
	}
	if err := v.disk.Sync(); err != nil {
		return err
	}
	return v.flash.Remount()
}
