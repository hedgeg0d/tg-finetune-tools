package persona

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
)

func loadCache(path string) ([]string, error) {
	if path == "" {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var prompts []string
	if err := json.Unmarshal(data, &prompts); err != nil {
		return nil, err
	}
	return prompts, nil
}

func saveCache(path string, prompts []string) error {
	if path == "" {
		return nil
	}
	data, err := json.MarshalIndent(prompts, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
