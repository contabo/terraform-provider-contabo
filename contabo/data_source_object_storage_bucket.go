package contabo

import (
	"context"

	"contabo.com/openapi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceObjectStorageBucket() *schema.Resource {
	return &schema.Resource{
		Description: "Manage buckets on your contabo Object Storage. With this resource you are able to manage your buckets the same way your are able to manage them in your contabo customer panel.",
		ReadContext: dataSourceObjectStorageBucketRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Description: "The name of your bucket, consider the naming restriction https://docs.aws.amazon.com/awscloudtrail/latest/userguide/cloudtrail-s3-bucket-naming-requirements.html.",
				Required:    true,
			},
			"object_storage_id": {
				Type:        schema.TypeString,
				Description: "The contabo objectStorageId on which the bucket should be created.",
				Required:    true,
			},
			"public_sharing": {
				Type:        schema.TypeBool,
				Description: "Choose the access to your bucket. You can not share it at all or share it publicly.",
				Default:     false,
				Optional:    true,
			},
			"public_sharing_link": {
				Type:        schema.TypeString,
				Description: "If your bucket is publicly shared, you can access it with this link.",
				Computed:    true,
			},
			"creation_date": {
				Type:        schema.TypeString,
				Description: "The creation date of the bucket.",
				Computed:    true,
			},
		},
	}
}

func dataSourceObjectStorageBucketRead(
	ctx context.Context,
	d *schema.ResourceData,
	m interface{},
) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*openapi.APIClient)
	bucketName := d.Get("name").(string)
	objectStorageIdOfBucket := d.Get("object_storage_id").(string)

	diags, objectStorage, s3Credentials := getObjectStorageAndCredentials(diags, client, objectStorageIdOfBucket)

	diags, bucketInfo, err := getBucket(diags, objectStorage, s3Credentials, bucketName)
	if err != nil {
		return diag.FromErr(err)
	}
	isPublicSharing := d.Get("public_sharing").(bool)
	publicSharingLink := d.Get("public_sharing_link").(string)

	bucket := Bucket{
		bucketName,
		objectStorageIdOfBucket,
		objectStorage.S3TenantId,
		bucketInfo.CreationDate,
		isPublicSharing,
		publicSharingLink,
	}
	return AddObjectStorageBucketToData(bucket, d, diags)
}
