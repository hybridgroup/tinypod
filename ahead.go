package tinypod

import (
	"io"
	"runtime"
	"sync/atomic"
	"time"
)

const (
	aheadSize  = 32 << 10
	aheadChunk = 2048
)

// ahead reads a song into a ring buffer in its own goroutine, so slow SD reads
// and screen updates do not starve the audio output.
type ahead struct {
	buf  [aheadSize]byte
	r, w atomic.Uint32
	done atomic.Bool
	stop atomic.Bool
	err  error
}

func (a *ahead) start(f io.Reader) {
	a.r.Store(0)
	a.w.Store(0)
	a.done.Store(false)
	a.stop.Store(false)
	a.err = nil
	go a.fill(f)
}

// halt stops the fill goroutine and waits for it to end.
func (a *ahead) halt() {
	a.stop.Store(true)
	for !a.done.Load() {
		time.Sleep(time.Millisecond)
	}
}

func (a *ahead) fill(f io.Reader) {
	defer a.done.Store(true)
	for !a.stop.Load() {
		w := a.w.Load()
		if aheadSize-int(w-a.r.Load()) < aheadChunk {
			time.Sleep(2 * time.Millisecond)
			continue
		}
		off := int(w % aheadSize)
		n, err := f.Read(a.buf[off : off+aheadChunk])
		a.w.Add(uint32(n))
		if err != nil {
			if err != io.EOF {
				a.err = err
			}
			return
		}
		if n == 0 {
			return
		}
	}
}

func (a *ahead) Read(b []byte) (int, error) {
	for {
		r, w := a.r.Load(), a.w.Load()
		if w != r {
			off := int(r % aheadSize)
			n := copy(b, a.buf[off:min(off+int(w-r), aheadSize)])
			a.r.Add(uint32(n))
			return n, nil
		}
		if a.done.Load() && a.w.Load() == r {
			if a.err != nil {
				return 0, a.err
			}
			return 0, io.EOF
		}
		runtime.Gosched()
	}
}
