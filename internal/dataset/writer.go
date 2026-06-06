package dataset

import (
	"bufio"
	"encoding/json"
	"io"
)

type Writer struct {
	enc *json.Encoder
	buf *bufio.Writer
}

func NewWriter(w io.Writer) *Writer {
	buf := bufio.NewWriterSize(w, 1<<20)
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	return &Writer{enc: enc, buf: buf}
}

func (w *Writer) Write(v any) error {
	return w.enc.Encode(v)
}

func (w *Writer) Flush() error {
	return w.buf.Flush()
}
