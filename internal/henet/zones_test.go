package henet

import (
	"errors"
	"testing"

	"github.com/momaek/henetdns/internal/errs"
	"github.com/momaek/henetdns/internal/model"
)

func TestNormalizeZoneName(t *testing.T) {
	cases := []struct{ in, want string }{
		{"example.com", "example.com"},
		{" Example.COM. ", "example.com"},
		{"example.ie", "example.ie"},
		{"sub.example.co.uk", "sub.example.co.uk"},
		{"123example.net", "123example.net"},
		{"xn--bcher-kva.example", "xn--bcher-kva.example"},
	}
	for _, c := range cases {
		got, err := NormalizeZoneName(c.in)
		if err != nil || got != c.want {
			t.Errorf("NormalizeZoneName(%q) = %q, %v; want %q", c.in, got, err, c.want)
		}
	}
	for _, bad := range []string{"", "com", "exa mple.com", "-bad.com", "bad-.com", "example..com", "http://example.com", "example.com/x", "10.0.0.0/24"} {
		if _, err := NormalizeZoneName(bad); !errors.Is(err, errs.ErrInvalidInput) {
			t.Errorf("NormalizeZoneName(%q) should fail with ErrInvalidInput, got %v", bad, err)
		}
	}
}

func TestParseErrorMessage(t *testing.T) {
	cases := []struct{ name, html, want string }{
		{"none", `<html><body><div id="dns_main_content"></div></body></html>`, ""},
		{"error", `<div id="dns_err">This zone already exists in our system.</div>`, "This zone already exists in our system."},
		{"whitespace collapsed", "<div id=\"dns_err\">\n  Something\n   went wrong \n</div>", "Something went wrong"},
		{"improper delegation ignored", `<div id="dns_err">This zone does not appear to be properly delegated to our nameservers.</div>`, ""},
		{"empty banner", `<div id="dns_err"> </div>`, ""},
	}
	for _, c := range cases {
		if got := ParseErrorMessage([]byte(c.html)); got != c.want {
			t.Errorf("%s: ParseErrorMessage = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestFindZone(t *testing.T) {
	zones := []model.Zone{{ID: "1", Name: "example.com"}, {ID: "2", Name: "Other.net"}}
	if z, ok := findZone(zones, "other.net"); !ok || z.ID != "2" {
		t.Errorf("findZone case-insensitive lookup failed: %+v %v", z, ok)
	}
	if _, ok := findZone(zones, "missing.org"); ok {
		t.Error("findZone matched a missing zone")
	}
}
