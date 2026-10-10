//go:build esp32c3 || rp2350

package main

import (
	"math/rand"
	"strconv"
	"time"

	"github.com/hybridgroup/tinyplayer"
)

const (
	rows       = 5
	rowHeight  = 10
	listTop    = 11
	volumeStep = 16
)

type menu struct {
	title string
	items []string
	sel   int
	top   int
	open  func(i int)
}

type app struct {
	scr        *screen
	lib        *library
	deck       *deck
	player     *tinyplayer.Player
	in         input
	stack      []*menu
	nowPlaying bool
	volume     int
	volumeAt   time.Time
	dirty      bool
	lastSong   string
	lastPaused bool
	lastSecond time.Duration
}

func newApp(scr *screen, lib *library, d *deck, p *tinyplayer.Player, in input) *app {
	a := &app{scr: scr, lib: lib, deck: d, player: p, in: in, volume: 160, dirty: true}
	p.SetVolume(a.volume)
	a.stack = []*menu{{
		title: "tinypod",
		items: []string{"Songs", "Albums", "Shuffle Songs", "Now Playing"},
		open:  a.openMain,
	}}
	return a
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
		if ev := a.in.poll(); ev != none {
			a.handle(ev)
			a.dirty = true
		}
		song, _, _ := a.deck.current()
		paused := a.player.Paused()
		if song != a.lastSong || paused != a.lastPaused {
			a.lastSong, a.lastPaused = song, paused
			a.dirty = true
		}
		if a.nowPlaying {
			el, _ := a.player.Progress()
			if s := el / time.Second; s != a.lastSecond {
				a.lastSecond = s
				a.dirty = true
			}
		}
		if a.dirty {
			a.dirty = false
			a.draw()
		}
		time.Sleep(30 * time.Millisecond)
	}
}

func (a *app) handle(ev event) {
	switch ev {
	case playPause:
		if song, _, _ := a.deck.current(); song == "" && len(a.lib.songs) > 0 {
			a.deck.play(a.lib.songs, 0)
			a.nowPlaying = true
			return
		}
		a.deck.togglePause()
		return
	case volumeUp:
		a.setVolume(a.volume + volumeStep)
		return
	case volumeDown:
		a.setVolume(a.volume - volumeStep)
		return
	}
	if a.nowPlaying {
		a.handleNowPlaying(ev)
		return
	}
	m := a.stack[len(a.stack)-1]
	switch ev {
	case up:
		m.sel = max(m.sel-1, 0)
	case down:
		m.sel = min(m.sel+1, len(m.items)-1)
	case selectItem, right:
		if len(m.items) > 0 {
			m.open(m.sel)
		}
	case left, back:
		if len(a.stack) > 1 {
			a.stack = a.stack[:len(a.stack)-1]
		}
	}
}

func (a *app) handleNowPlaying(ev event) {
	switch ev {
	case up:
		a.setVolume(a.volume + volumeStep)
	case down:
		a.setVolume(a.volume - volumeStep)
	case right:
		a.deck.skip(1)
	case left:
		if el, _ := a.player.Progress(); el > 3*time.Second {
			a.deck.skip(0)
		} else {
			a.deck.skip(-1)
		}
	case selectItem:
		a.deck.togglePause()
	case back:
		a.nowPlaying = false
	}
}

func (a *app) setVolume(v int) {
	a.volume = max(0, min(256, v))
	a.player.SetVolume(a.volume)
	a.volumeAt = time.Now()
}

func (a *app) draw() {
	s := a.scr
	s.clear()
	if a.nowPlaying {
		a.drawNowPlaying()
	} else {
		a.drawMenu(a.stack[len(a.stack)-1])
	}
	s.show()
}

func (a *app) titleBar(t string) {
	s := a.scr
	s.text(2, 8, t, white, 112)
	s.rect(0, 10, width, 1, white)
	if a.lastSong == "" {
		return
	}
	if a.lastPaused {
		s.rect(118, 1, 2, 7, white)
		s.rect(122, 1, 2, 7, white)
		return
	}
	for c := int16(0); c < 4; c++ {
		s.rect(118+c, 1+c, 1, 7-2*c, white)
	}
}

func (a *app) drawMenu(m *menu) {
	s := a.scr
	a.titleBar(m.title)
	if len(m.items) == 0 {
		s.text(2, listTop+18, "No songs", white, width)
		return
	}
	if m.sel < m.top {
		m.top = m.sel
	} else if m.sel >= m.top+rows {
		m.top = m.sel - rows + 1
	}
	for r := 0; r < rows && m.top+r < len(m.items); r++ {
		y := int16(listTop + 1 + r*rowHeight)
		c := white
		if m.top+r == m.sel {
			s.rect(0, y, width-4, rowHeight, white)
			c = black
		}
		s.text(2, y+8, m.items[m.top+r], c, width-6)
	}
	if n := len(m.items); n > rows {
		h := rows * rowHeight
		thumb := max(h*rows/n, 3)
		pos := (h - thumb) * m.top / (n - rows)
		s.rect(width-2, int16(listTop+1+pos), 2, int16(thumb), white)
	}
}

func (a *app) drawNowPlaying() {
	s := a.scr
	a.titleBar("Now Playing")
	song, pos, n := a.deck.current()
	if song == "" {
		s.text(2, 32, "Nothing playing", white, width)
		return
	}
	s.text(2, 21, strconv.Itoa(pos+1)+" of "+strconv.Itoa(n), white, width)
	s.text(2, 32, title(song), white, width)
	s.text(2, 43, albumOf(song), white, width)

	el, total := a.player.Progress()
	s.rect(2, 47, width-4, 1, white)
	s.rect(2, 51, width-4, 1, white)
	if total > 0 {
		s.rect(2, 48, int16(int64(width-4)*int64(el)/int64(total)), 3, white)
	}
	if time.Since(a.volumeAt) < 2*time.Second {
		s.text(2, 62, "Vol", white, width)
		s.rect(26, 56, int16((width-28)*a.volume/256), 5, white)
		return
	}
	s.text(2, 62, clock(el), white, width)
	rest := "-" + clock(total-el)
	s.text(width-2-int16(len(rest)*charWidth), 62, rest, white, width)
}

// clock formats d as m:ss.
func clock(d time.Duration) string {
	sec := int(max(d, 0) / time.Second)
	ss := strconv.Itoa(sec % 60)
	if len(ss) < 2 {
		ss = "0" + ss
	}
	return strconv.Itoa(sec/60) + ":" + ss
}
