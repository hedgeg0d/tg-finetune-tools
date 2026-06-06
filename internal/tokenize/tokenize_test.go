package tokenize

import "testing"

func TestCounter(t *testing.T) {
	c, err := New("cl100k_base")
	if err != nil {
		t.Fatal(err)
	}
	if c.Count("") != 0 {
		t.Fatal("empty should be 0")
	}
	if c.Count("hello world") <= 0 {
		t.Fatal("expected positive count")
	}
}

func TestUnknownEncoding(t *testing.T) {
	if _, err := New("does-not-exist"); err == nil {
		t.Fatal("expected error")
	}
}
