package dataset

import "strings"

func Dedup(convs []Conversation) []Conversation {
	seen := make(map[string]struct{}, len(convs))
	out := convs[:0]
	for _, c := range convs {
		key := fingerprint(c)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, c)
	}
	return out
}

func fingerprint(c Conversation) string {
	var b strings.Builder
	for _, t := range c.Turns {
		b.WriteString(t.Role)
		b.WriteByte('\x1f')
		b.WriteString(t.Content)
		b.WriteByte('\x1e')
	}
	return b.String()
}
