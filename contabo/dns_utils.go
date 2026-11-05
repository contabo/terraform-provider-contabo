package contabo

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"

	"contabo.com/openapi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	uuid "github.com/satori/go.uuid"
)

const dnsRecordsPageSize = 500

var dnsRecordTypes = []string{"A", "AAAA", "CAA", "CNAME", "MX", "SRV", "TXT"}

// dnsRecord mirrors DnsZoneRecordResponse. It is decoded from the raw
// response body because the generated model stores recordId as float32,
// which cannot represent record ids above 2^24 exactly, and because most
// numeric attributes are returned as null for record types they do not
// apply to.
type dnsRecord struct {
	RecordId int64    `json:"recordId"`
	Name     string   `json:"name"`
	Type     string   `json:"type"`
	Ttl      *float64 `json:"ttl"`
	Prio     *float64 `json:"prio"`
	Data     string   `json:"data"`
	Port     *float64 `json:"port"`
	Weight   *float64 `json:"weight"`
	Flag     *float64 `json:"flag"`
	Tag      *string  `json:"tag"`
}

type dnsRecordsResponse struct {
	Data       []dnsRecord `json:"data"`
	Pagination struct {
		TotalPages int64 `json:"totalPages"`
	} `json:"_pagination"`
}

func decodeDnsRecordsResponse(httpResp *http.Response) (dnsRecordsResponse, error) {
	var res dnsRecordsResponse
	if httpResp == nil || httpResp.Body == nil {
		return res, fmt.Errorf("empty response from DNS API")
	}
	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return res, err
	}
	err = json.Unmarshal(body, &res)
	return res, err
}

// listDnsZoneRecords returns every record of a zone, following pagination.
func listDnsZoneRecords(
	ctx context.Context,
	client *openapi.APIClient,
	zoneName string,
) ([]dnsRecord, *http.Response, error) {
	var records []dnsRecord
	for page := int64(1); ; page++ {
		_, httpResp, err := client.DNSApi.
			RetrieveDnsZoneRecordsList(ctx, zoneName).
			XRequestId(uuid.NewV4().String()).
			Page(page).
			Size(dnsRecordsPageSize).
			Execute()
		if err != nil {
			return nil, httpResp, err
		}

		res, err := decodeDnsRecordsResponse(httpResp)
		if err != nil {
			return nil, nil, err
		}
		records = append(records, res.Data...)

		if page >= res.Pagination.TotalPages || len(res.Data) == 0 {
			return records, httpResp, nil
		}
	}
}

func isNotFound(httpResp *http.Response) bool {
	return httpResp != nil && httpResp.StatusCode == http.StatusNotFound
}

func floatToInt(v *float64) int {
	if v == nil {
		return 0
	}
	return int(*v)
}

func stringValue(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func normalizeDnsName(name string) string {
	return strings.ToLower(strings.TrimSuffix(strings.TrimSpace(name), "."))
}

// relativeDnsRecordName converts a record name to its form relative to the
// zone. The apex is represented by an empty string. Both "@" and fully
// qualified names (with or without trailing dot) are accepted.
func relativeDnsRecordName(name string, zoneName string) string {
	n := normalizeDnsName(name)
	z := normalizeDnsName(zoneName)
	if n == "" || n == "@" || n == z {
		return ""
	}
	return strings.TrimSuffix(n, "."+z)
}

func fqdnDnsRecordName(name string, zoneName string) string {
	relative := relativeDnsRecordName(name, zoneName)
	z := normalizeDnsName(zoneName)
	if relative == "" {
		return z
	}
	return relative + "." + z
}

// dnsRecordDataEqual reports whether two record values are equivalent for
// the given record type, accounting for the normalisation the API applies
// (trailing dots on host names are removed, TXT quotes are stripped, IPv6
// addresses may be expanded or compressed).
func dnsRecordDataEqual(recordType string, a string, b string) bool {
	if a == b {
		return true
	}
	switch strings.ToUpper(recordType) {
	case "A", "AAAA":
		ipA, ipB := net.ParseIP(a), net.ParseIP(b)
		return ipA != nil && ipB != nil && ipA.Equal(ipB)
	case "CNAME", "MX", "SRV":
		return normalizeDnsName(a) == normalizeDnsName(b)
	case "TXT", "CAA":
		return trimQuotes(a) == trimQuotes(b)
	}
	return false
}

func trimQuotes(s string) string {
	if len(s) >= 2 && strings.HasPrefix(s, "\"") && strings.HasSuffix(s, "\"") {
		return s[1 : len(s)-1]
	}
	return s
}

func suppressEquivalentDnsName(k, old, new string, d *schema.ResourceData) bool {
	return normalizeDnsName(old) == normalizeDnsName(new)
}

func suppressEquivalentIp(k, old, new string, d *schema.ResourceData) bool {
	ipOld, ipNew := net.ParseIP(old), net.ParseIP(new)
	return ipOld != nil && ipNew != nil && ipOld.Equal(ipNew)
}

func dnsRecordId(zoneName string, recordId int64) string {
	return normalizeDnsName(zoneName) + "/" + strconv.FormatInt(recordId, 10)
}

func parseDnsRecordId(id string) (string, int64, error) {
	idx := strings.LastIndex(id, "/")
	if idx <= 0 || idx == len(id)-1 {
		return "", 0, fmt.Errorf("invalid DNS record id %q, expected <zone_name>/<record_id>", id)
	}
	recordId, err := strconv.ParseInt(id[idx+1:], 10, 64)
	if err != nil {
		return "", 0, fmt.Errorf("invalid DNS record id %q, expected <zone_name>/<record_id>: %w", id, err)
	}
	return id[:idx], recordId, nil
}

// zoneNameServers returns the targets of the NS records at the zone apex.
func zoneNameServers(records []dnsRecord, zoneName string) []string {
	nameServers := []string{}
	for _, record := range records {
		if record.Type == "NS" && relativeDnsRecordName(record.Name, zoneName) == "" {
			nameServers = append(nameServers, normalizeDnsName(record.Data))
		}
	}
	return nameServers
}
