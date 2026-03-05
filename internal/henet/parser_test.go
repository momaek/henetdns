package henet

import "testing"

func TestParseZones(t *testing.T) {
	html := `
<table id="domains_table"><tbody>
<tr>
  <td><img alt="edit" onclick="javascript:document.location.href='?hosted_dns_zoneid=123&menu=edit_zone&hosted_dns_editzone'"/></td>
  <td><span>example.com</span></td>
</tr>
</tbody></table>`
	zones, err := ParseZones([]byte(html))
	if err != nil {
		t.Fatalf("ParseZones err: %v", err)
	}
	if len(zones) != 1 || zones[0].ID != "123" || zones[0].Name != "example.com" {
		t.Fatalf("unexpected zones: %+v", zones)
	}
}

func TestParseRecords(t *testing.T) {
	html := `
<div id="dns_main_content"><table>
<tr class="dns_tr" id="111">
  <td class="hidden">123</td>
  <td class="hidden">111</td>
  <td>www.example.com</td>
  <td><span class="rrlabel A" data="A">A</span></td>
  <td>300</td>
  <td>-</td>
  <td data="1.2.3.4">1.2.3.4</td>
  <td class="hidden">0</td>
  <td class="dns_delete" onclick="deleteRecord('111','www.example.com','A')"></td>
</tr>
</table></div>`
	records, err := ParseRecords("123", []byte(html))
	if err != nil {
		t.Fatalf("ParseRecords err: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("unexpected records len: %d", len(records))
	}
	r := records[0]
	if r.RecordID != "111" || r.Type != "A" || r.TTL != 300 || r.Value != "1.2.3.4" {
		t.Fatalf("unexpected record: %+v", r)
	}
}
