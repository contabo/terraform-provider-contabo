package contabo

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// fakeAPI is an in-memory stand-in for the Contabo API and its token
// endpoint. It implements the endpoints the provider uses, with the quirks
// of the real API that the provider has to handle (for example the firewall
// "instances" attribute as a JSON-encoded string).
type fakeAPI struct {
	t      *testing.T
	server *httptest.Server
	mux    *http.ServeMux

	mu             sync.Mutex
	userId         string
	nextId         int64
	instances      map[int64]map[string]interface{}
	instancePolls  map[int64]int
	notCancelable  map[int64]bool
	reinstalls     []map[string]interface{}
	upgrades       []map[string]interface{}
	creates        []map[string]interface{}
	firewalls      map[string]map[string]interface{}
	vips           map[string]map[string]interface{}
	roles          map[int64]map[string]interface{}
	users          map[string]map[string]interface{}
	credentials    []map[string]interface{}
	imageStats     map[string]interface{}
	vnc            map[int64]map[string]interface{}
	products       []map[string]interface{}
	tagAssignments map[int64][]map[string]interface{}
	calls          []string
}

func newFakeAPI(t *testing.T) *fakeAPI {
	t.Helper()
	t.Setenv("CNTB_TOKEN_CACHE_DIR", t.TempDir())
	previousInterval := pollInterval
	pollInterval = 10 * time.Millisecond
	t.Cleanup(func() { pollInterval = previousInterval })

	f := &fakeAPI{
		t:              t,
		mux:            http.NewServeMux(),
		userId:         "a2b3c4d5-0000-4000-8000-000000000001",
		nextId:         100,
		instances:      map[int64]map[string]interface{}{},
		instancePolls:  map[int64]int{},
		notCancelable:  map[int64]bool{},
		firewalls:      map[string]map[string]interface{}{},
		vips:           map[string]map[string]interface{}{},
		roles:          map[int64]map[string]interface{}{},
		users:          map[string]map[string]interface{}{},
		vnc:            map[int64]map[string]interface{}{},
		tagAssignments: map[int64][]map[string]interface{}{},
		imageStats: map[string]interface{}{
			"tenantId": "INT", "customerId": "12345", "currentImagesCount": 1,
			"totalSizeMb": 25600, "usedSizeMb": 1024, "freeSizeMb": 24576,
		},
	}
	f.routes()
	f.server = httptest.NewServer(f)
	t.Cleanup(f.server.Close)
	return f
}

func (f *fakeAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	f.calls = append(f.calls, r.Method+" "+r.URL.Path)
	f.mu.Unlock()
	if r.URL.Path != "/token" {
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
			writeAPIError(w, http.StatusUnauthorized, "missing bearer token")
			return
		}
		if r.Header.Get("x-request-id") == "" {
			writeAPIError(w, http.StatusBadRequest, "missing x-request-id")
			return
		}
	}
	f.mux.ServeHTTP(w, r)
}

// providerConfig returns a provider block that points at the fake.
func (f *fakeAPI) providerConfig() string {
	return fmt.Sprintf(`
provider "contabo" {
  api                  = %q
  oauth2_token_url     = %q
  oauth2_client_id     = "client"
  oauth2_client_secret = "secret"
  oauth2_user          = "user@example.com"
  oauth2_pass          = "password"
}
`, f.server.URL, f.server.URL+"/token")
}

func testProviderFactories() map[string]func() (*schema.Provider, error) {
	return map[string]func() (*schema.Provider, error){
		// Acceptance test checks read testAccProvider.Meta(), so every test
		// uses the shared provider instance.
		"contabo": func() (*schema.Provider, error) { return testAccProvider, nil },
	}
}

func (f *fakeAPI) callCount(prefix string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, call := range f.calls {
		if strings.HasPrefix(call, prefix) {
			n++
		}
	}
	return n
}

func (f *fakeAPI) id() int64 {
	f.nextId++
	return f.nextId
}

func fakeJWT(claims map[string]interface{}) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payload, _ := json.Marshal(claims)
	return header + "." + base64.RawURLEncoding.EncodeToString(payload) + ".signature"
}

func writeJSON(w http.ResponseWriter, status int, value interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeData(w http.ResponseWriter, status int, items ...map[string]interface{}) {
	if items == nil {
		items = []map[string]interface{}{}
	}
	writeJSON(w, status, map[string]interface{}{"data": items, "_links": map[string]string{"self": ""}})
}

func writePage(w http.ResponseWriter, r *http.Request, items []map[string]interface{}) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 10
	}
	totalPages := (len(items) + size - 1) / size
	start := (page - 1) * size
	end := start + size
	if start > len(items) {
		start = len(items)
	}
	if end > len(items) {
		end = len(items)
	}
	data := items[start:end]
	if data == nil {
		data = []map[string]interface{}{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": data,
		"_pagination": map[string]interface{}{
			"size": size, "totalElements": len(items), "totalPages": totalPages, "page": page,
		},
		"_links": map[string]string{"self": ""},
	})
}

func writeAPIError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]interface{}{"statusCode": status, "message": message})
}

func readBody(r *http.Request) map[string]interface{} {
	body := map[string]interface{}{}
	data, _ := io.ReadAll(r.Body)
	if len(data) > 0 {
		_ = json.Unmarshal(data, &body)
	}
	return body
}

func copyMap(m map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func pathInt(r *http.Request, name string) int64 {
	id, _ := strconv.ParseInt(r.PathValue(name), 10, 64)
	return id
}

// addInstance stores an instance as the API would return it.
func (f *fakeAPI) addInstance(overrides map[string]interface{}) int64 {
	f.mu.Lock()
	defer f.mu.Unlock()
	id := f.id() + 200000000
	instance := map[string]interface{}{
		"tenantId": "INT", "customerId": "12345", "additionalIps": []interface{}{},
		"name": fmt.Sprintf("vmi%d", id), "displayName": "", "instanceId": id,
		"dataCenter": "European Union 2", "region": "EU", "regionName": "European Union",
		"productId": "V153", "imageId": "afecbb85-e2fc-46f0-9684-b46b1faf00bb",
		"ipConfig": map[string]interface{}{
			"v4": map[string]interface{}{"ip": "203.0.113.10", "gateway": "203.0.113.1", "netmaskCidr": 24},
			"v6": map[string]interface{}{"ip": "2001:db8::1", "gateway": "fe80::1", "netmaskCidr": 64},
		},
		"macAddress": "00:50:56:00:00:01", "ramMb": 8192, "cpuCores": 4, "osType": "Linux",
		"diskMb": 102400, "sshKeys": []interface{}{}, "createdDate": time.Now().UTC().Format(time.RFC3339),
		"cancelDate": nil, "status": "running", "vHostId": 1, "vHostNumber": 1, "vHostName": "host",
		"addOns":       []interface{}{map[string]interface{}{"id": 1501, "quantity": 1}},
		"errorMessage": nil, "productType": "ssd", "productName": "Cloud VPS 4",
		"defaultUser": "admin", "applicationId": nil,
	}
	for k, v := range overrides {
		instance[k] = v
	}
	f.instances[id] = instance
	return id
}

func (f *fakeAPI) instance(id int64) map[string]interface{} {
	f.mu.Lock()
	defer f.mu.Unlock()
	return copyMap(f.instances[id])
}

func (f *fakeAPI) routes() {
	m := f.mux

	m.HandleFunc("POST /token", func(w http.ResponseWriter, r *http.Request) {
		now := time.Now().Unix()
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"access_token":  fakeJWT(map[string]interface{}{"sub": f.userId, "exp": now + 300}),
			"refresh_token": fakeJWT(map[string]interface{}{"exp": now + 1800}),
			"expires_in":    300,
			"token_type":    "Bearer",
		})
	})

	// Instances.
	m.HandleFunc("GET /v1/compute/instances", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		var items []map[string]interface{}
		ids := make([]int64, 0, len(f.instances))
		for id := range f.instances {
			ids = append(ids, id)
		}
		sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
		q := r.URL.Query()
		for _, id := range ids {
			instance := f.instances[id]
			if v := q.Get("displayName"); v != "" && !strings.Contains(instance["displayName"].(string), v) {
				continue
			}
			if v := q.Get("name"); v != "" && instance["name"] != v {
				continue
			}
			items = append(items, copyMap(instance))
		}
		f.mu.Unlock()
		writePage(w, r, items)
	})
	m.HandleFunc("POST /v1/compute/instances", func(w http.ResponseWriter, r *http.Request) {
		body := readBody(r)
		f.mu.Lock()
		f.creates = append(f.creates, body)
		f.mu.Unlock()
		overrides := map[string]interface{}{"status": "provisioning"}
		for _, key := range []string{"displayName", "imageId", "region", "productId", "defaultUser"} {
			if v, ok := body[key]; ok {
				overrides[key] = v
			}
		}
		if v, ok := body["sshKeys"]; ok {
			overrides["sshKeys"] = v
		}
		if addOns, ok := body["addOns"].(map[string]interface{}); ok {
			if ids, ok := addOns["addonsIds"].([]interface{}); ok {
				overrides["addOns"] = ids
			}
		}
		id := f.addInstance(overrides)
		writeData(w, http.StatusCreated, map[string]interface{}{"instanceId": id, "status": "provisioning"})
	})
	m.HandleFunc("GET /v1/compute/instances/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := pathInt(r, "id")
		f.mu.Lock()
		instance, ok := f.instances[id]
		if ok {
			// Instances settle after two polls.
			if status := instance["status"]; status == "provisioning" || status == "installing" {
				f.instancePolls[id]++
				if f.instancePolls[id] >= 2 {
					instance["status"] = "running"
					f.instancePolls[id] = 0
				}
			}
			instance = copyMap(instance)
		}
		f.mu.Unlock()
		if !ok {
			writeAPIError(w, http.StatusNotFound, "Entry Instance not found by instanceId "+r.PathValue("id"))
			return
		}
		writeData(w, http.StatusOK, instance)
	})
	m.HandleFunc("PATCH /v1/compute/instances/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := pathInt(r, "id")
		body := readBody(r)
		f.mu.Lock()
		instance, ok := f.instances[id]
		if ok {
			if v, ok := body["displayName"]; ok {
				instance["displayName"] = v
			}
		}
		f.mu.Unlock()
		if !ok {
			writeAPIError(w, http.StatusNotFound, "instance not found")
			return
		}
		writeData(w, http.StatusOK, map[string]interface{}{"instanceId": id})
	})
	m.HandleFunc("PUT /v1/compute/instances/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := pathInt(r, "id")
		body := readBody(r)
		f.mu.Lock()
		instance, ok := f.instances[id]
		if ok {
			f.reinstalls = append(f.reinstalls, body)
			instance["status"] = "installing"
			instance["imageId"] = body["imageId"]
			if v, ok := body["sshKeys"]; ok {
				instance["sshKeys"] = v
			} else {
				instance["sshKeys"] = []interface{}{}
			}
			if v, ok := body["defaultUser"]; ok {
				instance["defaultUser"] = v
			} else {
				// Like the real API: a reinstall without defaultUser resets it.
				instance["defaultUser"] = "admin"
			}
		}
		f.mu.Unlock()
		if !ok {
			writeAPIError(w, http.StatusNotFound, "instance not found")
			return
		}
		writeData(w, http.StatusOK, map[string]interface{}{"instanceId": id})
	})
	m.HandleFunc("POST /v1/compute/instances/{id}/cancel", func(w http.ResponseWriter, r *http.Request) {
		id := pathInt(r, "id")
		f.mu.Lock()
		instance, ok := f.instances[id]
		notCancelable := f.notCancelable[id]
		if ok && !notCancelable {
			instance["cancelDate"] = time.Now().AddDate(0, 1, 0).Format("2006-01-02")
		}
		f.mu.Unlock()
		switch {
		case !ok:
			writeAPIError(w, http.StatusNotFound, "instance not found")
		case notCancelable:
			writeAPIError(w, http.StatusUnprocessableEntity, "instance cannot be cancelled")
		default:
			writeData(w, http.StatusOK, map[string]interface{}{"instanceId": id})
		}
	})
	m.HandleFunc("POST /v1/compute/instances/{id}/upgrade", func(w http.ResponseWriter, r *http.Request) {
		id := pathInt(r, "id")
		body := readBody(r)
		f.mu.Lock()
		instance, ok := f.instances[id]
		if ok {
			f.upgrades = append(f.upgrades, body)
			addOns, _ := instance["addOns"].([]interface{})
			if _, ok := body["privateNetworking"]; ok {
				addOns = append(addOns, map[string]interface{}{"id": 1477, "quantity": 1})
			}
			if _, ok := body["backup"]; ok {
				addOns = append(addOns, map[string]interface{}{"id": 2214, "quantity": 1})
			}
			instance["addOns"] = addOns
		}
		f.mu.Unlock()
		if !ok {
			writeAPIError(w, http.StatusNotFound, "instance not found")
			return
		}
		writeData(w, http.StatusOK, map[string]interface{}{"instanceId": id})
	})
	m.HandleFunc("GET /v1/compute/instances/{id}/products/available", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		products := f.products
		f.mu.Unlock()
		writeData(w, http.StatusOK, products...)
	})
	m.HandleFunc("POST /v1/compute/instances/{id}/actions/{action}", func(w http.ResponseWriter, r *http.Request) {
		id := pathInt(r, "id")
		f.mu.Lock()
		instance, ok := f.instances[id]
		if ok {
			switch r.PathValue("action") {
			case "start", "restart":
				instance["status"] = "running"
			case "stop", "shutdown":
				instance["status"] = "stopped"
			case "rescue":
				instance["status"] = "rescue"
			}
		}
		f.mu.Unlock()
		if !ok {
			writeAPIError(w, http.StatusNotFound, "instance not found")
			return
		}
		writeData(w, http.StatusCreated, map[string]interface{}{"instanceId": id, "action": r.PathValue("action")})
	})
	m.HandleFunc("GET /v1/compute/instances/{id}/vnc", func(w http.ResponseWriter, r *http.Request) {
		id := pathInt(r, "id")
		f.mu.Lock()
		vnc, ok := f.vnc[id]
		if !ok {
			vnc = map[string]interface{}{"tenantId": "INT", "customerId": "12345", "instanceId": id, "enabled": true, "vncIp": "198.51.100.7", "vncPort": 63031}
			f.vnc[id] = vnc
		}
		vnc = copyMap(vnc)
		f.mu.Unlock()
		writeData(w, http.StatusOK, vnc)
	})
	m.HandleFunc("PATCH /v1/compute/instances/{id}/vnc", func(w http.ResponseWriter, r *http.Request) {
		id := pathInt(r, "id")
		body := readBody(r)
		f.mu.Lock()
		vnc, ok := f.vnc[id]
		if !ok {
			vnc = map[string]interface{}{"tenantId": "INT", "customerId": "12345", "instanceId": id, "enabled": true, "vncIp": "198.51.100.7", "vncPort": 63031}
			f.vnc[id] = vnc
		}
		password, _ := body["vncPassword"].(string)
		if !vncPasswordPattern.MatchString(password) {
			f.mu.Unlock()
			// Like the real API: the password is mandatory, and "enabled" is ignored.
			writeJSON(w, http.StatusBadRequest, map[string]interface{}{"statusCode": 400, "message": []string{"vncPassword must be longer than or equal to 8 characters"}})
			return
		}
		vnc["_password"] = password
		vnc = copyMap(vnc)
		f.mu.Unlock()
		writeData(w, http.StatusOK, vnc)
	})
	m.HandleFunc("GET /v1/compute/images/stats", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		stats := copyMap(f.imageStats)
		f.mu.Unlock()
		writeData(w, http.StatusOK, stats)
	})

	f.firewallRoutes()
	f.vipRoutes()
	f.userRoutes()

	m.HandleFunc("GET /v1/tags/{tagId}/assignments", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		items := f.tagAssignments[pathInt(r, "tagId")]
		f.mu.Unlock()
		writePage(w, r, items)
	})
}

func (f *fakeAPI) firewallJSON(firewall map[string]interface{}) map[string]interface{} {
	out := copyMap(firewall)
	ids, _ := firewall["_instances"].([]int64)
	var statuses []interface{}
	encoded := make([]string, 0, len(ids))
	for _, id := range ids {
		statuses = append(statuses, map[string]interface{}{"instanceId": id, "status": "ok"})
		encoded = append(encoded, strconv.FormatInt(id, 10))
	}
	if statuses == nil {
		statuses = []interface{}{}
	}
	out["instanceStatus"] = statuses
	// The real API returns the instances as a JSON-encoded string.
	out["instances"] = "[" + strings.Join(encoded, ",") + "]"
	delete(out, "_instances")
	return out
}

func (f *fakeAPI) firewallRoutes() {
	m := f.mux
	m.HandleFunc("POST /v1/firewalls", func(w http.ResponseWriter, r *http.Request) {
		body := readBody(r)
		f.mu.Lock()
		id := fmt.Sprintf("fw-%d", f.id())
		rules, _ := body["rules"].(map[string]interface{})
		firewall := map[string]interface{}{
			"tenantId": "INT", "customerId": "12345", "firewallId": id,
			"name": body["name"], "description": body["description"], "status": body["status"],
			"rules":       normalizeFakeRules(rules),
			"createdDate": time.Now().UTC().Format(time.RFC3339), "updatedDate": time.Now().UTC().Format(time.RFC3339),
			"_instances": []int64{},
		}
		f.firewalls[id] = firewall
		out := f.firewallJSON(firewall)
		f.mu.Unlock()
		writeData(w, http.StatusCreated, out)
	})
	m.HandleFunc("GET /v1/firewalls", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		var items []map[string]interface{}
		for _, firewall := range f.firewalls {
			if name := r.URL.Query().Get("name"); name != "" && !strings.Contains(firewall["name"].(string), name) {
				continue
			}
			items = append(items, f.firewallJSON(firewall))
		}
		f.mu.Unlock()
		sort.Slice(items, func(i, j int) bool { return items[i]["firewallId"].(string) < items[j]["firewallId"].(string) })
		writePage(w, r, items)
	})
	m.HandleFunc("GET /v1/firewalls/{id}", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		firewall, ok := f.firewalls[r.PathValue("id")]
		var out map[string]interface{}
		if ok {
			out = f.firewallJSON(firewall)
		}
		f.mu.Unlock()
		if !ok {
			writeAPIError(w, http.StatusNotFound, "firewall not found")
			return
		}
		writeData(w, http.StatusOK, out)
	})
	m.HandleFunc("PATCH /v1/firewalls/{id}", func(w http.ResponseWriter, r *http.Request) {
		body := readBody(r)
		f.mu.Lock()
		firewall, ok := f.firewalls[r.PathValue("id")]
		if ok {
			for _, key := range []string{"name", "description", "status"} {
				if v, ok := body[key]; ok {
					firewall[key] = v
				}
			}
		}
		f.mu.Unlock()
		if !ok {
			writeAPIError(w, http.StatusNotFound, "firewall not found")
			return
		}
		writeData(w, http.StatusOK)
	})
	m.HandleFunc("PUT /v1/firewalls/{id}", func(w http.ResponseWriter, r *http.Request) {
		body := readBody(r)
		f.mu.Lock()
		firewall, ok := f.firewalls[r.PathValue("id")]
		if ok {
			rules, _ := body["rules"].(map[string]interface{})
			firewall["rules"] = normalizeFakeRules(rules)
		}
		f.mu.Unlock()
		if !ok {
			writeAPIError(w, http.StatusNotFound, "firewall not found")
			return
		}
		writeData(w, http.StatusOK)
	})
	m.HandleFunc("DELETE /v1/firewalls/{id}", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		_, ok := f.firewalls[r.PathValue("id")]
		delete(f.firewalls, r.PathValue("id"))
		f.mu.Unlock()
		if !ok {
			writeAPIError(w, http.StatusNotFound, "firewall not found")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	assign := func(add bool) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			instanceId := pathInt(r, "instanceId")
			f.mu.Lock()
			firewall, ok := f.firewalls[r.PathValue("id")]
			_, instanceOk := f.instances[instanceId]
			if ok && instanceOk {
				ids := firewall["_instances"].([]int64)
				var next []int64
				for _, id := range ids {
					if id != instanceId {
						next = append(next, id)
					}
				}
				if add {
					next = append(next, instanceId)
				}
				if next == nil {
					next = []int64{}
				}
				firewall["_instances"] = next
			}
			f.mu.Unlock()
			if !ok || !instanceOk {
				writeAPIError(w, http.StatusNotFound, "firewall or instance not found")
				return
			}
			writeData(w, http.StatusOK)
		}
	}
	m.HandleFunc("POST /v1/firewalls/{id}/instances/{instanceId}", assign(true))
	m.HandleFunc("DELETE /v1/firewalls/{id}/instances/{instanceId}", assign(false))
}

// normalizeFakeRules stores rules the way the API returns them: every rule
// has a display name and an srcCidr object without empty lists.
func normalizeFakeRules(rules map[string]interface{}) map[string]interface{} {
	inbound, _ := rules["inbound"].([]interface{})
	out := []interface{}{}
	for _, item := range inbound {
		rule := copyMap(item.(map[string]interface{}))
		if _, ok := rule["displayName"]; !ok || rule["displayName"] == "" {
			// The real API returns null for rules without a name.
			rule["displayName"] = nil
		}
		if cidr, ok := rule["srcCidr"].(map[string]interface{}); ok {
			cidr = copyMap(cidr)
			for _, key := range []string{"ipv4", "ipv6"} {
				if list, ok := cidr[key].([]interface{}); ok && len(list) == 0 {
					delete(cidr, key)
				}
			}
			rule["srcCidr"] = cidr
		}
		out = append(out, rule)
	}
	// Like the real API, append the drop-all rule.
	out = append(out, map[string]interface{}{
		"protocol": "", "destPorts": []interface{}{}, "srcCidr": map[string]interface{}{"ipv4": []interface{}{}, "ipv6": []interface{}{}},
		"action": "drop", "status": "active", "displayName": "Block all traffic",
	})
	return map[string]interface{}{"inbound": out}
}
