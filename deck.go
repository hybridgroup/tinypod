package tinypod

import (
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hybridgroup/tinyplayer"
)

const (
	volumeStep    = 16
	volumeDefault = 160
)

// deck plays a queue of songs from the library in its own goroutine.
type deck struct {
	player    *tinyplayer.Player
	out       tinyplayer.Output
	on        bool
	lib       *library
	readAhead bool
	wake      chan struct{}
	f         io.ReadCloser
	vol       int
	ahead     *ahead
	cut       atomic.Bool
	idle      atomic.Bool

	mu     sync.Mutex
	queue  []string
	pos    int
	jump   int
	active bool
}

func newDeck(out tinyplayer.Output, lib *library, readAhead bool) *deck {
	d := &deck{player: tinyplayer.New(out), out: out, on: true, lib: lib, readAhead: readAhead, wake: make(chan struct{}, 1), jump: -1}
	d.setVolume(volumeDefault)
	if readAhead {
		d.ahead = &ahead{}
	}
	go d.run()
	return d
}

// stop ends playback, clears the queue and waits until no file is open.
func (d *deck) stop() {
	d.mu.Lock()
	d.queue = nil
	d.active = false
	d.cut.Store(true)
	d.mu.Unlock()
	d.restart()
	for !d.idle.Load() {
		time.Sleep(10 * time.Millisecond)
	}
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

// previous goes to the start of the song, or to the song before in the first
// 3 seconds.
func (d *deck) previous() {
	if el, _ := d.player.Progress(); el > 3*time.Second {
		d.skip(0)
	} else {
		d.skip(-1)
	}
}

func (d *deck) paused() bool {
	return d.player.Paused()
}

// progress returns the elapsed and total time of the song.
func (d *deck) progress() (elapsed, total time.Duration) {
	return d.player.Progress()
}

func (d *deck) volume() int {
	return d.vol
}

// setVolume sets the volume from 0 to 256.
func (d *deck) setVolume(v int) {
	d.vol = max(0, min(256, v))
	d.player.SetVolume(d.vol)
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
	if !d.readAhead {
		return d.f.Read(b)
	}
	return d.ahead.Read(b)
}

// power turns the output off while nothing plays, so speakers do not hiss.
func (d *deck) power(on bool) {
	if on != d.on {
		d.on = on
		d.out.Enable(on)
	}
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
			d.power(false)
			d.idle.Store(true)
			<-d.wake
			continue
		}
		d.idle.Store(false)
		song := d.queue[d.pos]
		d.mu.Unlock()

		d.power(true)
		f, err := d.lib.open(song)
		if err != nil {
			println("open", song, err.Error())
		} else {
			d.f = f
			if d.readAhead {
				d.ahead.start(f)
			}
			if err := d.player.Play(d); err != nil && !d.cut.Load() {
				println("play", song, err.Error())
			}
			if d.readAhead {
				d.ahead.halt()
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
