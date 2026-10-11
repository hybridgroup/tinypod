package tinypod

import (
	"image/color"
	"strconv"
	"time"

	"tinygo.org/x/tinyfont"
	"tinygo.org/x/tinyfont/notosans"
	"tinygo.org/x/tinyfont/proggy"
)

// Display is a screen driver such as ssd1306.Device or st7789.Device.
// tinypod also uses FillRectangle when the driver has it.
type Display interface {
	Size() (w, h int16)
	SetPixel(x, y int16, c color.RGBA)
	Display() error
}

type filler interface {
	FillRectangle(x, y, w, h int16, c color.RGBA) error
}

// Theme sets the font and colors. Sel colors are for the selected row, the
// time bar and the icons.
type Theme struct {
	Font  *tinyfont.Font
	FG    color.RGBA
	BG    color.RGBA
	SelFG color.RGBA
	SelBG color.RGBA
}

var (
	white = color.RGBA{255, 255, 255, 255}
	black = color.RGBA{0, 0, 0, 255}
)

// MonoTheme suits small monochrome screens such as a 128x64 SSD1306.
func MonoTheme() Theme {
	return Theme{Font: &proggy.TinySZ8pt7b, FG: white, BG: black, SelFG: black, SelBG: white}
}

// ColorTheme suits color screens of about 240x240.
func ColorTheme() Theme {
	return Theme{
		Font:  &notosans.Notosans12pt,
		FG:    white,
		BG:    color.RGBA{16, 24, 48, 255},
		SelFG: white,
		SelBG: color.RGBA{0, 110, 200, 255},
	}
}

const maxRows = 64

// screen draws rows of text. Row 0 is the title bar, with a line under it.
// It keeps a key for each row and skips rows that did not change, because
// unbuffered displays write each pixel over SPI.
type screen struct {
	d      Display
	t      Theme
	w, h   int16
	rowH   int16
	titleH int16
	ascent int16
	keys   [maxRows]string
	bottom string
	dirty  bool
}

func newScreen(d Display, t Theme) *screen {
	w, h := d.Size()
	s := &screen{d: d, t: t, w: w, h: h}
	s.rowH = int16(t.Font.BBox[1])
	s.titleH = s.rowH + 2
	s.ascent = -int16(t.Font.BBox[3])
	s.invalidate()
	return s
}

func (s *screen) rows() int {
	return min(1+int((s.h-s.titleH)/s.rowH), maxRows)
}

func (s *screen) top(i int) int16 {
	if i == 0 {
		return 0
	}
	return s.titleH + int16(i-1)*s.rowH
}

func (s *screen) height(i int) int16 {
	if i == 0 {
		return s.titleH
	}
	return s.rowH
}

// invalidate makes the next frame draw every row.
func (s *screen) invalidate() {
	for i := range s.keys {
		s.keys[i] = "\x00"
	}
	s.bottom = "\x00"
}

// row draws row i with draw if key differs from what the row shows now.
func (s *screen) row(i int, key string, bg color.RGBA, draw func(top int16)) {
	if i >= s.rows() || s.keys[i] == key {
		return
	}
	s.keys[i] = key
	top := s.top(i)
	s.rect(0, top, s.w, s.height(i), bg)
	if draw != nil {
		draw(top)
	}
	s.dirty = true
}

// clearFrom blanks rows from i to the bottom, and the space below the last row.
func (s *screen) clearFrom(i int) {
	for ; i < s.rows(); i++ {
		s.row(i, "", s.t.BG, nil)
	}
	if end := s.top(s.rows()); end < s.h && s.bottom != "" {
		s.bottom = ""
		s.rect(0, end, s.w, s.h-end, s.t.BG)
		s.dirty = true
	}
}

// show sends the frame if a row changed.
func (s *screen) show() {
	if s.dirty {
		s.dirty = false
		s.d.Display()
	}
}

func (s *screen) rect(x, y, w, h int16, c color.RGBA) {
	if x < 0 {
		w, x = w+x, 0
	}
	if y < 0 {
		h, y = h+y, 0
	}
	w, h = min(w, s.w-x), min(h, s.h-y)
	if w <= 0 || h <= 0 {
		return
	}
	if f, ok := s.d.(filler); ok {
		f.FillRectangle(x, y, w, h, c)
		return
	}
	for j := y; j < y+h; j++ {
		for i := x; i < x+w; i++ {
			s.d.SetPixel(i, j, c)
		}
	}
}

// text draws str in the row that starts at top, cut to end before maxX.
func (s *screen) text(x, top int16, str string, c color.RGBA, maxX int16) {
	str = s.fit(str, maxX-x)
	tinyfont.WriteLine(s.d, s.t.Font, x, top+s.ascent, str, c)
}

func (s *screen) width(str string) int16 {
	_, w := tinyfont.LineWidth(s.t.Font, str)
	return int16(w)
}

func (s *screen) fit(str string, max int16) string {
	for len(str) > 0 && s.width(str) > max {
		str = str[:len(str)-1]
	}
	return str
}

// Message shows a few lines of text, for example while starting or after an
// error.
func Message(d Display, t Theme, lines ...string) {
	s := newScreen(d, t)
	for i, l := range lines {
		s.row(i, l, t.BG, func(top int16) { s.text(2, top, l, t.FG, s.w) })
	}
	s.clearFrom(len(lines))
	s.show()
}

// playState is the icon in the title bar.
type playState uint8

const (
	stopped playState = iota
	playing
	paused
)

func (s *screen) titleBar(t string, st playState) {
	s.row(0, "T"+strconv.Itoa(int(st))+t, s.t.BG, func(top int16) {
		size := s.ascent
		x := s.w - size - 2
		s.text(2, top, t, s.t.FG, x-2)
		s.rect(0, s.titleH-1, s.w, 1, s.t.SelBG)
		y := (s.titleH - 1 - size) / 2
		switch st {
		case paused:
			bar := max(size/3, 1)
			s.rect(x, y, bar, size, s.t.FG)
			s.rect(x+size-bar, y, bar, size, s.t.FG)
		case playing:
			for c := int16(0); c <= size/2; c++ {
				s.rect(x+c, y+c, 1, size-2*c, s.t.FG)
			}
		}
	})
}

// menu draws a list with the selected row and a scroll bar. It moves m.top so
// the selected row is on the screen.
func (s *screen) menu(m *menu, st playState, usb bool) {
	s.titleBar(m.title, st)
	rows := s.rows() - 1
	if len(m.items) == 0 {
		lines := []string{"No songs"}
		if usb {
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

// nowPlaying is what the Now Playing screen shows. song is empty when
// nothing plays.
type nowPlaying struct {
	song           string
	pos, n         int
	state          playState
	elapsed, total time.Duration
	volume         int
	showVolume     bool
}

func (s *screen) nowPlaying(np nowPlaying) {
	s.titleBar("Now Playing", np.state)
	if np.song == "" {
		s.row(1, "NNothing playing", s.t.BG, func(top int16) {
			s.text(2, top, "Nothing playing", s.t.FG, s.w)
		})
		s.clearFrom(2)
		return
	}
	lines := [...]string{strconv.Itoa(np.pos+1) + " of " + strconv.Itoa(np.n), title(np.song), albumOf(np.song)}
	for i, l := range lines {
		s.row(1+i, "N"+l, s.t.BG, func(top int16) { s.text(2, top, l, s.t.FG, s.w) })
	}

	el, total := np.elapsed, np.total
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

	if np.showVolume {
		s.row(5, "V"+strconv.Itoa(np.volume), s.t.BG, func(top int16) {
			x := s.width("Vol ") + 2
			s.text(2, top, "Vol", s.t.FG, s.w)
			s.rect(x, top+s.rowH/2-2, (s.w-x-2)*int16(np.volume)/256, 4, s.t.SelBG)
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
