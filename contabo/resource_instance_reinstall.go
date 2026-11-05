package contabo

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"contabo.com/openapi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceInstanceReinstall() *schema.Resource {
	return &schema.Resource{
		Description:   "This resource reinstalls an existing instance. The provider reinstalls the instance when Terraform creates this resource. The provider reinstalls the instance again whenever a setting of this resource or a value in `triggers` changes. For example, an instance needs a fresh installation after you attach a private network to the instance. When Terraform destroys this resource, nothing happens to the instance.\n\nCAUTION: A reinstall erases the disk of the instance.\n\nWhen this resource manages the installation of an instance, add `image_id`, `ssh_keys`, `root_password`, `user_data`, `default_user` and `application_id` to the `lifecycle.ignore_changes` list of the `contabo_instance` resource. Otherwise the two resources take turns to reinstall the instance.",
		CreateContext: resourceInstanceReinstallCreate,
		ReadContext:   resourceInstanceReinstallRead,
		DeleteContext: schema.NoopContext,
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(60 * time.Minute),
		},
		Schema: map[string]*schema.Schema{
			"instance_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The id of the instance that the provider reinstalls.",
			},
			"image_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The id of the image that the provider installs.",
			},
			"ssh_keys": {
				Type:        schema.TypeList,
				Optional:    true,
				ForceNew:    true,
				Elem:        &schema.Schema{Type: schema.TypeInt},
				Description: "The secret ids of the public SSH keys. The installation adds these keys for the default user.",
			},
			"root_password": {
				Type:        schema.TypeInt,
				Optional:    true,
				ForceNew:    true,
				Description: "The secret id of the root password.",
			},
			"user_data": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "The cloud-init configuration that the instance runs on the first start.",
			},
			"default_user": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice([]string{"root", "admin", "administrator"}, false),
				Description:  "The default user that the installation creates. The value is `root` or `admin`, or `administrator` on Windows. When you do not set this value, the API uses `admin`.",
			},
			"application_id": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "The id of an application that the provider installs together with the image.",
			},
			"triggers": {
				Type:        schema.TypeMap,
				Optional:    true,
				ForceNew:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "A map of any values that you choose. When a value in this map changes, the provider reinstalls the instance.",
			},
			"reinstalled_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The time of the reinstall in RFC 3339 format.",
			},
			"status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The status of the instance after the reinstall.",
			},
		},
	}
}

func resourceInstanceReinstallCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*openapi.APIClient)
	instanceId, err := strconv.ParseInt(d.Get("instance_id").(string), 10, 64)
	if err != nil {
		return diag.Errorf("instance_id must be a numeric instance id: %s", err)
	}

	request := map[string]interface{}{"imageId": d.Get("image_id").(string)}
	setIfNotEmpty(request, "userData", d.Get("user_data").(string))
	setIfNotEmpty(request, "defaultUser", d.Get("default_user").(string))
	setIfNotEmpty(request, "applicationId", d.Get("application_id").(string))
	if sshKeys := interfacesToInt64s(d.Get("ssh_keys").([]interface{})); len(sshKeys) > 0 {
		request["sshKeys"] = sshKeys
	}
	if rootPassword := d.Get("root_password").(int); rootPassword > 0 {
		request["rootPassword"] = rootPassword
	}

	if diags := reinstallInstance(ctx, client, instanceId, request, d.Timeout(schema.TimeoutCreate)); diags.HasError() {
		return diags
	}
	reinstalledAt := time.Now().UTC().Format(time.RFC3339)
	d.SetId(fmt.Sprintf("%d/%s", instanceId, reinstalledAt))
	_ = d.Set("reinstalled_at", reinstalledAt)
	return resourceInstanceReinstallRead(ctx, d, m)
}

func resourceInstanceReinstallRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*openapi.APIClient)
	instanceId, err := strconv.ParseInt(d.Get("instance_id").(string), 10, 64)
	if err != nil {
		return diag.Errorf("instance_id must be a numeric instance id: %s", err)
	}
	instance, httpResp, err := retrieveInstance(ctx, client, instanceId)
	if err != nil {
		if removeFromStateIfNotFound(d, httpResp) {
			return nil
		}
		return apiErrorDiags(httpResp, err)
	}
	if err := d.Set("status", instance.Status); err != nil {
		return diag.FromErr(err)
	}
	return nil
}
