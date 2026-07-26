package henet

import (
	"testing"

	"github.com/momaek/henetdns/internal/model"
)

func TestQualifyRecordName(t *testing.T) {
	cases := []struct {
		name string
		zone string
		want string
	}{
		{"www", "example.com", "www.example.com"},
		{"www.example.com", "example.com", "www.example.com"},
		{"www.example.com.", "example.com", "www.example.com"},
		{"WWW.Example.COM", "example.com", "WWW.Example.COM"},
		{"@", "example.com", "example.com"},
		{"", "example.com", "example.com"},
		{"example.com", "example.com", "example.com"},
		{"_acme-challenge.tolato", "ckneedu.com", "_acme-challenge.tolato.ckneedu.com"},
		// zone unknown: leave as-is
		{"www", "", "www"},
		// name that merely ends with the zone string but not on a label
		// boundary is still qualified
		{"badexample.com", "example.com", "badexample.com.example.com"},
	}
	for _, c := range cases {
		if got := qualifyRecordName(c.name, c.zone); got != c.want {
			t.Errorf("qualifyRecordName(%q, %q) = %q, want %q", c.name, c.zone, got, c.want)
		}
	}
}

func TestNormalizeTXTValue(t *testing.T) {
	cases := []struct{ in, want string }{
		{`"token"`, "token"},
		{"token", "token"},
		{`  "token"  `, "token"},
		{`"`, `"`},
		{`""`, ""},
		{`"a"b"`, `a"b`},
	}
	for _, c := range cases {
		if got := normalizeTXTValue(c.in); got != c.want {
			t.Errorf("normalizeTXTValue(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestFindExactRecordTXTQuotes(t *testing.T) {
	records := []model.Record{
		{RecordID: "1", Type: "TXT", Name: "_acme-challenge.tolato.example.com", Value: `"token123"`},
	}
	for _, value := range []string{"token123", `"token123"`} {
		in := RecordInput{Type: "TXT", Name: "_acme-challenge.tolato.example.com", Value: value}
		if _, found := findExactRecord(records, in); !found {
			t.Errorf("TXT value %q did not match cached quoted value", value)
		}
	}
	in := RecordInput{Type: "TXT", Name: "_acme-challenge.tolato.example.com", Value: "other"}
	if _, found := findExactRecord(records, in); found {
		t.Errorf("TXT value %q unexpectedly matched", "other")
	}
}

func TestFindExactRecordMXPriority(t *testing.T) {
	p := 10
	records := []model.Record{
		{RecordID: "1", Type: "MX", Name: "example.com", Value: "mail.example.com", Priority: &p},
	}
	in := RecordInput{Type: "MX", Name: "example.com", Value: "mail.example.com", Priority: 10, HasPriority: true}
	if _, found := findExactRecord(records, in); !found {
		t.Fatal("MX record with matching priority not found")
	}
	in.Priority = 20
	if _, found := findExactRecord(records, in); found {
		t.Fatal("MX record matched despite different priority")
	}
}

func TestNearMissHints(t *testing.T) {
	records := []model.Record{
		{RecordID: "1", Type: "TXT", Name: "www.example.com", Value: `"aaa"`},
		{RecordID: "2", Type: "A", Name: "www.example.com", Value: "1.2.3.4"},
		{RecordID: "3", Type: "A", Name: "other.example.com", Value: "5.6.7.8"},
	}
	in := RecordInput{Type: "TXT", Name: "www.example.com", Value: "nope"}
	hints := nearMissHints(records, in)
	if hints == "" {
		t.Fatal("expected near-miss hints for same-name records")
	}
	if want := `[TXT www.example.com "aaa"] [A www.example.com 1.2.3.4]`; hints != want {
		t.Errorf("hints = %q, want %q", hints, want)
	}

	in.Name = "missing.example.com"
	if hints := nearMissHints(records, in); hints != "" {
		t.Errorf("expected no hints, got %q", hints)
	}
}
