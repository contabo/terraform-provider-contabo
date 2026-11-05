package contabo

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestRelativeDnsRecordName(t *testing.T) {
	cases := []struct {
		name, zone, want string
	}{
		{"", "example.com", ""},
		{"@", "example.com", ""},
		{"example.com", "example.com", ""},
		{"example.com.", "example.com", ""},
		{"www", "example.com", "www"},
		{"WWW", "example.com", "www"},
		{"www.example.com", "example.com", "www"},
		{"www.example.com.", "example.com.", "www"},
		{"_sip._tcp.example.com", "example.com", "_sip._tcp"},
		{"a.b", "example.com", "a.b"},
	}
	for _, c := range cases {
		if got := relativeDnsRecordName(c.name, c.zone); got != c.want {
			t.Errorf("relativeDnsRecordName(%q, %q) = %q, want %q", c.name, c.zone, got, c.want)
		}
	}
}

func TestFqdnDnsRecordName(t *testing.T) {
	if got := fqdnDnsRecordName("", "example.com"); got != "example.com" {
		t.Errorf("apex fqdn = %q", got)
	}
	if got := fqdnDnsRecordName("www.example.com", "example.com"); got != "www.example.com" {
		t.Errorf("fqdn = %q", got)
	}
	if got := fqdnDnsRecordName("www", "Example.com."); got != "www.example.com" {
		t.Errorf("fqdn = %q", got)
	}
}

func TestDnsRecordDataEqual(t *testing.T) {
	cases := []struct {
		recordType, a, b string
		want             bool
	}{
		{"A", "192.0.2.1", "192.0.2.1", true},
		{"A", "192.0.2.1", "192.0.2.2", false},
		{"AAAA", "2001:db8::1", "2001:0db8:0000:0000:0000:0000:0000:0001", true},
		{"AAAA", "2001:db8::1", "2001:db8::2", false},
		{"CNAME", "target.example.com.", "target.example.com", true},
		{"MX", "Mail.Example.com.", "mail.example.com", true},
		{"SRV", "sip.example.com", "sip2.example.com", false},
		{"TXT", "\"v=spf1 -all\"", "v=spf1 -all", true},
		{"TXT", "Hello", "hello", false},
		{"CAA", "\"letsencrypt.org\"", "letsencrypt.org", true},
	}
	for _, c := range cases {
		if got := dnsRecordDataEqual(c.recordType, c.a, c.b); got != c.want {
			t.Errorf("dnsRecordDataEqual(%q, %q, %q) = %v, want %v", c.recordType, c.a, c.b, got, c.want)
		}
	}
}

func TestParseDnsRecordId(t *testing.T) {
	zone, id, err := parseDnsRecordId("example.com/123456789012")
	if err != nil || zone != "example.com" || id != 123456789012 {
		t.Errorf("parseDnsRecordId = %q, %d, %v", zone, id, err)
	}
	for _, invalid := range []string{"", "example.com", "example.com/", "/1", "example.com/abc"} {
		if _, _, err := parseDnsRecordId(invalid); err == nil {
			t.Errorf("parseDnsRecordId(%q) expected error", invalid)
		}
	}
	if got := dnsRecordId("Example.com.", 42); got != "example.com/42" {
		t.Errorf("dnsRecordId = %q", got)
	}
}

func TestDecodeDnsRecordsResponse(t *testing.T) {
	body := `{"data":[` +
		`{"recordId":123456789,"name":"example.com","type":"NS","prio":null,"ttl":3600,"data":"ns1.contabo.net","weight":null,"port":null,"flag":null,"tag":null},` +
		`{"recordId":123456790,"name":"_sip._tcp.example.com","type":"SRV","prio":10,"ttl":300,"data":"sip.example.com","weight":5,"port":5060,"flag":null,"tag":null}` +
		`],"_pagination":{"size":100,"totalElements":2,"totalPages":1,"page":1}}`
	res, err := decodeDnsRecordsResponse(&http.Response{Body: io.NopCloser(strings.NewReader(body))})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Data) != 2 || res.Pagination.TotalPages != 1 {
		t.Fatalf("unexpected response %+v", res)
	}
	// 123456789 is not representable as float32; the id must survive exactly.
	if res.Data[0].RecordId != 123456789 {
		t.Errorf("recordId = %d", res.Data[0].RecordId)
	}
	if res.Data[0].Prio != nil || floatToInt(res.Data[0].Prio) != 0 {
		t.Errorf("null prio decoded as %v", res.Data[0].Prio)
	}
	srv := res.Data[1]
	if floatToInt(srv.Prio) != 10 || floatToInt(srv.Weight) != 5 || floatToInt(srv.Port) != 5060 {
		t.Errorf("unexpected SRV record %+v", srv)
	}
	if got := zoneNameServers(res.Data, "example.com"); len(got) != 1 || got[0] != "ns1.contabo.net" {
		t.Errorf("zoneNameServers = %v", got)
	}
}
