package progress

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Reader struct {
	r io.Reader
	n int64
}

func NewReader(r io.Reader) *Reader {
	return &Reader{r: r}
}

func (p *Reader) Read(b []byte) (int, error) {
	n, err := p.r.Read(b)
	atomic.AddInt64(&p.n, int64(n))
	return n, err
}

func (p *Reader) Bytes() int64 {
	return atomic.LoadInt64(&p.n)
}

func Track(label string, total int64, cur func() int64) func() {
	done := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		t := time.NewTicker(80 * time.Millisecond)
		defer t.Stop()
		for {
			select {
			case <-done:
				render(label, total, total)
				fmt.Fprintln(os.Stderr)
				return
			case <-t.C:
				render(label, cur(), total)
			}
		}
	}()

	return func() {
		close(done)
		wg.Wait()
	}
}

func render(label string, cur, total int64) {
	if total <= 0 {
		return
	}
	if cur > total {
		cur = total
	}
	const width = 30
	frac := float64(cur) / float64(total)
	fill := int(frac * width)
	bar := strings.Repeat("█", fill) + strings.Repeat("░", width-fill)
	fmt.Fprintf(os.Stderr, "\r  %s [%s] %3.0f%%  %s / %s   ", label, bar, frac*100, human(cur), human(total))
}

func human(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%dB", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%cB", float64(b)/float64(div), "KMGT"[exp])
}

func IsTerminal(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}
