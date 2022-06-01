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
				Type:     schema.TypeString,
				Required: true,
			},
			"name": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"instance_id": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"created_date": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"auto_delete_date": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"image_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"image_name": {
				Type:     schema.TypeString,
				Computed: true,
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
