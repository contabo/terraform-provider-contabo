package contabo

import (
	"context"
	"strconv"

	apiClient "contabo.com/openapi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	uuid "github.com/satori/go.uuid"
)

func dataSourceSnapshot() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceSnapshotRead,
		Schema: map[string]*schema.Schema{
			"id": {
				Type:        schema.TypeString,
				Required:    true,
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
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Instance identifier associated with the snapshot",
			},
			"created_date": {
				Type:        schema.TypeString,
				Computed:    true,
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

func dataSourceSnapshotRead(
	ctx context.Context,
	d *schema.ResourceData,
	m interface{},
) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*apiClient.APIClient)

	var snapshotId string
	var err error
	id := d.Get("id").(string)
	if id != "" {
		snapshotId = id
	}
	var instanceId int64
	instanceIdStr := d.Get("instance_id").(string)
	if instanceIdStr != "" {
		instanceId, err = strconv.ParseInt(instanceIdStr, 10, 64)
	}
	if err != nil {
		return diag.FromErr(err)
	}

	res, httpResp, err := client.SnapshotsApi.
		RetrieveSnapshot(ctx, int64(instanceId), snapshotId).
		XRequestId(uuid.NewV4().String()).
		Execute()

	if err != nil {
		return HandleResponseErrors(diags, httpResp)
	} else if len(res.Data) != 1 {
		return MultipleDataObjectsError(diags)
	}

	d.SetId(res.Data[0].SnapshotId)

	return AddSnapshotToData(
		res.Data[0],
		d,
		diags,
	)
}
