package telegram

import (
	"encoding/json"
	"fmt"
	"io"
)

func Stream(r io.Reader, emit func(RawMessage) error) error {
	dec := json.NewDecoder(r)

	if _, err := dec.Token(); err != nil {
		return fmt.Errorf("read root token: %w", err)
	}

	for dec.More() {
		key, err := dec.Token()
		if err != nil {
			return fmt.Errorf("read field key: %w", err)
		}

		if key != "messages" {
			var skip json.RawMessage
			if err := dec.Decode(&skip); err != nil {
				return fmt.Errorf("skip field %v: %w", key, err)
			}
			continue
		}

		if _, err := dec.Token(); err != nil {
			return fmt.Errorf("read messages open: %w", err)
		}

		for dec.More() {
			var m RawMessage
			if err := dec.Decode(&m); err != nil {
				return fmt.Errorf("decode message: %w", err)
			}
			if err := emit(m); err != nil {
				return err
			}
		}

		if _, err := dec.Token(); err != nil {
			return fmt.Errorf("read messages close: %w", err)
		}
	}

	return nil
}
