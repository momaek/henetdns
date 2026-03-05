package henet

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/wentx/henetdns/internal/errs"
	"github.com/wentx/henetdns/internal/model"
)

var zoneIDRe = regexp.MustCompile(`hosted_dns_zoneid=(\d+)`)

func ParseZones(body []byte) ([]model.Zone, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("parse zones document: %w: %w", err, errs.ErrParseChanged)
	}

	zonesByID := map[string]model.Zone{}
	doc.Find("#domains_table tbody tr").Each(func(_ int, row *goquery.Selection) {
		editImg := row.Find("img[alt='edit']")
		onclick, _ := editImg.Attr("onclick")
		match := zoneIDRe.FindStringSubmatch(onclick)
		if len(match) != 2 {
			return
		}
		zoneID := strings.TrimSpace(match[1])
		name := strings.TrimSpace(row.Find("td span").First().Text())
		if zoneID == "" || name == "" {
			return
		}
		zonesByID[zoneID] = model.Zone{ID: zoneID, Name: name}
	})

	zones := make([]model.Zone, 0, len(zonesByID))
	for _, z := range zonesByID {
		zones = append(zones, z)
	}
	if len(zones) == 0 {
		return nil, fmt.Errorf("no zones found in page: %w", errs.ErrParseChanged)
	}
	return zones, nil
}
