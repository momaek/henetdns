package henet

import "fmt"

func ZonePagePath(zoneID string) string {
	return fmt.Sprintf("/?hosted_dns_zoneid=%s&menu=edit_zone&hosted_dns_editzone", zoneID)
}
