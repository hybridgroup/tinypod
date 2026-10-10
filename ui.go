package tinypod

import (
	"math/rand"
	"strconv"
	"time"

	"github.com/hybridgroup/tinyplayer"
)

const (
	volumeStep    = 16
	volumeDefault = 160
	volumeShow    = 2 * time.Second
	loopDelay     = 30 * time.Millisecond
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
	vol        *Volume
	lib        *library
	deck       *deck
	player     *tinyplayer.Player
	in         Input
	stack      []*menu
	nowPlaying bool
	volume     int
	volumeAt   time.Time
	dirty      bool
	lastSong   string
	lastPaused bool
	lastSecond time.Duration
	volumeOn   bool
}

func newApp(scr *screen, vol *Volume, lib *library, d *deck, p *tinyplayer.Player, in Input) *app {
	a := &app{scr: scr, vol: vol, lib: lib, deck: d, player: p, in: in, volume: volumeDefault, dirty: true}
	p.SetVolume(a.volume)
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

// step polls the input, checks the player and the volume, and draws.
func (a *app) step() {
	if a.vol.settled() {
		a.reload()
	}
	if ev := a.in.Poll(); ev != None {
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
	if err := a.vol.reload(); err != nil {
		println("reload:", err.Error())
	}
	lib, err := scan(a.vol.fs)
	if err != nil {
		println("scan:", err.Error())
		lib = &library{}
	}
	a.lib = lib
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
		a.setVolume(a.volume + volumeStep)
		return
	case VolumeDown:
		a.setVolume(a.volume - volumeStep)
		return
	case Next:
		a.deck.skip(1)
		return
	case Previous:
		a.previous()
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
		a.setVolume(a.volume + volumeStep)
	case Down:
		a.setVolume(a.volume - volumeStep)
	case Right:
		a.deck.skip(1)
	case Left:
		a.previous()
	case Select:
		a.deck.togglePause()
	case Back:
		a.nowPlaying = false
	}
}

// previous goes to the start of the song, or to the song before in the first
// 3 seconds.
func (a *app) previous() {
	if el, _ := a.player.Progress(); el > 3*time.Second {
		a.deck.skip(0)
	} else {
		a.deck.skip(-1)
	}
}

func (a *app) setVolume(v int) {
	a.volume = max(0, min(256, v))
	a.player.SetVolume(a.volume)
	a.volumeAt = now()
}

func (a *app) draw() {
	if a.nowPlaying {
		a.drawNowPlaying()
	} else {
		a.drawMenu(a.stack[len(a.stack)-1])
	}
	a.scr.show()
}

func (a *app) titleBar(t string) {
	s := a.scr
	state := "s"
	if a.lastSong != "" {
		state = "p"
		if a.lastPaused {
			state = "z"
		}
	}
	s.row(0, "T"+state+t, s.t.BG, func(top int16) {
		size := s.ascent
		x := s.w - size - 2
		s.text(2, top, t, s.t.FG, x-2)
		s.rect(0, s.titleH-1, s.w, 1, s.t.SelBG)
		y := (s.titleH - 1 - size) / 2
		switch state {
		case "z":
			bar := max(size/3, 1)
			s.rect(x, y, bar, size, s.t.FG)
			s.rect(x+size-bar, y, bar, size, s.t.FG)
		case "p":
			for c := int16(0); c <= size/2; c++ {
				s.rect(x+c, y+c, 1, size-2*c, s.t.FG)
			}
		}
	})
}

func (a *app) drawMenu(m *menu) {
	s := a.scr
	a.titleBar(m.title)
	rows := s.rows() - 1
	if len(m.items) == 0 {
		lines := []string{"No songs"}
		if a.vol.USB() {
			lines = []string{"No songs.", "Copy WAV files", "to the USB drive."}
		}
		for i, l := range lines {
			s.row(1+i, "E"+l, s.t.BG, func(top int16) { s.text(2, top, l, s.t.FG, s.w) })
		}
		s.clearFrom(1 + len(lines))
		return
	}
	if m.sel < m.top {
		m.top = m.sel
	} else if m.sel >= m.top+rows {
		m.top = m.sel - rows + 1
	}
	n := len(m.items)
	ts, te := 0, rows
	if n > rows {
		ts = m.top * rows / n
		te = max(ts+1, (m.top+rows)*rows/n)
	}
	for r := 0; r < rows && m.top+r < n; r++ {
		i := m.top + r
		sel := i == m.sel
		thumb := n > rows && r >= ts && r < te
		key := "M" + flag(sel) + flag(thumb) + m.items[i]
		bg, fg := s.t.BG, s.t.FG
		if sel {
			bg, fg = s.t.SelBG, s.t.SelFG
		}
		s.row(1+r, key, bg, func(top int16) {
			s.text(2, top, m.items[i], fg, s.w-6)
			if thumb {
				s.rect(s.w-3, top, 3, s.rowH, s.t.FG)
			}
		})
	}
	s.clearFrom(1 + min(rows, n-m.top))
}

func (a *app) drawNowPlaying() {
	s := a.scr
	a.titleBar("Now Playing")
	song, pos, n := a.deck.current()
	if song == "" {
		s.row(1, "NNothing playing", s.t.BG, func(top int16) {
			s.text(2, top, "Nothing playing", s.t.FG, s.w)
		})
		s.clearFrom(2)
		return
	}
	lines := [...]string{strconv.Itoa(pos+1) + " of " + strconv.Itoa(n), title(song), albumOf(song)}
	for i, l := range lines {
		s.row(1+i, "N"+l, s.t.BG, func(top int16) { s.text(2, top, l, s.t.FG, s.w) })
	}

	el, total := a.player.Progress()
	barW := s.w - 4
	fill := int16(0)
	if total > 0 {
		fill = int16(int64(barW) * int64(min(el, total)) / int64(total))
	}
	s.row(4, "B"+strconv.Itoa(int(fill)), s.t.BG, func(top int16) {
		mid := top + s.rowH/2
		s.rect(2, mid, barW, 1, s.t.FG)
		s.rect(2, mid-1, fill, 3, s.t.SelBG)
	})

	if a.volumeOn {
		s.row(5, "V"+strconv.Itoa(a.volume), s.t.BG, func(top int16) {
			x := s.width("Vol ") + 2
			s.text(2, top, "Vol", s.t.FG, s.w)
			s.rect(x, top+s.rowH/2-2, (s.w-x-2)*int16(a.volume)/256, 4, s.t.SelBG)
		})
	} else {
		left, right := clock(el), "-"+clock(total-el)
		s.row(5, "C"+left+right, s.t.BG, func(top int16) {
			s.text(2, top, left, s.t.FG, s.w)
			x := s.w - 2 - s.width(right)
			s.text(x, top, right, s.t.FG, s.w)
		})
	}
	s.clearFrom(6)
}

func flag(b bool) string {
	if b {
		return "1"
	}
	return "0"
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
