package henet

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/wentx/henetdns/internal/errs"
	"github.com/wentx/henetdns/internal/model"
)

var deleteRecordRE = regexp.MustCompile(`deleteRecord\('([^']+)'`)

func ParseRecords(zoneID string, body []byte) ([]model.Record, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("parse records document: %w: %w", err, errs.ErrParseChanged)
	}
	var out []model.Record
	doc.Find("#dns_main_content table tr").Each(func(_ int, row *goquery.Selection) {
		tds := row.Find("td")
		if tds.Length() < 8 {
			return
		}

		recordID := strings.TrimSpace(tds.Eq(1).Text())
		name := strings.TrimSpace(tds.Eq(2).Text())
		typeCell := tds.Eq(3)
		typeLabel := strings.TrimSpace(typeCell.Find("span.rrlabel").AttrOr("data", typeCell.Text()))
		ttlRaw := strings.TrimSpace(tds.Eq(4).Text())
		prioRaw := strings.TrimSpace(tds.Eq(5).Text())
		dataCell := tds.Eq(6)
		value := strings.TrimSpace(dataCell.AttrOr("data", dataCell.Text()))
		dynRaw := strings.TrimSpace(tds.Eq(7).Text())

		if recordID == "" || name == "" || typeLabel == "" {
			return
		}

		ttl, _ := strconv.Atoi(ttlRaw)
		var prio *int
		if prioRaw != "" && prioRaw != "-" {
			if p, err := strconv.Atoi(prioRaw); err == nil {
				prio = &p
			}
		}
		dynamic := dynRaw == "1"

		locked := row.HasClass("dns_tr_locked")
		if delCell := tds.Eq(8); delCell.Length() > 0 {
			onclick, _ := delCell.Attr("onclick")
			if m := deleteRecordRE.FindStringSubmatch(onclick); len(m) == 2 {
				recordID = strings.TrimSpace(m[1])
			}
		}

		typeLabel = strings.ToUpper(strings.TrimSpace(typeLabel))
		uid := buildRecordUID(recordID, name, typeLabel, value, prio)
		out = append(out, model.Record{
			ZoneID:    zoneID,
			RecordID:  recordID,
			RecordUID: uid,
			Name:      name,
			Type:      typeLabel,
			TTL:       ttl,
			Priority:  prio,
			Value:     value,
			Dynamic:   dynamic,
			Locked:    locked,
		})
	})

	if len(out) == 0 {
		if strings.Contains(string(body), "Free DNS Login") {
			return nil, fmt.Errorf("not authenticated while parsing records: %w", errs.ErrAuthRequired)
		}
		return nil, fmt.Errorf("no records parsed: %w", errs.ErrParseChanged)
	}
	return out, nil
}

func buildRecordUID(recordID, name, rrType, value string, priority *int) string {
	prio := ""
	if priority != nil {
		prio = strconv.Itoa(*priority)
	}
	return strings.Join([]string{recordID, name, rrType, value, prio}, "|")
}
