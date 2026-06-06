package persona

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/hedgeg0d/tg-finetune-tools/internal/config"
	"github.com/hedgeg0d/tg-finetune-tools/internal/dataset"
)

func convsOf(n int) []dataset.Conversation {
	out := make([]dataset.Conversation, n)
	for i := range out {
		out[i] = dataset.Conversation{Turns: []dataset.Turn{
			{Role: "user", Content: "q"},
			{Role: "assistant", Content: "a"},
		}}
	}
	return out
}

func TestAssignFixed(t *testing.T) {
	cfg := config.Default()
	cfg.Build.System = config.System{Mode: "fixed", Fixed: "be x"}
	got, err := Assign(convsOf(3), cfg, false)
	if err != nil || len(got) != 3 || got[0] != "be x" || got[2] != "be x" {
		t.Fatalf("got %v err %v", got, err)
	}
}

func TestAssignEmpty(t *testing.T) {
	cfg := config.Default()
	cfg.Build.System = config.System{Mode: "empty"}
	got, _ := Assign(convsOf(2), cfg, false)
	if got[0] != "" || got[1] != "" {
		t.Fatalf("got %v", got)
	}
}

func TestAssignPoolDeterministic(t *testing.T) {
	cfg := config.Default()
	cfg.Build.System = config.System{Mode: "pool", Pool: []string{"a", "b", "c"}}
	cfg.Build.Seed = 7
	a, _ := Assign(convsOf(20), cfg, false)
	b, _ := Assign(convsOf(20), cfg, false)
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("non-deterministic at %d", i)
		}
	}
}

func TestGeneratedWithMockServer(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]string{"role": "assistant", "content": "prompt"}},
			},
		})
	}))
	defer srv.Close()

	cfg := config.Default()
	cfg.Build.System = config.System{
		Mode: "generated",
		Generated: config.Generated{
			APIBase:        srv.URL,
			Model:          "test",
			BatchSize:      2,
			ContextSamples: 5,
			CacheFile:      filepath.Join(t.TempDir(), "cache.json"),
		},
	}

	got, err := Assign(convsOf(5), cfg, false)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 3 {
		t.Fatalf("expected 3 batch calls, got %d", calls)
	}
	if len(got) != 5 || got[0] != "prompt" || got[4] != "prompt" {
		t.Fatalf("got %v", got)
	}

	calls = 0
	if _, err := Assign(convsOf(5), cfg, false); err != nil {
		t.Fatal(err)
	}
	if calls != 0 {
		t.Fatalf("expected cache hit, got %d calls", calls)
	}
}
