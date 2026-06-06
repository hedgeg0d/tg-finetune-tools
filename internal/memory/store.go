package memory

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"time"
)

func loadJSON[T any](path string) (T, error) {
	var v T
	if path == "" {
		return v, nil
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return v, nil
	}
	if err != nil {
		return v, err
	}
	err = json.Unmarshal(data, &v)
	return v, err
}

func saveJSON(path string, v any) error {
	if path == "" {
		return nil
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func dateStr(unix int64) string {
	if unix == 0 {
		return "?"
	}
	return time.Unix(unix, 0).Format("2006-01")
}
