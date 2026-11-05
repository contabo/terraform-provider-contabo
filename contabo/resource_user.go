package contabo

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"contabo.com/openapi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

var userLocales = []string{"de-DE", "de", "en-US", "en", "es", "es-ES", "pt-BR", "pt"}

type userResponse struct {
	UserId           string `json:"userId"`
	FirstName        string `json:"firstName"`
	LastName         string `json:"lastName"`
	Email            string `json:"email"`
	EmailVerified    bool   `json:"emailVerified"`
	Enabled          bool   `json:"enabled"`
	Totp             bool   `json:"totp"`
	SecondaryEmail   bool   `json:"secondaryEmail"`
	Locale           string `json:"locale"`
	Owner            bool   `json:"owner"`
	SendInvoiceEmail bool   `json:"sendInvoiceEmail"`
	Roles            []struct {
		RoleId int64  `json:"roleId"`
		Name   string `json:"name"`
	} `json:"roles"`
}

func userPath(userId string) string {
	return "/v1/users/" + url.PathEscape(userId)
}

func resourceUser() *schema.Resource {
	return &schema.Resource{
		Description:   "This resource manages a user of your Contabo account. The API sends the new user an email with a link to set a password. With the password, the user gets API credentials. The API credentials of the user only allow what the roles of the user allow. When Terraform destroys this resource, the provider deletes the user. This resource cannot manage the owner of the account.",
		CreateContext: resourceUserCreate,
		ReadContext:   resourceUserRead,
		UpdateContext: resourceUserUpdate,
		DeleteContext: resourceUserDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"email": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The email address of the user. Contabo sends the activation link and the password reset links to this address. When you change the email address, Terraform creates a new user.",
			},
			"first_name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The first name of the user.",
			},
			"last_name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The last name of the user.",
			},
			"enabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Whether the user can log in. The default value is `true`.",
			},
			"totp": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Whether the user must log in with a second code from an authenticator app. The default value is `false`.",
			},
			"secondary_email": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Whether the user must log in with a second code that Contabo sends to a second email address. The default value is `false`.",
			},
			"locale": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "en",
				ValidateFunc: validation.StringInSlice(userLocales, false),
				Description:  "The language setting of the user. The value is `de-DE`, `de`, `en-US`, `en`, `es-ES`, `es`, `pt-BR` or `pt`. The default value is `en`.",
			},
			"roles": {
				Type:        schema.TypeSet,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeInt},
				Description: "The ids of the roles of the user. You can use the ids of `contabo_role` resources.",
			},
			"send_invoice_email": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Whether the user receives emails with invoices. The default value is `false`.",
			},
			"email_verified": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether the user has confirmed the email address.",
			},
			"owner": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether the user is the owner of the account.",
			},
		},
	}
}

func expandUser(d *schema.ResourceData, create bool) map[string]interface{} {
	request := map[string]interface{}{
		"firstName":        d.Get("first_name").(string),
		"lastName":         d.Get("last_name").(string),
		"enabled":          d.Get("enabled").(bool),
		"totp":             d.Get("totp").(bool),
		"secondaryEmail":   d.Get("secondary_email").(bool),
		"locale":           d.Get("locale").(string),
		"roles":            setToInt64s(d.Get("roles").(*schema.Set)),
		"sendInvoiceEmail": d.Get("send_invoice_email").(bool),
	}
	if create {
		request["email"] = d.Get("email").(string)
	}
	return request
}

func resourceUserCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*openapi.APIClient)
	var res struct {
		Data []struct {
			UserId string `json:"userId"`
		} `json:"data"`
	}
	httpResp, err := apiRequest(ctx, client, http.MethodPost, "/v1/users", nil, expandUser(d, true), &res)
	if err != nil {
		return apiErrorDiags(httpResp, err)
	}
	if len(res.Data) != 1 {
		return MultipleDataObjectsError(nil)
	}
	d.SetId(res.Data[0].UserId)
	return resourceUserRead(ctx, d, m)
}

func retrieveUser(ctx context.Context, client *openapi.APIClient, userId string) (*userResponse, *http.Response, error) {
	var res struct {
		Data []userResponse `json:"data"`
	}
	httpResp, err := apiRequest(ctx, client, http.MethodGet, userPath(userId), nil, nil, &res)
	if err != nil {
		return nil, httpResp, err
	}
	if len(res.Data) != 1 {
		return nil, httpResp, fmt.Errorf("expected one user with id %s", userId)
	}
	return &res.Data[0], httpResp, nil
}

func flattenUser(user userResponse) map[string]interface{} {
	roles := make([]interface{}, 0, len(user.Roles))
	for _, role := range user.Roles {
		roles = append(roles, int(role.RoleId))
	}
	return map[string]interface{}{
		"email":              user.Email,
		"first_name":         user.FirstName,
		"last_name":          user.LastName,
		"enabled":            user.Enabled,
		"totp":               user.Totp,
		"secondary_email":    user.SecondaryEmail,
		"locale":             user.Locale,
		"roles":              roles,
		"send_invoice_email": user.SendInvoiceEmail,
		"email_verified":     user.EmailVerified,
		"owner":              user.Owner,
	}
}

func resourceUserRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*openapi.APIClient)
	user, httpResp, err := retrieveUser(ctx, client, d.Id())
	if err != nil {
		if removeFromStateIfNotFound(d, httpResp) {
			return nil
		}
		return apiErrorDiags(httpResp, err)
	}
	if user.Owner {
		return diag.Errorf("user %s is the account owner, which contabo_user does not manage", d.Id())
	}
	return setSchemaValues(d, resourceUser().Schema, flattenUser(*user))
}

func resourceUserUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*openapi.APIClient)
	httpResp, err := apiRequest(ctx, client, http.MethodPatch, userPath(d.Id()), nil, expandUser(d, false), nil)
	if err != nil {
		return apiErrorDiags(httpResp, err)
	}
	return resourceUserRead(ctx, d, m)
}

func resourceUserDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*openapi.APIClient)
	user, httpResp, err := retrieveUser(ctx, client, d.Id())
	if err != nil {
		if isNotFound(httpResp) {
			return nil
		}
		return apiErrorDiags(httpResp, err)
	}
	if user.Owner {
		return diag.Errorf("refusing to delete user %s: it is the account owner", d.Id())
	}
	httpResp, err = apiRequest(ctx, client, http.MethodDelete, userPath(d.Id()), nil, nil, nil)
	if err != nil && !isNotFound(httpResp) {
		return apiErrorDiags(httpResp, err)
	}
	return nil
}

// contabo_users data source

func dataSourceUsers() *schema.Resource {
	userAttributes := map[string]*schema.Schema{
		"id": {Type: schema.TypeString, Computed: true, Description: "The id of the user."},
	}
	for name, attribute := range resourceUser().Schema {
		computed := &schema.Schema{Type: attribute.Type, Elem: attribute.Elem, Computed: true, Description: attribute.Description}
		userAttributes[name] = computed
	}
	return &schema.Resource{
		Description: "This data source lists the users of your Contabo account. The list includes the owner of the account.",
		ReadContext: dataSourceUsersRead,
		Schema: map[string]*schema.Schema{
			"email": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "When you set this filter, the list only contains the user with this email address.",
			},
			"users": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "The users that match the filter.",
				Elem:        &schema.Resource{Schema: userAttributes},
			},
		},
	}
}

func dataSourceUsersRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*openapi.APIClient)
	email := d.Get("email").(string)
	query := url.Values{}
	if email != "" {
		query.Set("email", email)
	}
	var users []userResponse
	httpResp, err := listAll(ctx, client, "/v1/users", query, func(body []byte) (int64, error) {
		var page struct {
			Data       []userResponse `json:"data"`
			Pagination pagination     `json:"_pagination"`
		}
		if err := json.Unmarshal(body, &page); err != nil {
			return 0, err
		}
		users = append(users, page.Data...)
		return page.Pagination.TotalPages, nil
	})
	if err != nil {
		return apiErrorDiags(httpResp, err)
	}
	out := []interface{}{}
	ids := []string{}
	for _, user := range users {
		if email != "" && !strings.EqualFold(user.Email, email) {
			continue
		}
		values := flattenUser(user)
		values["id"] = user.UserId
		out = append(out, values)
		ids = append(ids, user.UserId)
	}
	d.SetId(fmt.Sprintf("users-%d", hashStrings(append(ids, email))))
	if err := d.Set("users", out); err != nil {
		return diag.FromErr(err)
	}
	return nil
}
