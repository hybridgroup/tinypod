package tinypod

import (
	"math/rand"
	"time"
)

const (
	volumeShow = 2 * time.Second
	loopDelay  = 30 * time.Millisecond
)

type menu struct {
	title string
	items []string
	sel   int
	top   int
	open  func(i int)
}

// app runs the menus. It reads events from the input, sends them to the deck,
// and draws on the screen.
type app struct {
	scr        *screen
	in         Input
	lib        *library
	deck       *deck
	stack      []*menu
	nowPlaying bool
	volumeAt   time.Time
	dirty      bool
	lastSong   string
	lastPaused bool
	lastSecond time.Duration
	volumeOn   bool
}

func newApp(scr *screen, in Input, lib *library, d *deck) *app {
	a := &app{scr: scr, in: in, lib: lib, deck: d, dirty: true}
	a.home()
	return a
}

func (a *app) home() {
	a.stack = []*menu{{
		title: "tinypod",
		items: []string{"Songs", "Albums", "Shuffle Songs", "Now Playing"},
		open:  a.openMain,
	}}
	a.nowPlaying = false
}

func (a *app) openMain(i int) {
	switch i {
	case 0:
		a.push(a.songMenu("Songs", a.lib.songs))
	case 1:
		names := make([]string, len(a.lib.albums))
		for j, al := range a.lib.albums {
			names[j] = al.name
		}
		a.push(&menu{title: "Albums", items: names, open: func(j int) {
			al := a.lib.albums[j]
			a.push(a.songMenu(al.name, al.songs))
		}})
	case 2:
		if len(a.lib.songs) == 0 {
			return
		}
		q := append([]string(nil), a.lib.songs...)
		rand.Shuffle(len(q), func(i, j int) { q[i], q[j] = q[j], q[i] })
		a.deck.play(q, 0)
		a.nowPlaying = true
	case 3:
		a.nowPlaying = true
	}
}

func (a *app) songMenu(name string, songs []string) *menu {
	titles := make([]string, len(songs))
	for i, s := range songs {
		titles[i] = title(s)
	}
	return &menu{title: name, items: titles, open: func(i int) {
		a.deck.play(songs, i)
		a.nowPlaying = true
	}}
}

func (a *app) push(m *menu) {
	a.stack = append(a.stack, m)
}

func (a *app) run() {
	for {
		a.step()
		time.Sleep(loopDelay)
	}
}

// step polls the input, checks the deck and the library, and draws.
func (a *app) step() {
	if a.lib.changed() {
		a.reload()
	}
	if ev := a.in.Poll(); ev != None {
		a.handle(ev)
		a.dirty = true
	}
	song, _, _ := a.deck.current()
	paused := a.deck.paused()
	if song != a.lastSong || paused != a.lastPaused {
		a.lastSong, a.lastPaused = song, paused
		a.dirty = true
	}
	if a.nowPlaying {
		el, _ := a.deck.progress()
		if s := el / time.Second; s != a.lastSecond {
			a.lastSecond = s
			a.dirty = true
		}
		if on := now().Sub(a.volumeAt) < volumeShow; on != a.volumeOn {
			a.volumeOn = on
			a.dirty = true
		}
	}
	if a.dirty {
		a.dirty = false
		a.draw()
	}
}

// reload reads the songs again after the user copied files over USB.
func (a *app) reload() {
	a.deck.stop()
	a.scr.invalidate()
	Message(a.scr.d, a.scr.t, "tinypod", "reading songs")
	a.scr.invalidate()
	a.lib.reload()
	a.home()
	a.dirty = true
}

func (a *app) handle(ev Event) {
	switch ev {
	case PlayPause:
		if song, _, _ := a.deck.current(); song == "" && len(a.lib.songs) > 0 {
			a.deck.play(a.lib.songs, 0)
			a.nowPlaying = true
			return
		}
		a.deck.togglePause()
		return
	case VolumeUp:
		a.setVolume(a.deck.volume() + volumeStep)
		return
	case VolumeDown:
		a.setVolume(a.deck.volume() - volumeStep)
		return
	case Next:
		a.deck.skip(1)
		return
	case Previous:
		a.deck.previous()
		return
	}
	if a.nowPlaying {
		a.handleNowPlaying(ev)
		return
	}
	m := a.stack[len(a.stack)-1]
	switch ev {
	case Up:
		m.sel = max(m.sel-1, 0)
	case Down:
		m.sel = min(m.sel+1, len(m.items)-1)
	case Select, Right:
		if len(m.items) > 0 {
			m.open(m.sel)
		}
	case Left, Back:
		if len(a.stack) > 1 {
			a.stack = a.stack[:len(a.stack)-1]
		}
	}
}

func (a *app) handleNowPlaying(ev Event) {
	switch ev {
	case Up:
		a.setVolume(a.deck.volume() + volumeStep)
	case Down:
		a.setVolume(a.deck.volume() - volumeStep)
	case Right:
		a.deck.skip(1)
	case Left:
		a.deck.previous()
	case Select:
		a.deck.togglePause()
	case Back:
		a.nowPlaying = false
	}
}

func (a *app) setVolume(v int) {
	a.deck.setVolume(v)
	a.volumeAt = now()
}

func (a *app) state() playState {
	switch {
	case a.lastSong == "":
		return stopped
	case a.lastPaused:
		return paused
	}
	return playing
}

func (a *app) draw() {
	if a.nowPlaying {
		song, pos, n := a.deck.current()
		el, total := a.deck.progress()
		a.scr.nowPlaying(nowPlaying{
			song: song, pos: pos, n: n, state: a.state(),
			elapsed: el, total: total,
			volume: a.deck.volume(), showVolume: a.volumeOn,
		})
	} else {
		a.scr.menu(a.stack[len(a.stack)-1], a.state(), a.lib.usb())
	}
	a.scr.show()
}
