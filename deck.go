//go:build esp32c3 || rp2350

package main

import (
	"io"
	"sync"
	"sync/atomic"

	"github.com/hybridgroup/tinyplayer"
	"github.com/soypat/fat"
)

// deck plays a queue of songs in its own goroutine.
type deck struct {
	player *tinyplayer.Player
	fs     *fat.FS
	wake   chan struct{}
	f      fat.File
	cut    atomic.Bool

	mu     sync.Mutex
	queue  []string
	pos    int
	jump   int
	active bool
}

func newDeck(player *tinyplayer.Player, fs *fat.FS) *deck {
	d := &deck{player: player, fs: fs, wake: make(chan struct{}, 1), jump: -1}
	go d.run()
	return d
}

// play starts the queue at song i.
func (d *deck) play(queue []string, i int) {
	d.mu.Lock()
	d.queue = queue
	d.jump = i
	d.active = true
	d.cut.Store(true)
	d.mu.Unlock()
	d.restart()
}

// skip moves n songs forward or back in the queue.
func (d *deck) skip(n int) {
	d.mu.Lock()
	if len(d.queue) == 0 {
		d.mu.Unlock()
		return
	}
	d.jump = ((d.pos+n)%len(d.queue) + len(d.queue)) % len(d.queue)
	d.active = true
	d.cut.Store(true)
	d.mu.Unlock()
	d.restart()
}

func (d *deck) restart() {
	d.player.Pause(false)
	d.player.Stop()
	select {
	case d.wake <- struct{}{}:
	default:
	}
}

func (d *deck) togglePause() {
	d.mu.Lock()
	active := d.active
	d.mu.Unlock()
	if !active {
		d.skip(0)
		return
	}
	d.player.Pause(!d.player.Paused())
}

// current returns the song that is playing, its place in the queue, and the
// queue length. The song is empty when nothing plays.
func (d *deck) current() (song string, pos, n int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if !d.active || len(d.queue) == 0 {
		return "", 0, 0
	}
	return d.queue[d.pos], d.pos, len(d.queue)
}

// Read ends the song early once the UI asks for another one. Player.Stop
// alone can be missed since Play clears it when it starts.
func (d *deck) Read(b []byte) (int, error) {
	if d.cut.Load() {
		return 0, io.EOF
	}
	return d.f.Read(b)
}

func (d *deck) run() {
	for {
		d.mu.Lock()
		if d.jump >= 0 {
			d.pos = d.jump
			d.jump = -1
		}
		d.cut.Store(false)
		if !d.active || len(d.queue) == 0 {
			d.mu.Unlock()
			<-d.wake
			continue
		}
		song := d.queue[d.pos]
		d.mu.Unlock()

		if err := d.fs.OpenFile(&d.f, song, fat.ModeRead); err != nil {
			println("open", song, err.Error())
		} else {
			if err := d.player.Play(d); err != nil && !d.cut.Load() {
				println("play", song, err.Error())
			}
			d.f.Close()
		}

		d.mu.Lock()
		if d.jump < 0 {
			d.pos++
			if d.pos >= len(d.queue) {
				d.pos = 0
				d.active = false
			}
		}
		d.mu.Unlock()
	}
}
