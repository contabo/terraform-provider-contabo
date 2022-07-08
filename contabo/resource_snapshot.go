package contabo

import (
	"context"
	"time"

	"contabo.com/openapi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	uuid "github.com/satori/go.uuid"
)

func resourceSnapshot() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceSnapshotCreate,
		ReadContext:   resourceSnapshotRead,
		UpdateContext: resourceSnapshotUpdate,
		DeleteContext: resourceSnapshotDelete,
		Schema: map[string]*schema.Schema{
			"id": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The identifier of the instance snapshot. Use it to manage it!",
			},
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Name of the snapshot.",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Description of this snapshot.",
			},
			"instance_id": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Instance identifier associated with the snapshot.",
			},
			"created_date": {
				Type:        schema.TypeString,
				Computed:    true,
				Optional:    true,
				Description: "The creation date of this instance snapshot.",
			},
			"auto_delete_date": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The date when the snapshot will be autmatically deleted.",
			},
			"image_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Id of the Image the snapshot was taken from.",
			},
			"image_name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Name of the Image the snapshot was taken from.",
			},
		},
	}
}

func resourceSnapshotCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*openapi.APIClient)

	createSnapshotRequest := openapi.NewCreateSnapshotRequestWithDefaults()

	name := d.Get("name").(string)
	description := d.Get("description").(string)
	instanceId := d.Get("instance_id").(int)
	if name != "" {
		createSnapshotRequest.Name = name
	}
	if description != "" {
		createSnapshotRequest.Description = &description
	}
	var instanceId64 int64

	instanceId64 = int64(instanceId)
	res, httpResp, err := client.SnapshotsApi.
		CreateSnapshot(ctx, instanceId64).
		XRequestId(uuid.NewV4().String()).
		CreateSnapshotRequest(*createSnapshotRequest).
		Execute()
	if err != nil {
		return HandleResponseErrors(diags, httpResp)
	} else if len(res.Data) != 1 {
		return MultipleDataObjectsError(diags)
	}

	d.SetId(res.Data[0].SnapshotId)

	return resourceSnapshotRead(ctx, d, m)
}

func resourceSnapshotRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*openapi.APIClient)

	snapshotId := d.Id()

	instanceId := d.Get("instance_id").(int)
	var instanceId64 int64
	instanceId64 = int64(instanceId)

	res, httpResp, err := client.SnapshotsApi.
		RetrieveSnapshot(ctx, instanceId64, snapshotId).
		XRequestId(uuid.NewV4().String()).
		Execute()

	if err != nil {
		return HandleResponseErrors(diags, httpResp)
	} else if len(res.Data) != 1 {
		return MultipleDataObjectsError(diags)
	}

	return AddSnapshotToData(res.Data[0], d, diags)
}

func resourceSnapshotUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*openapi.APIClient)
	anyChange := false
	patchSnapshotRequest := openapi.NewUpdateSnapshotRequest()

	if d.HasChange("name") || d.HasChange("description") {
		newName := d.Get("name").(string)
		newDescription := d.Get("name").(string)
		patchSnapshotRequest.Name = &newName
		patchSnapshotRequest.Description = &newDescription
		anyChange = true
	}

	snapshotId := d.Id()
	instanceId := d.Get("instance_id").(int64)

	if anyChange {
		_, httpResp, err := client.SnapshotsApi.
			UpdateSnapshot(ctx, instanceId, snapshotId).
			XRequestId(uuid.NewV4().String()).
			UpdateSnapshotRequest(*patchSnapshotRequest).
			Execute()

		if err != nil {
			return HandleResponseErrors(diags, httpResp)
		} else {
			d.SetId(snapshotId)
			d.Set("instanceId", instanceId)
			return resourceSnapshotRead(ctx, d, m)
		}
	}
	return diags
}

func resourceSnapshotDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*openapi.APIClient)

	snapshotId := d.Id()

	instanceId := d.Get("instance_id").(int)
	var instanceId64 int64 = int64(instanceId)

	httpResp, err := client.SnapshotsApi.
		DeleteSnapshot(ctx, instanceId64, snapshotId).
		XRequestId(uuid.NewV4().String()).
		Execute()

	if err != nil {
		return HandleResponseErrors(diags, httpResp)
	}

	d.SetId("")

	return diags
}

func AddSnapshotToData(
	snapshot openapi.SnapshotResponse,
	d *schema.ResourceData,
	diags diag.Diagnostics,
) diag.Diagnostics {
	id := snapshot.SnapshotId
	if err := d.Set("id", id); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("name", snapshot.Name); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("image_id", snapshot.ImageId); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("instance_id", snapshot.InstanceId); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("description", snapshot.Description); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("image_name", snapshot.ImageName); err != nil {
		return diag.FromErr(err)
	}
	createdDate := snapshot.CreatedDate.Format(time.RFC850)
	if err := d.Set("created_date", createdDate); err != nil {
		return diag.FromErr(err)
	}
	autoDeleteDate := snapshot.AutoDeleteDate.Format(time.RFC850)
	if err := d.Set("auto_delete_date", autoDeleteDate); err != nil {
		return diag.FromErr(err)
	}

	return diags
}
