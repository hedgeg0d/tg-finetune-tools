package model

import (
	"bufio"
	"encoding/json"
	"io"
)

type Message struct {
	ID     int64  `json:"id"`
	Date   int64  `json:"date"`
	FromID string `json:"from_id"`
	From   string `json:"from"`
	Text   string `json:"text"`
}

func ReadAll(r io.Reader) ([]Message, error) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)

	var out []Message
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		var m Message
		if err := json.Unmarshal(line, &m); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, sc.Err()
}
