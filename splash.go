package tinypod

import (
	"image/color"
	"time"
)

const splashTime = 1500 * time.Millisecond

var splashAt time.Time

// Splash shows a tinypod icon and name. Run keeps it on the screen for at
// least 1.5 seconds while it scans the songs.
func Splash(d Display, t Theme) {
	s := newScreen(d, t)
	s.rect(0, 0, s.w, s.h, t.BG)

	ih := min(s.h-s.rowH-8, s.h*6/10)
	iw := ih * 5 / 8
	top := (s.h - ih - 4 - s.rowH) / 2
	x := (s.w - iw) / 2
	screenColor := t.SelBG
	if screenColor == t.FG {
		screenColor = t.BG
	}

	r := max(iw/8, 1)
	s.rect(x+r, top, iw-2*r, ih, t.FG)
	s.rect(x, top+r, iw, ih-2*r, t.FG)
	for _, c := range [...][2]int16{{x + r, top + r}, {x + iw - 1 - r, top + r}, {x + r, top + ih - 1 - r}, {x + iw - 1 - r, top + ih - 1 - r}} {
		s.circle(c[0], c[1], r, t.FG)
	}

	m := max(iw/10, 1)
	s.rect(x+m, top+m, iw-2*m, ih*38/100, screenColor)

	wr := iw * 32 / 100
	cx, cy := x+iw/2, top+ih*70/100
	s.circle(cx, cy, wr, t.BG)
	s.circle(cx, cy, wr*38/100, t.FG)

	name := "tinypod"
	s.text((s.w-s.width(name))/2, top+ih+4, name, t.FG, s.w)
	d.Display()
	splashAt = now()
}

// circle fills a circle one line at a time, so displays with FillRectangle
// draw it fast. It uses r+0.5 for the width, which rounds off the tips.
func (s *screen) circle(cx, cy, r int16, c color.RGBA) {
	for dy := -r; dy <= r; dy++ {
		dx := isqrt(int(r)*int(r) + int(r) - int(dy)*int(dy))
		s.rect(cx-int16(dx), cy+dy, int16(2*dx+1), 1, c)
	}
}

func isqrt(n int) int {
	x := 0
	for (x+1)*(x+1) <= n {
		x++
	}
	return x
}

// waitSplash keeps the splash on the screen until it was shown for
// splashTime.
func waitSplash() {
	if splashAt.IsZero() {
		return
	}
	if left := splashTime - now().Sub(splashAt); left > 0 {
		time.Sleep(left)
	}
	splashAt = time.Time{}
}
