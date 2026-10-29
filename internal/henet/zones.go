package henet

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/momaek/henetdns/internal/errs"
	"github.com/momaek/henetdns/internal/model"
)

// improperDelegationMsg is the warning he.net shows when a zone is added
// before the registrar points its nameservers at ns1-5.he.net. The zone is
// still created, so it is not treated as an error.
const improperDelegationMsg = "This zone does not appear to be properly delegated to our nameservers."

var zoneNameRE = regexp.MustCompile(`^([a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)

// NormalizeZoneName lowercases a domain, strips a trailing dot, and checks it
// looks like a hostname with at least two labels.
func NormalizeZoneName(name string) (string, error) {
	name = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(name), "."))
	if !zoneNameRE.MatchString(name) {
		return "", fmt.Errorf("invalid zone name %q: %w", name, errs.ErrInvalidInput)
	}
	return name, nil
}

// AddZone creates a primary zone, mirroring the "Add a new domain" form on the
// he.net dashboard. It is idempotent: if the zone already exists it is
// returned with created=false.
func (s *Service) AddZone(ctx context.Context, name string) (model.Zone, bool, error) {
	name, err := NormalizeZoneName(name)
	if err != nil {
		return model.Zone{}, false, err
	}
	zones, err := s.ListZones(ctx)
	if err != nil {
		return model.Zone{}, false, err
	}
	if z, ok := findZone(zones, name); ok {
		return z, false, nil
	}

	form := url.Values{}
	form.Set("action", "add_zone")
	form.Set("retmain", "0")
	form.Set("add_domain", name)
	form.Set("submit", "Add Domain!")
	resp, err := s.client.PostForm(ctx, "/index.cgi", form, s.client.BaseURL().String())
	if err != nil {
		return model.Zone{}, false, err
	}
	if msg := ParseErrorMessage(resp.Body); msg != "" {
		return model.Zone{}, false, fmt.Errorf("he.net rejected zone %s: %s: %w", name, msg, errs.ErrRemote)
	}

	zones, err = s.ListZones(ctx)
	if err != nil {
		return model.Zone{}, false, err
	}
	z, ok := findZone(zones, name)
	if !ok {
		return model.Zone{}, false, fmt.Errorf("zone %s not found after add: %w", name, errs.ErrRemote)
	}
	return z, true, nil
}

func findZone(zones []model.Zone, name string) (model.Zone, bool) {
	for _, z := range zones {
		if strings.EqualFold(z.Name, name) {
			return z, true
		}
	}
	return model.Zone{}, false
}

// ParseErrorMessage returns the text of he.net's div#dns_err banner, ignoring
// the improper-delegation warning. Empty means no error was shown.
func ParseErrorMessage(body []byte) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return ""
	}
	var msgs []string
	doc.Find("div#dns_err").Each(func(_ int, sel *goquery.Selection) {
		msg := strings.Join(strings.Fields(sel.Text()), " ")
		if msg == "" || strings.Contains(msg, improperDelegationMsg) {
			return
		}
		msgs = append(msgs, msg)
	})
	return strings.Join(msgs, "; ")
}
