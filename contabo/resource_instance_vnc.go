package contabo

import (
	"context"
	"net/http"
	"regexp"
	"strconv"

	"contabo.com/openapi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// The API only accepts alphanumeric VNC passwords with at least 8
// characters. It ignores an "enabled" attribute, so VNC cannot be turned off
// through the API (checked against the live API on 2026-11-05).
var vncPasswordPattern = regexp.MustCompile(`^[A-Za-z0-9]{8,}$`)

func resourceInstanceVnc() *schema.Resource {
	return &schema.Resource{
		Description:   "This resource sets the password of the VNC console of an instance. Contabo turns on the VNC console for every instance, and the API cannot turn the VNC console off. A strong VNC password is the best protection that the API offers.\n\nThe API does not return the password, so Terraform cannot see a password change that someone makes in the Customer Control Panel. When Terraform destroys this resource, the VNC console keeps the last password.",
		CreateContext: resourceInstanceVncSet,
		ReadContext:   resourceInstanceVncRead,
		UpdateContext: resourceInstanceVncSet,
		DeleteContext: schema.NoopContext,
		Schema: map[string]*schema.Schema{
			"instance_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The id of the instance.",
			},
			"password": {
				Type:      schema.TypeString,
				Required:  true,
				Sensitive: true,
				ValidateFunc: validation.StringMatch(vncPasswordPattern,
					"the VNC password must have at least 8 characters, and the characters must be letters or digits"),
				Description: "The password of the VNC console. The password must have at least 8 characters, and the characters must be letters or digits. Many VNC clients only use the first 8 characters. Terraform stores the password in the Terraform state.",
			},
			"enabled": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether the VNC console of the instance is enabled.",
			},
			"vnc_ip": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The IP address of the VNC console.",
			},
			"vnc_port": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The port of the VNC console.",
			},
		},
	}
}

func resourceInstanceVncSet(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*openapi.APIClient)
	instanceId, err := strconv.ParseInt(d.Get("instance_id").(string), 10, 64)
	if err != nil {
		return diag.Errorf("instance_id must be a numeric instance id: %s", err)
	}
	httpResp, err := apiRequest(ctx, client, http.MethodPatch, instancePath(instanceId)+"/vnc", nil,
		map[string]interface{}{"vncPassword": d.Get("password").(string)}, nil)
	if err != nil {
		return apiErrorDiags(httpResp, err)
	}
	d.SetId(strconv.FormatInt(instanceId, 10))
	return resourceInstanceVncRead(ctx, d, m)
}

func resourceInstanceVncRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*openapi.APIClient)
	instanceId, err := strconv.ParseInt(d.Id(), 10, 64)
	if err != nil {
		return diag.Errorf("invalid instance id %q: %s", d.Id(), err)
	}
	vnc, httpResp, err := retrieveInstanceVnc(ctx, client, instanceId)
	if err != nil {
		if removeFromStateIfNotFound(d, httpResp) {
			return nil
		}
		return apiErrorDiags(httpResp, err)
	}
	return setSchemaValues(d, resourceInstanceVnc().Schema, map[string]interface{}{
		"instance_id": d.Id(),
		"enabled":     vnc.Enabled,
		"vnc_ip":      vnc.VncIp,
		"vnc_port":    int(vnc.VncPort),
	})
}
