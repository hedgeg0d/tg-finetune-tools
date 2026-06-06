package pipeline

import "testing"

func TestSplitPaths(t *testing.T) {
	train, val := SplitPaths("data/dataset.jsonl")
	if train != "data/dataset.train.jsonl" || val != "data/dataset.val.jsonl" {
		t.Fatalf("got %q %q", train, val)
	}
}

func TestSplitPathsNoExt(t *testing.T) {
	train, val := SplitPaths("dataset")
	if train != "dataset.train" || val != "dataset.val" {
		t.Fatalf("got %q %q", train, val)
	}
}
