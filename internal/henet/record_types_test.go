package henet

import (
	"errors"
	"testing"

	"github.com/momaek/henetdns/internal/errs"
	"github.com/momaek/henetdns/internal/model"
)

func TestNormalizeRecordInputSRV(t *testing.T) {
	cases := []struct {
		name     string
		in       RecordInput
		wantVal  string
		wantPrio int
	}{
		{"three fields default priority", RecordInput{Type: "srv", Name: "_imaps._tcp", Value: "1 993 imap.migadu.com"}, "1 993 imap.migadu.com", 0},
		{"three fields explicit priority", RecordInput{Type: "SRV", Name: "_imaps._tcp", Value: "1 993 imap.migadu.com", Priority: 5, HasPriority: true}, "1 993 imap.migadu.com", 5},
		{"zone-file form", RecordInput{Type: "SRV", Name: "_imaps._tcp", Value: "0 1 993 imap.migadu.com."}, "1 993 imap.migadu.com", 0},
		{"extra whitespace", RecordInput{Type: "SRV", Name: "_imaps._tcp", Value: "  1   993  imap.migadu.com  "}, "1 993 imap.migadu.com", 0},
	}
	for _, c := range cases {
		got, err := normalizeRecordInput(c.in)
		if err != nil {
			t.Errorf("%s: unexpected error: %v", c.name, err)
			continue
		}
		if got.Value != c.wantVal || got.Priority != c.wantPrio || !got.HasPriority {
			t.Errorf("%s: got value=%q prio=%d has=%v, want value=%q prio=%d", c.name, got.Value, got.Priority, got.HasPriority, c.wantVal, c.wantPrio)
		}
	}
}

func TestNormalizeRecordInputSRVInvalid(t *testing.T) {
	cases := []RecordInput{
		{Type: "SRV", Name: "_x._tcp", Value: "imap.migadu.com"},
		{Type: "SRV", Name: "_x._tcp", Value: "1 imap.migadu.com"},
		{Type: "SRV", Name: "_x._tcp", Value: "a 993 imap.migadu.com"},
		{Type: "SRV", Name: "_x._tcp", Value: "1 70000 imap.migadu.com"},
		// four fields are ambiguous once a priority was given explicitly
		{Type: "SRV", Name: "_x._tcp", Value: "0 1 993 imap.migadu.com", Priority: 0, HasPriority: true},
	}
	for _, in := range cases {
		if _, err := normalizeRecordInput(in); !errors.Is(err, errs.ErrInvalidInput) {
			t.Errorf("value %q: expected ErrInvalidInput, got %v", in.Value, err)
		}
	}
}

func TestNormalizeRecordInputTypesAndTTL(t *testing.T) {
	for _, rrType := range []string{"caa", "NS", "alias", "PTR", "SSHFP", "HTTPS", "SVCB", "NAPTR", "LOC", "HINFO", "RP", "AFSDB", "SPF"} {
		if _, err := normalizeRecordInput(RecordInput{Type: rrType, Name: "x", Value: "v"}); err != nil {
			t.Errorf("type %s rejected: %v", rrType, err)
		}
	}
	if _, err := normalizeRecordInput(RecordInput{Type: "SOA", Name: "x", Value: "v"}); !errors.Is(err, errs.ErrInvalidInput) {
		t.Errorf("SOA should be rejected, got %v", err)
	}
	if _, err := normalizeRecordInput(RecordInput{Type: "A", Name: "x", Value: "1.2.3.4", TTL: 600}); !errors.Is(err, errs.ErrInvalidInput) {
		t.Errorf("ttl 600 should be rejected, got %v", err)
	}
	got, err := normalizeRecordInput(RecordInput{Type: "A", Name: "x", Value: "1.2.3.4"})
	if err != nil || got.TTL != 300 {
		t.Errorf("default ttl: got %d err %v, want 300", got.TTL, err)
	}
	got, err = normalizeRecordInput(RecordInput{Type: "NS", Name: "sub", Value: "ns1.example.net."})
	if err != nil || got.Value != "ns1.example.net" {
		t.Errorf("NS trailing dot: got %q err %v", got.Value, err)
	}
}

func TestBuildCreateForm(t *testing.T) {
	srv, err := normalizeRecordInput(RecordInput{Type: "SRV", Name: "_imaps._tcp.example.com", Value: "1 993 imap.migadu.com", TTL: 3600})
	if err != nil {
		t.Fatal(err)
	}
	form := buildCreateForm("42", srv)
	want := map[string]string{
		"Type": "SRV", "hosted_dns_zoneid": "42", "Name": "_imaps._tcp.example.com", "TTL": "3600",
		"Priority": "0", "Weight": "1", "Port": "993", "Target": "imap.migadu.com", "hosted_dns_editrecord": "Submit",
	}
	for k, v := range want {
		if got := form.Get(k); got != v {
			t.Errorf("SRV form %s = %q, want %q", k, got, v)
		}
	}
	if form.Has("Content") {
		t.Errorf("SRV form should not send Content, got %q", form.Get("Content"))
	}

	caa, err := normalizeRecordInput(RecordInput{Type: "CAA", Name: "example.com", Value: `0 issue "letsencrypt.org"`})
	if err != nil {
		t.Fatal(err)
	}
	form = buildCreateForm("42", caa)
	if form.Get("Content") != `0 issue "letsencrypt.org"` || form.Get("Priority") != "" || form.Has("Weight") {
		t.Errorf("unexpected CAA form: %v", form)
	}

	mx, err := normalizeRecordInput(RecordInput{Type: "MX", Name: "example.com", Value: "aspmx1.migadu.com"})
	if err != nil {
		t.Fatal(err)
	}
	if form = buildCreateForm("42", mx); form.Get("Priority") != "10" || form.Get("Content") != "aspmx1.migadu.com" {
		t.Errorf("unexpected MX form: %v", form)
	}
}

func TestFindExactRecordSRV(t *testing.T) {
	p0 := 0
	records := []model.Record{
		{RecordID: "1", Type: "SRV", Name: "_imaps._tcp.example.com", Value: "1 993 imap.migadu.com", Priority: &p0},
	}
	match := RecordInput{Type: "SRV", Name: "_imaps._tcp.example.com", Value: "1 993 IMAP.migadu.com.", Priority: 0, HasPriority: true}
	if _, found := findExactRecord(records, match); !found {
		t.Error("SRV with same fields (case/trailing dot differences) not found")
	}
	for _, in := range []RecordInput{
		{Type: "SRV", Name: "_imaps._tcp.example.com", Value: "1 993 imap.migadu.com", Priority: 10, HasPriority: true},
		{Type: "SRV", Name: "_imaps._tcp.example.com", Value: "1 995 imap.migadu.com", Priority: 0, HasPriority: true},
		{Type: "SRV", Name: "_imaps._tcp.example.com", Value: "2 993 imap.migadu.com", Priority: 0, HasPriority: true},
	} {
		if _, found := findExactRecord(records, in); found {
			t.Errorf("SRV %+v unexpectedly matched", in)
		}
	}
}

func TestFindExactRecordHostnameCase(t *testing.T) {
	records := []model.Record{{RecordID: "1", Type: "CNAME", Name: "key1._domainkey.example.com", Value: "key1.example.com._domainkey.migadu.com"}}
	in := RecordInput{Type: "CNAME", Name: "key1._domainkey.example.com", Value: "KEY1.example.com._domainkey.migadu.com."}
	if _, found := findExactRecord(records, in); !found {
		t.Error("CNAME target should match case-insensitively and ignore a trailing dot")
	}
}

func TestIsApexNS(t *testing.T) {
	cases := []struct {
		in   RecordInput
		zone string
		want bool
	}{
		{RecordInput{Type: "NS", Name: "example.com"}, "example.com", true},
		{RecordInput{Type: "NS", Name: "Example.COM."}, "example.com", true},
		{RecordInput{Type: "NS", Name: "@"}, "", true},
		{RecordInput{Type: "NS", Name: "sub.example.com"}, "example.com", false},
		{RecordInput{Type: "A", Name: "example.com"}, "example.com", false},
	}
	for _, c := range cases {
		if got := isApexNS(c.in, c.zone); got != c.want {
			t.Errorf("isApexNS(%+v, %q) = %v, want %v", c.in, c.zone, got, c.want)
		}
	}
}

// Rows modeled on a live he.net zone page (2026-10-28), trimmed to one
// SRV and one CAA record. Note the real markup has an extra empty <td>
// before the delete cell.
func TestParseRecordsSRVAndCAA(t *testing.T) {
	html := `
<div id="dns_main_content"><table>
<tr class="dns_tr" id="2000001" title="Click to edit this item." onclick="editRow(this)">
	<td class="hidden">1000001</td>
	<td class="hidden">2000001</td>
	<td width="95%" class="dns_view">example.com</td>
	<td align="center" ><span class="rrlabel CAA" data="CAA" alt="CAA" >CAA</span></td>
	<td align="left">172800</td>	<td align="center">-</td>	<td align="left" data="0 issue &quot;letsencrypt.org&quot;">0 issue &quot;letsencrypt.org&quot;</td>	<td class="hidden">0</td>      <td></td>
	<td align="center" class="dns_delete"  onclick="event.cancelBubble=true;deleteRecord('2000001','example.com','CAA')"></td>
</tr>
<tr class="dns_tr" id="2000002" title="Click to edit this item." onclick="editRow(this)">
	<td class="hidden">1000001</td>
	<td class="hidden">2000002</td>
	<td width="95%" class="dns_view">_autodiscover._tcp.example.com</td>
	<td align="center" ><span class="rrlabel SRV" data="SRV" alt="SRV" >SRV</span></td>
	<td align="left">172800</td>
	<td align="center">0</td>	<td align="left" data="1 443 autodiscover.migadu.com">1 443 autodiscover.migadu.com</td>	<td class="hidden">0</td>      <td></td>
	<td align="center" class="dns_delete"  onclick="event.cancelBubble=true;deleteRecord('2000002','_autodiscover._tcp.example.com','SRV')"></td>
</tr>
</table></div>`
	records, err := ParseRecords("1000001", []byte(html))
	if err != nil {
		t.Fatalf("ParseRecords err: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("unexpected records len: %d", len(records))
	}
	caa, srv := records[0], records[1]
	if caa.Type != "CAA" || caa.Value != `0 issue "letsencrypt.org"` || caa.Priority != nil {
		t.Errorf("unexpected CAA record: %+v", caa)
	}
	if srv.Type != "SRV" || srv.Value != "1 443 autodiscover.migadu.com" || srv.Priority == nil || *srv.Priority != 0 || srv.TTL != 172800 {
		t.Errorf("unexpected SRV record: %+v", srv)
	}

	in, err := normalizeRecordInput(RecordInput{Type: "SRV", Name: "_autodiscover._tcp.example.com", Value: "0 1 443 autodiscover.migadu.com"})
	if err != nil {
		t.Fatal(err)
	}
	if _, found := findExactRecord(records, in); !found {
		t.Error("zone-file style SRV input did not match the parsed record")
	}
}
