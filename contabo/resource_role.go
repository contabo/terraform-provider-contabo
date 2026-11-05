package contabo

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"

	"contabo.com/openapi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

var roleActions = []string{"CREATE", "READ", "UPDATE", "DELETE"}

type roleResponse struct {
	RoleId             int64  `json:"roleId"`
	Name               string `json:"name"`
	Admin              bool   `json:"admin"`
	AccessAllResources bool   `json:"accessAllResources"`
	Type               string `json:"type"`
	Permissions        []struct {
		ApiName   string   `json:"apiName"`
		Actions   []string `json:"actions"`
		Resources []struct {
			TagId   int64  `json:"tagId"`
			TagName string `json:"tagName"`
		} `json:"resources"`
	} `json:"permissions"`
}

func rolePath(roleId string) string {
	return "/v1/roles/" + url.PathEscape(roleId)
}

func resourceRole() *schema.Resource {
	return &schema.Resource{
		Description:   "This resource manages a custom role. A role gives users permissions for the API. A role can give permissions for all resources, or only for resources that have certain tags. Use this resource together with the `contabo_user` resource to create users that can only do what the users need, for example only manage DNS records.",
		CreateContext: resourceRoleCreate,
		ReadContext:   resourceRoleRead,
		UpdateContext: resourceRoleUpdate,
		DeleteContext: resourceRoleDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringLenBetween(1, 255),
				Description:  "The name of the role.",
			},
			"admin": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Whether the role gives access to every API endpoint and to every resource. The default value is `false`.",
			},
			"access_all_resources": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Whether the permissions of the role apply to all resources. When the value is `false`, each permission only applies to resources that have one of the tags in the `resources` list of the permission. The default value is `false`.",
			},
			"permission": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "The API permissions of the role. The `contabo_api_permissions` data source lists the API names that you can use.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"api_name": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The API endpoint, for example `/v1/dns/zones` or `/v1/compute/instances/:instanceId`.",
						},
						"actions": {
							Type:        schema.TypeSet,
							Required:    true,
							Elem:        &schema.Schema{Type: schema.TypeString, ValidateFunc: validation.StringInSlice(roleActions, false)},
							Description: "The actions that the permission allows. `CREATE` allows POST requests, `READ` allows GET requests, `UPDATE` allows PATCH and PUT requests, and `DELETE` allows DELETE requests.",
						},
						"resources": {
							Type:        schema.TypeSet,
							Optional:    true,
							Elem:        &schema.Schema{Type: schema.TypeInt},
							Description: "The ids of tags. The permission only applies to resources that have one of these tags. The provider ignores this list when `access_all_resources` is `true`.",
						},
					},
				},
			},
			"type": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The type of the role. The value is `custom` for roles that you create, and `default` for the roles that Contabo provides.",
			},
		},
	}
}

func expandRole(d *schema.ResourceData) map[string]interface{} {
	permissions := []map[string]interface{}{}
	for _, item := range d.Get("permission").(*schema.Set).List() {
		permission := item.(map[string]interface{})
		actions := stringSetToSlice(permission["actions"].(*schema.Set))
		sort.Strings(actions)
		entry := map[string]interface{}{
			"apiName": permission["api_name"].(string),
			"actions": actions,
		}
		if resources := setToInt64s(permission["resources"].(*schema.Set)); len(resources) > 0 {
			entry["resources"] = resources
		}
		permissions = append(permissions, entry)
	}
	return map[string]interface{}{
		"name":               d.Get("name").(string),
		"admin":              d.Get("admin").(bool),
		"accessAllResources": d.Get("access_all_resources").(bool),
		"permissions":        permissions,
	}
}

func resourceRoleCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*openapi.APIClient)
	var res struct {
		Data []struct {
			RoleId int64 `json:"roleId"`
		} `json:"data"`
	}
	httpResp, err := apiRequest(ctx, client, http.MethodPost, "/v1/roles", nil, expandRole(d), &res)
	if err != nil {
		return apiErrorDiags(httpResp, err)
	}
	if len(res.Data) != 1 {
		return MultipleDataObjectsError(nil)
	}
	d.SetId(strconv.FormatInt(res.Data[0].RoleId, 10))
	return resourceRoleRead(ctx, d, m)
}

func retrieveRole(ctx context.Context, client *openapi.APIClient, roleId string) (*roleResponse, *http.Response, error) {
	var res struct {
		Data []roleResponse `json:"data"`
	}
	httpResp, err := apiRequest(ctx, client, http.MethodGet, rolePath(roleId), nil, nil, &res)
	if err != nil {
		return nil, httpResp, err
	}
	if len(res.Data) != 1 {
		return nil, httpResp, fmt.Errorf("expected one role with id %s", roleId)
	}
	return &res.Data[0], httpResp, nil
}

func flattenRolePermissions(role roleResponse) []interface{} {
	permissions := make([]interface{}, 0, len(role.Permissions))
	for _, permission := range role.Permissions {
		resources := make([]interface{}, 0, len(permission.Resources))
		for _, resource := range permission.Resources {
			resources = append(resources, int(resource.TagId))
		}
		actions := make([]interface{}, 0, len(permission.Actions))
		for _, action := range permission.Actions {
			actions = append(actions, action)
		}
		permissions = append(permissions, map[string]interface{}{
			"api_name":  permission.ApiName,
			"actions":   schema.NewSet(schema.HashString, actions),
			"resources": schema.NewSet(schema.HashInt, resources),
		})
	}
	return permissions
}

func resourceRoleRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*openapi.APIClient)
	role, httpResp, err := retrieveRole(ctx, client, d.Id())
	if err != nil {
		if removeFromStateIfNotFound(d, httpResp) {
			return nil
		}
		return apiErrorDiags(httpResp, err)
	}
	return setSchemaValues(d, resourceRole().Schema, map[string]interface{}{
		"name":                 role.Name,
		"admin":                role.Admin,
		"access_all_resources": role.AccessAllResources,
		"permission":           flattenRolePermissions(*role),
		"type":                 role.Type,
	})
}

func resourceRoleUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*openapi.APIClient)
	httpResp, err := apiRequest(ctx, client, http.MethodPut, rolePath(d.Id()), nil, expandRole(d), nil)
	if err != nil {
		return apiErrorDiags(httpResp, err)
	}
	return resourceRoleRead(ctx, d, m)
}

func resourceRoleDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*openapi.APIClient)
	httpResp, err := apiRequest(ctx, client, http.MethodDelete, rolePath(d.Id()), nil, nil, nil)
	if err != nil && !isNotFound(httpResp) {
		return apiErrorDiags(httpResp, err)
	}
	return nil
}

// contabo_role data source

func dataSourceRole() *schema.Resource {
	permission := resourceRole().Schema["permission"].Elem.(*schema.Resource)
	computedPermission := &schema.Resource{Schema: map[string]*schema.Schema{}}
	for name, attribute := range permission.Schema {
		computedPermission.Schema[name] = &schema.Schema{
			Type: attribute.Type, Elem: attribute.Elem, Computed: true, Description: attribute.Description,
		}
	}
	computedPermission.Schema["actions"].Elem = &schema.Schema{Type: schema.TypeString}
	return &schema.Resource{
		Description: "This data source reads one role by id or by name. The role can be a custom role or a role that Contabo provides, for example `S3 Read` or `S3 Read and Write`.",
		ReadContext: dataSourceRoleRead,
		Schema: map[string]*schema.Schema{
			"id": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ExactlyOneOf: []string{"id", "name"},
				Description:  "The id of the role.",
			},
			"name": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ExactlyOneOf: []string{"id", "name"},
				Description:  "The name of the role. Exactly one role must have this name.",
			},
			"admin":                {Type: schema.TypeBool, Computed: true, Description: "Whether the role gives access to every API endpoint and to every resource."},
			"access_all_resources": {Type: schema.TypeBool, Computed: true, Description: "Whether the permissions of the role apply to all resources."},
			"type":                 {Type: schema.TypeString, Computed: true, Description: "The type of the role. The value is `default` or `custom`."},
			"permission": {
				Type:        schema.TypeSet,
				Computed:    true,
				Description: "The API permissions of the role.",
				Elem:        computedPermission,
			},
		},
	}
}

func dataSourceRoleRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*openapi.APIClient)
	var role *roleResponse
	if roleId := d.Get("id").(string); roleId != "" {
		found, httpResp, err := retrieveRole(ctx, client, roleId)
		if err != nil {
			return apiErrorDiags(httpResp, err)
		}
		role = found
	} else {
		name := d.Get("name").(string)
		var matches []roleResponse
		httpResp, err := listAll(ctx, client, "/v1/roles", url.Values{"name": []string{name}}, func(body []byte) (int64, error) {
			var page struct {
				Data       []roleResponse `json:"data"`
				Pagination pagination     `json:"_pagination"`
			}
			if err := json.Unmarshal(body, &page); err != nil {
				return 0, err
			}
			for _, candidate := range page.Data {
				if candidate.Name == name {
					matches = append(matches, candidate)
				}
			}
			return page.Pagination.TotalPages, nil
		})
		if err != nil {
			return apiErrorDiags(httpResp, err)
		}
		if len(matches) != 1 {
			return diag.Errorf("expected one role named %q, found %d", name, len(matches))
		}
		role = &matches[0]
	}
	d.SetId(strconv.FormatInt(role.RoleId, 10))
	return setSchemaValues(d, dataSourceRole().Schema, map[string]interface{}{
		"name":                 role.Name,
		"admin":                role.Admin,
		"access_all_resources": role.AccessAllResources,
		"permission":           flattenRolePermissions(*role),
		"type":                 role.Type,
	})
}

// contabo_api_permissions data source

func dataSourceApiPermissions() *schema.Resource {
	return &schema.Resource{
		Description: "This data source lists the API endpoints that a role can give permissions for. For each API endpoint, the data source lists the actions that the endpoint allows.",
		ReadContext: dataSourceApiPermissionsRead,
		Schema: map[string]*schema.Schema{
			"api_name_prefix": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "When you set this filter, the list only contains API names that start with this text, for example `/v1/dns`.",
			},
			"permissions": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "The API permissions, sorted by the API name.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"api_name": {Type: schema.TypeString, Computed: true, Description: "The API endpoint."},
						"actions":  {Type: schema.TypeList, Computed: true, Elem: &schema.Schema{Type: schema.TypeString}, Description: "The actions that the API endpoint allows."},
					},
				},
			},
		},
	}
}

func dataSourceApiPermissionsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*openapi.APIClient)
	prefix := d.Get("api_name_prefix").(string)
	type apiPermission struct {
		ApiName string   `json:"apiName"`
		Actions []string `json:"actions"`
	}
	var permissions []apiPermission
	httpResp, err := listAll(ctx, client, "/v1/roles/api-permissions", nil, func(body []byte) (int64, error) {
		var page struct {
			Data       []apiPermission `json:"data"`
			Pagination pagination      `json:"_pagination"`
		}
		if err := json.Unmarshal(body, &page); err != nil {
			return 0, err
		}
		permissions = append(permissions, page.Data...)
		// The endpoint is not paginated in practice.
		if page.Pagination.TotalPages == 0 {
			return 1, nil
		}
		return page.Pagination.TotalPages, nil
	})
	if err != nil {
		return apiErrorDiags(httpResp, err)
	}
	sort.Slice(permissions, func(i, j int) bool { return permissions[i].ApiName < permissions[j].ApiName })
	out := []interface{}{}
	for _, permission := range permissions {
		if prefix != "" && (len(permission.ApiName) < len(prefix) || permission.ApiName[:len(prefix)] != prefix) {
			continue
		}
		actions := append([]string(nil), permission.Actions...)
		sort.Strings(actions)
		out = append(out, map[string]interface{}{"api_name": permission.ApiName, "actions": actions})
	}
	d.SetId("api-permissions" + prefix)
	if err := d.Set("permissions", out); err != nil {
		return diag.FromErr(err)
	}
	return nil
}
