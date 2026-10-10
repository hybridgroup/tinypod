package tinypod

import (
	"image/color"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/hybridgroup/tinyplayer"
	"github.com/hybridgroup/tinyplayer/flashdisk"
)

type nor struct{ data []byte }

func (n *nor) ReadAt(p []byte, off int64) (int, error) { return copy(p, n.data[off:]), nil }
func (n *nor) WriteAt(p []byte, off int64) (int, error) {
	for i, b := range p {
		n.data[int(off)+i] &= b
	}
	return len(p), nil
}
func (n *nor) Size() int64           { return int64(len(n.data)) }
func (n *nor) WriteBlockSize() int64 { return 4 }
func (n *nor) EraseBlockSize() int64 { return 4096 }
func (n *nor) EraseBlocks(start, count int64) error {
	for i := start * 4096; i < (start+count)*4096; i++ {
		n.data[i] = 0xff
	}
	return nil
}

func newFlash(t *testing.T) (*flashdisk.Disk, *Volume) {
	t.Helper()
	dev := &nor{data: make([]byte, 1<<20)}
	for i := range dev.data {
		dev.data[i] = 0xff
	}
	disk, err := flashdisk.New(dev, 0, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	vol, err := MountFlash(disk)
	if err != nil {
		t.Fatal(err)
	}
	return disk, vol
}

func writeSongs(t *testing.T, vol *Volume, paths ...string) {
	t.Helper()
	data, err := os.ReadFile("../../wav/testdata/mono_s16_list.wav")
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range paths {
		for i := 1; i < len(p); i++ {
			if p[i] == '/' {
				vol.fs.Mkdir(p[:i])
			}
		}
		if err := vol.flash.WriteFile(p, string(data)); err != nil {
			t.Fatalf("write %s: %v", p, err)
		}
	}
}

func TestScan(t *testing.T) {
	_, vol := newFlash(t)
	writeSongs(t, vol,
		"/Music/b.wav",
		"/Music/Zed/2 two.wav",
		"/Music/Zed/1 one.wav",
		"/Music/Alpha/Disc 1/a.wav",
		"/Music/Empty/notes.txt",
	)
	lib, err := scan(vol.fs)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, a := range lib.albums {
		names = append(names, a.name)
	}
	if !slices.Equal(names, []string{"Alpha", "Zed"}) {
		t.Fatalf("albums %v", names)
	}
	if !slices.Equal(lib.albums[1].songs, []string{"/Music/Zed/1 one.wav", "/Music/Zed/2 two.wav"}) {
		t.Fatalf("Zed songs %v", lib.albums[1].songs)
	}
	var titles []string
	for _, s := range lib.songs {
		titles = append(titles, title(s))
	}
	if !slices.Equal(titles, []string{"1 one", "2 two", "a", "b"}) {
		t.Fatalf("songs %v", titles)
	}
	if albumOf("/Music/Alpha/Disc 1/a.wav") != "Disc 1" {
		t.Fatal("albumOf")
	}
}

func TestScanRoot(t *testing.T) {
	_, vol := newFlash(t)
	writeSongs(t, vol, "/x.wav", "/Band/y.wav")
	lib, err := scan(vol.fs)
	if err != nil {
		t.Fatal(err)
	}
	if len(lib.songs) != 2 || len(lib.albums) != 1 || lib.albums[0].name != "Band" {
		t.Fatalf("songs %v albums %d", lib.songs, len(lib.albums))
	}
}

type fakeDisplay struct {
	w, h   int16
	pixels map[[2]int16]color.RGBA
	frames int
}

func newFakeDisplay(w, h int16) *fakeDisplay {
	return &fakeDisplay{w: w, h: h, pixels: map[[2]int16]color.RGBA{}}
}

func (d *fakeDisplay) Size() (int16, int16) { return d.w, d.h }
func (d *fakeDisplay) SetPixel(x, y int16, c color.RGBA) {
	d.pixels[[2]int16{x, y}] = c
}
func (d *fakeDisplay) Display() error { d.frames++; return nil }

func TestLayout(t *testing.T) {
	for _, tc := range []struct {
		w, h  int16
		theme Theme
		rows  int
	}{
		{128, 64, MonoTheme(), 6},
		{320, 240, ColorTheme(), 15},
	} {
		s := newScreen(newFakeDisplay(tc.w, tc.h), tc.theme)
		if s.rows() != tc.rows {
			t.Errorf("%dx%d: %d rows, want %d", tc.w, tc.h, s.rows(), tc.rows)
		}
		if end := s.top(s.rows()); end > tc.h {
			t.Errorf("%dx%d: rows end at %d", tc.w, tc.h, end)
		}
	}
}

func TestRowSkip(t *testing.T) {
	d := newFakeDisplay(128, 64)
	s := newScreen(d, MonoTheme())
	drawn := 0
	for i := 0; i < 2; i++ {
		s.row(1, "same", s.t.BG, func(int16) { drawn++ })
		s.show()
	}
	if drawn != 1 || d.frames != 1 {
		t.Fatalf("drawn %d frames %d", drawn, d.frames)
	}
}

type fakeClock struct{ t time.Time }

func (c *fakeClock) now() time.Time      { return c.t }
func (c *fakeClock) add(d time.Duration) { c.t = c.t.Add(d) }
func useClock(t *testing.T) *fakeClock {
	c := &fakeClock{t: time.Unix(1000, 0)}
	now = c.now
	t.Cleanup(func() { now = time.Now })
	return c
}

func TestButtons(t *testing.T) {
	c := useClock(t)
	var up, sel bool
	in := Buttons(
		Button{Pressed: func() bool { return up }, Press: Up, Repeat: true},
		Button{Pressed: func() bool { return sel }, Press: Select, Hold: PlayPause},
	)
	step := func(d time.Duration) Event { c.add(d); return in.Poll() }

	up = true
	if e := step(0); e != Up {
		t.Fatalf("press: %v", e)
	}
	if e := step(300 * time.Millisecond); e != None {
		t.Fatalf("before repeat: %v", e)
	}
	if e := step(100 * time.Millisecond); e != Up {
		t.Fatalf("repeat: %v", e)
	}
	if e := step(130 * time.Millisecond); e != Up {
		t.Fatalf("repeat rate: %v", e)
	}
	up = false
	step(0)

	sel = true
	if e := step(0); e != None {
		t.Fatalf("hold button down: %v", e)
	}
	sel = false
	if e := step(100 * time.Millisecond); e != Select {
		t.Fatalf("short press: %v", e)
	}
	sel = true
	step(0)
	if e := step(600 * time.Millisecond); e != PlayPause {
		t.Fatalf("hold: %v", e)
	}
	sel = false
	if e := step(10 * time.Millisecond); e != None {
		t.Fatalf("release after hold: %v", e)
	}
}

func TestHoldButton(t *testing.T) {
	c := useClock(t)
	var down bool
	in := HoldButton(func() bool { return down })
	press := func(d time.Duration) Event {
		down = true
		in.Poll()
		c.add(d)
		down = false
		return in.Poll()
	}
	if e := press(100 * time.Millisecond); e != Down {
		t.Fatalf("tap: %v", e)
	}
	if e := press(700 * time.Millisecond); e != Select {
		t.Fatalf("hold: %v", e)
	}
	if e := press(2 * time.Second); e != Back {
		t.Fatalf("long hold: %v", e)
	}
}

func TestSettled(t *testing.T) {
	c := useClock(t)
	disk, vol := newFlash(t)
	if vol.settled() {
		t.Fatal("settled with no writes")
	}
	var b [512]byte
	disk.WriteBlocks(b[:], 10)
	if vol.settled() {
		t.Fatal("settled right after a write")
	}
	c.add(500 * time.Millisecond)
	if vol.settled() {
		t.Fatal("settled too early")
	}
	c.add(time.Second)
	if !vol.settled() {
		t.Fatal("not settled")
	}
	if vol.settled() {
		t.Fatal("settled twice")
	}
}

type fakeOutput struct{}

func (fakeOutput) SetSampleFrequency(uint32) error { return nil }
func (fakeOutput) Enable(bool)                     {}
func (fakeOutput) WriteStereo(b []uint32) (int, error) {
	time.Sleep(50 * time.Millisecond)
	return len(b), nil
}

type script []Event

func (s *script) Poll() Event {
	if len(*s) == 0 {
		return None
	}
	e := (*s)[0]
	*s = (*s)[1:]
	return e
}

func TestUI(t *testing.T) {
	_, vol := newFlash(t)
	writeSongs(t, vol, "/Music/A/one.wav", "/Music/A/two.wav", "/Music/B/three.wav")
	lib, err := scan(vol.fs)
	if err != nil {
		t.Fatal(err)
	}
	player := tinyplayer.New(fakeOutput{})
	d := newDeck(player, vol.fs, true)
	in := &script{}
	a := newApp(newScreen(newFakeDisplay(128, 64), MonoTheme()), vol, lib, d, player, in)
	run := func(evs ...Event) {
		*in = append(*in, evs...)
		for len(*in) > 0 {
			a.step()
		}
	}

	run(Down, Select)
	if top := a.stack[len(a.stack)-1]; top.title != "Albums" || len(top.items) != 2 {
		t.Fatalf("albums menu %q %v", top.title, top.items)
	}
	run(Select)
	if top := a.stack[len(a.stack)-1]; top.title != "A" || !slices.Equal(top.items, []string{"one", "two"}) {
		t.Fatalf("album menu %q %v", top.title, top.items)
	}
	run(Down, Select)
	if !a.nowPlaying {
		t.Fatal("not on Now Playing")
	}
	waitSong(t, d, "/Music/A/two.wav")

	run(Up)
	if a.volume != volumeDefault+volumeStep {
		t.Fatalf("volume %d", a.volume)
	}
	run(Select)
	if !player.Paused() {
		t.Fatal("not paused")
	}
	run(Select, Back, Back, Back)
	if a.nowPlaying || len(a.stack) != 1 {
		t.Fatalf("now playing %v stack %d", a.nowPlaying, len(a.stack))
	}
	d.stop()
	if song, _, _ := d.current(); song != "" {
		t.Fatalf("still playing %s", song)
	}
}

func waitSong(t *testing.T, d *deck, want string) {
	t.Helper()
	for i := 0; i < 100; i++ {
		if song, _, _ := d.current(); song == want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	song, _, _ := d.current()
	t.Fatalf("playing %q, want %q", song, want)
}
