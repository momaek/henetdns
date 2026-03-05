package output

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/wentx/henetdns/internal/model"
)

func PrintZones(w io.Writer, zones []model.Zone, asJSON bool) error {
	if asJSON {
		return writeJSON(w, zones)
	}
	sort.Slice(zones, func(i, j int) bool { return strings.ToLower(zones[i].Name) < strings.ToLower(zones[j].Name) })
	tw := tabwriter.NewWriter(w, 2, 8, 2, ' ', 0)
	fmt.Fprintln(tw, "ZONE_ID\tZONE_NAME")
	for _, z := range zones {
		fmt.Fprintf(tw, "%s\t%s\n", z.ID, z.Name)
	}
	return tw.Flush()
}

func PrintRecords(w io.Writer, records []model.Record, asJSON bool) error {
	if asJSON {
		return writeJSON(w, records)
	}
	tw := tabwriter.NewWriter(w, 2, 8, 2, ' ', 0)
	fmt.Fprintln(tw, "RECORD_ID\tNAME\tTYPE\tTTL\tPRIORITY\tVALUE")
	for _, r := range records {
		prio := "-"
		if r.Priority != nil {
			prio = fmt.Sprintf("%d", *r.Priority)
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%d\t%s\t%s\n", r.RecordID, r.Name, r.Type, r.TTL, prio, r.Value)
	}
	return tw.Flush()
}

func PrintMessage(w io.Writer, msg string, asJSON bool) error {
	if asJSON {
		return writeJSON(w, map[string]string{"message": msg})
	}
	_, err := fmt.Fprintln(w, msg)
	return err
}

func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
