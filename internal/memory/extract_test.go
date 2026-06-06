package memory

import "testing"

func TestParseListStrictJSON(t *testing.T) {
	got := parseList(`["любит кошек","учится на юриста"]`)
	if len(got) != 2 || got[0] != "любит кошек" {
		t.Fatalf("got %v", got)
	}
}

func TestParseListSmartQuotesAndFences(t *testing.T) {
	got := parseList("```json\n[«любит кошек», «учится на юриста»]\n```")
	if len(got) != 2 || got[1] != "учится на юриста" {
		t.Fatalf("got %v", got)
	}
}

func TestParseListDropsJunk(t *testing.T) {
	got := parseList("[\n  \"\",\n  \"]\",\n  \": ок\",\n  \"имеет кошку по кличке бро\"\n]")
	if len(got) != 1 || got[0] != "имеет кошку по кличке бро" {
		t.Fatalf("got %v", got)
	}
}

func TestParseListDedup(t *testing.T) {
	got := parseList(`["любит кошек","Любит кошек"]`)
	if len(got) != 1 {
		t.Fatalf("got %v", got)
	}
}

func TestAddDedupsAcrossWindows(t *testing.T) {
	seen := map[string]bool{}
	var facts []fact
	add(&facts, seen, fact{Text: "имеет кошку"})
	add(&facts, seen, fact{Text: "Имеет кошку "})
	add(&facts, seen, fact{Text: "играет в доту"})
	add(&facts, seen, fact{Text: ""})
	if len(facts) != 2 {
		t.Fatalf("got %d: %v", len(facts), facts)
	}
}
