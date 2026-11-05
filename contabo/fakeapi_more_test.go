package contabo

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
)

func (f *fakeAPI) addVip(ip string, vipType string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.vips[ip] = map[string]interface{}{
		"tenantId": "INT", "customerId": "12345", "vipId": fmt.Sprintf("vip-%d", f.id()),
		"dataCenter": "European Union 2", "region": "EU", "resourceId": "", "resourceType": "null",
		"resourceName": "", "resourceDisplayName": "", "ipVersion": "v4", "type": vipType,
		"v4": map[string]interface{}{"ip": ip, "netmaskCidr": 32, "gateway": "203.0.113.1", "network": ip, "broadcast": ip},
	}
}

func (f *fakeAPI) vipRoutes() {
	m := f.mux
	m.HandleFunc("GET /v1/vips", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		ips := make([]string, 0, len(f.vips))
		for ip := range f.vips {
			ips = append(ips, ip)
		}
		sort.Strings(ips)
		var items []map[string]interface{}
		for _, ip := range ips {
			vip := f.vips[ip]
			if v := r.URL.Query().Get("type"); v != "" && vip["type"] != v {
				continue
			}
			items = append(items, copyMap(vip))
		}
		f.mu.Unlock()
		writePage(w, r, items)
	})
	m.HandleFunc("GET /v1/vips/{ip}", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		vip, ok := f.vips[r.PathValue("ip")]
		if ok {
			vip = copyMap(vip)
		}
		f.mu.Unlock()
		if !ok {
			writeAPIError(w, http.StatusNotFound, "vip not found")
			return
		}
		writeData(w, http.StatusOK, vip)
	})
	assign := func(add bool) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			f.mu.Lock()
			vip, ok := f.vips[r.PathValue("ip")]
			conflict := false
			if ok {
				if add {
					if vip["resourceId"] != "" {
						conflict = true
					} else {
						vip["resourceId"] = r.PathValue("resourceId")
						vip["resourceType"] = r.PathValue("resourceType")
					}
				} else {
					vip["resourceId"] = ""
					vip["resourceType"] = "null"
				}
			}
			f.mu.Unlock()
			switch {
			case !ok:
				writeAPIError(w, http.StatusNotFound, "vip not found")
			case conflict:
				writeAPIError(w, http.StatusConflict, "vip already assigned")
			default:
				writeData(w, http.StatusOK)
			}
		}
	}
	m.HandleFunc("POST /v1/vips/{ip}/{resourceType}/{resourceId}", assign(true))
	m.HandleFunc("DELETE /v1/vips/{ip}/{resourceType}/{resourceId}", assign(false))
}

func (f *fakeAPI) userRoutes() {
	m := f.mux

	m.HandleFunc("GET /v1/roles/api-permissions", func(w http.ResponseWriter, r *http.Request) {
		writePage(w, r, []map[string]interface{}{
			{"apiName": "/v1/dns/zones", "actions": []string{"READ", "CREATE"}},
			{"apiName": "/v1/dns/zones/:zoneName/records", "actions": []string{"READ", "CREATE"}},
			{"apiName": "/v1/compute/instances", "actions": []string{"READ", "CREATE"}},
		})
	})
	m.HandleFunc("GET /v1/roles", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		var items []map[string]interface{}
		ids := make([]int64, 0, len(f.roles))
		for id := range f.roles {
			ids = append(ids, id)
		}
		sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
		for _, id := range ids {
			items = append(items, copyMap(f.roles[id]))
		}
		f.mu.Unlock()
		writePage(w, r, items)
	})
	storeRole := func(id int64, body map[string]interface{}) map[string]interface{} {
		permissions := []interface{}{}
		if list, ok := body["permissions"].([]interface{}); ok {
			for _, item := range list {
				permission := copyMap(item.(map[string]interface{}))
				var resources []interface{}
				if tagIds, ok := permission["resources"].([]interface{}); ok {
					for _, tagId := range tagIds {
						resources = append(resources, map[string]interface{}{"tagId": tagId, "tagName": "tag"})
					}
				}
				if resources == nil {
					resources = []interface{}{}
				}
				permission["resources"] = resources
				permissions = append(permissions, permission)
			}
		}
		return map[string]interface{}{
			"roleId": id, "tenantId": "INT", "customerId": "12345", "name": body["name"],
			"admin": body["admin"], "accessAllResources": body["accessAllResources"],
			"type": "custom", "permissions": permissions,
		}
	}
	m.HandleFunc("POST /v1/roles", func(w http.ResponseWriter, r *http.Request) {
		body := readBody(r)
		f.mu.Lock()
		id := f.id()
		f.roles[id] = storeRole(id, body)
		f.mu.Unlock()
		writeData(w, http.StatusCreated, map[string]interface{}{"roleId": id})
	})
	m.HandleFunc("GET /v1/roles/{id}", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		role, ok := f.roles[pathInt(r, "id")]
		if ok {
			role = copyMap(role)
		}
		f.mu.Unlock()
		if !ok {
			writeAPIError(w, http.StatusNotFound, "role not found")
			return
		}
		writeData(w, http.StatusOK, role)
	})
	m.HandleFunc("PUT /v1/roles/{id}", func(w http.ResponseWriter, r *http.Request) {
		body := readBody(r)
		id := pathInt(r, "id")
		f.mu.Lock()
		_, ok := f.roles[id]
		if ok {
			f.roles[id] = storeRole(id, body)
		}
		f.mu.Unlock()
		if !ok {
			writeAPIError(w, http.StatusNotFound, "role not found")
			return
		}
		writeData(w, http.StatusOK, map[string]interface{}{"roleId": id})
	})
	m.HandleFunc("DELETE /v1/roles/{id}", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		delete(f.roles, pathInt(r, "id"))
		f.mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	})

	storeUser := func(user map[string]interface{}, body map[string]interface{}) {
		for _, key := range []string{"firstName", "lastName", "email", "enabled", "totp", "secondaryEmail", "locale", "sendInvoiceEmail"} {
			if v, ok := body[key]; ok {
				user[key] = v
			}
		}
		if ids, ok := body["roles"].([]interface{}); ok {
			roles := []interface{}{}
			for _, id := range ids {
				roles = append(roles, map[string]interface{}{"roleId": id, "name": "role"})
			}
			user["roles"] = roles
		}
	}
	m.HandleFunc("GET /v1/users", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		var items []map[string]interface{}
		for _, user := range f.users {
			items = append(items, copyMap(user))
		}
		f.mu.Unlock()
		sort.Slice(items, func(i, j int) bool { return items[i]["userId"].(string) < items[j]["userId"].(string) })
		writePage(w, r, items)
	})
	m.HandleFunc("POST /v1/users", func(w http.ResponseWriter, r *http.Request) {
		body := readBody(r)
		f.mu.Lock()
		id := fmt.Sprintf("user-%d", f.id())
		user := map[string]interface{}{
			"tenantId": "INT", "customerId": "12345", "userId": id, "emailVerified": false,
			"owner": false, "roles": []interface{}{}, "secondaryEmailAddress": nil,
		}
		storeUser(user, body)
		f.users[id] = user
		f.mu.Unlock()
		writeData(w, http.StatusCreated, map[string]interface{}{"userId": id})
	})
	m.HandleFunc("GET /v1/users/{id}", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		user, ok := f.users[r.PathValue("id")]
		if ok {
			user = copyMap(user)
		}
		f.mu.Unlock()
		if !ok {
			writeAPIError(w, http.StatusNotFound, "user not found")
			return
		}
		writeData(w, http.StatusOK, user)
	})
	m.HandleFunc("PATCH /v1/users/{id}", func(w http.ResponseWriter, r *http.Request) {
		body := readBody(r)
		f.mu.Lock()
		user, ok := f.users[r.PathValue("id")]
		if ok {
			storeUser(user, body)
		}
		f.mu.Unlock()
		if !ok {
			writeAPIError(w, http.StatusNotFound, "user not found")
			return
		}
		writeData(w, http.StatusOK)
	})
	m.HandleFunc("DELETE /v1/users/{id}", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		delete(f.users, r.PathValue("id"))
		f.mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	})
	m.HandleFunc("GET /v1/users/{id}/object-storages/credentials", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("id") != f.userId {
			writeAPIError(w, http.StatusForbidden, "wrong user")
			return
		}
		f.mu.Lock()
		var items []map[string]interface{}
		for _, credential := range f.credentials {
			if id := r.URL.Query().Get("objectStorageId"); id != "" && credential["objectStorageId"] != id {
				continue
			}
			items = append(items, copyMap(credential))
		}
		f.mu.Unlock()
		writePage(w, r, items)
	})
}

func (f *fakeAPI) addCredential(objectStorageId string, region string) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	id := f.id()
	accessKey := "AK" + strconv.FormatInt(id, 10)
	f.credentials = append(f.credentials, map[string]interface{}{
		"tenantId": "INT", "customerId": "12345", "accessKey": accessKey, "secretKey": "SK" + strconv.FormatInt(id, 10),
		"objectStorageId": objectStorageId, "displayName": "storage " + objectStorageId, "region": region, "credentialId": id,
	})
	return accessKey
}
