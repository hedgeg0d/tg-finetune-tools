package tokenize

import (
	"github.com/pkoukk/tiktoken-go"
	loader "github.com/pkoukk/tiktoken-go-loader"
)

type Counter struct {
	enc *tiktoken.Tiktoken
}

func New(encoding string) (*Counter, error) {
	tiktoken.SetBpeLoader(loader.NewOfflineLoader())
	enc, err := tiktoken.GetEncoding(encoding)
	if err != nil {
		return nil, err
	}
	return &Counter{enc: enc}, nil
}

func (c *Counter) Count(s string) int {
	return len(c.enc.Encode(s, nil, nil))
}
