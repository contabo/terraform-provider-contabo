package contabo

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"contabo.com/openapi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	uuid "github.com/satori/go.uuid"
)

type S3Credentials struct {
	AccessKey string
	SecretKey string
}

type Bucket struct {
	Name              string
	ObjectStorageId   string
	S3TenantId        string
	CreationDate      time.Time
	IsPublicSharing   bool
	PublicSharingLink string
}

const AWS_S3_STATEMENT_RESOURCE_PREFIX = "arn:aws:s3:::"

const ACCESS_FORBIDEN_BUCKET_POLICY_JSON_OBJECT = `{"Id":"CntbPolicy","Version":"2012-10-17","Statement":[]}`

func resourceObjectStorageBucket() *schema.Resource {
	return &schema.Resource{
		Description:   "Manage buckets on your contabo Object Storage. With this resource you are able to manage your buckets the same way your are able to manage them in your contabo customer panel.",
		CreateContext: resourceObjectStorageBucketCreate,
		ReadContext:   resourceObjectStorageBucketRead,
		UpdateContext: resourceObjectStorageBucketUpdate,
		DeleteContext: resourceObjectStorageBucketDelete,
		Schema: map[string]*schema.Schema{
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The identifier of the Object Storage. Use it to manage it!",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of your bucket, consider the naming restriction https://docs.aws.amazon.com/awscloudtrail/latest/userguide/cloudtrail-s3-bucket-naming-requirements.html.",
			},
			"object_storage_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The contabo objectStorageId on which the bucket should be created.",
			},
			"public_sharing": {
				Type:        schema.TypeBool,
				Default:     false,
				Optional:    true,
				Description: "Choose the access to your bucket. You can not share it at all or share it publicly.",
			},
			"public_sharing_link": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "If your bucket is publicly shared, you can access it with this link.",
			},
			"creation_date": {
				Type:        schema.TypeString,
				Description: "The creation date of the bucket.",
				Computed:    true,
			},
		},
	}
}

func resourceObjectStorageBucketCreate(
	ctx context.Context,
	data *schema.ResourceData,
	m interface{},
) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*openapi.APIClient)
	bucketName := data.Get("name").(string)
	objectStorageIdOfBucket := data.Get("object_storage_id").(string)
	isPublicSharing := data.Get("public_sharing").(bool)

	diags, objectStorage, s3Credentials := getObjectStorageAndCredentials(diags, client, objectStorageIdOfBucket)

	diags, err := createBucket(diags, objectStorage, s3Credentials, bucketName)
	if err != nil {
		if strings.Contains(err.Error(), "invalid characters") {
			return diag.FromErr(errors.New("could not create bucket. Name may contain unaccepted characters. See https://docs.aws.amazon.com/awscloudtrail/latest/userguide/cloudtrail-s3-bucket-naming-requirements.html for more info"))
		} else {
			return diag.FromErr(err)
		}
	}

	var publicSharingLink string

	if isPublicSharing {
		diags, publicSharingLink, err = enablePublicSharing(diags, objectStorage, s3Credentials, bucketName)
		if err != nil {
			return diag.FromErr(err)
		}
	} else {
		publicSharingLink = ""
	}

	bucket := Bucket{
		bucketName,
		objectStorageIdOfBucket,
		objectStorage.S3TenantId,
		time.Now(),
		isPublicSharing,
		publicSharingLink,
	}

	id := generateBucketId(objectStorage.ObjectStorageId, bucket.Name)
	data.SetId(id)
	return AddObjectStorageBucketToData(bucket, data, diags)
}

func enablePublicSharing(diags diag.Diagnostics,
	objectStorage openapi.ObjectStorageResponse,
	s3Credentials S3Credentials,
	bucketName string) (diag.Diagnostics, string, error) {

	diags, s3Url := getS3Url(diags, objectStorage)

	objectPolicyResourcePath := AWS_S3_STATEMENT_RESOURCE_PREFIX + bucketName + "/*"

	bucketPolicyJsonObject := fmt.Sprintf(`{"Version": "2012-10-17","Statement": [{"Action": ["s3:GetObject"],"Effect": "Allow","Principal": {"AWS": ["*"]},"Resource": ["%v"],"Sid": ""}]}`, objectPolicyResourcePath)

	minioClient, err := minio.New(s3Url.Host, &minio.Options{
		Creds: credentials.NewStaticV4(
			s3Credentials.AccessKey,
			s3Credentials.SecretKey,
			"",
		),
		Secure: true,
	})
	if err != nil {
		return diags, "", err
	}

	err = minioClient.SetBucketPolicy(
		context.Background(),
		bucketName,
		bucketPolicyJsonObject,
	)
	if err != nil {
		return diags, "", err
	}
	publicSharingUrl := generatePublicSharingUrl(objectStorage, bucketName)
	return diags, publicSharingUrl, nil
}

func generatePublicSharingUrl(objectStorage openapi.ObjectStorageResponse,
	bucketName string) string {
	return objectStorage.S3Url + "/" + objectStorage.S3TenantId + ":" + bucketName
}

func createBucket(diags diag.Diagnostics, objectStorage openapi.ObjectStorageResponse, s3Credentials S3Credentials, bucketName string) (diag.Diagnostics, error) {
	diags, s3Url := getS3Url(diags, objectStorage)

	minoClient, err := minio.New(s3Url.Host, &minio.Options{
		Creds: credentials.NewStaticV4(
			s3Credentials.AccessKey,
			s3Credentials.SecretKey,
			"",
		),
		Secure: true,
	})
	if err != nil {
		return diags, err
	}

	err = minoClient.MakeBucket(
		context.Background(),
		bucketName,
		minio.MakeBucketOptions{},
	)
	if err != nil {
		return diags, err
	}
	return diags, nil
}

func getObjectStorage(diags diag.Diagnostics, client *openapi.APIClient, objectStorageId string) (diag.Diagnostics, openapi.ObjectStorageResponse) {
	ApiRetrieveObjectStorageRequest := client.
		ObjectStoragesApi.RetrieveObjectStorage(context.Background(), objectStorageId).
		XRequestId(uuid.NewV4().String())

	objectStorageRetrieveResponse, httpResp, err := ApiRetrieveObjectStorageRequest.Execute()

	if err != nil {
		return HandleResponseErrors(diags, httpResp), openapi.ObjectStorageResponse{}
	}
	if len(objectStorageRetrieveResponse.Data) == 0 {
		return diag.FromErr(errors.New("No Object Storage could be found with id : " + objectStorageId)), objectStorageRetrieveResponse.Data[0]

	}
	return diags, objectStorageRetrieveResponse.Data[0]
}

func getCredentials(diags diag.Diagnostics, client *openapi.APIClient, objectStorageIdOfBucket string) (diag.Diagnostics, S3Credentials) {
	retrieveCredentialResponse, httpResp, err := client.UsersApi.
		ListObjectStorageCredentials(context.Background(), userId).
		XRequestId(uuid.NewV4().String()).
		ObjectStorageId(objectStorageIdOfBucket).
		Execute()

	if err != nil {
		return HandleResponseErrors(diags, httpResp), S3Credentials{}
	}

	if len(retrieveCredentialResponse.Data) != 1 {
		return diag.FromErr(errors.New("No credentials found for user id : " + userId + " on ObjectStorage: " + objectStorageIdOfBucket)), S3Credentials{}
	}

	return diags, S3Credentials{
		retrieveCredentialResponse.Data[0].AccessKey, retrieveCredentialResponse.Data[0].SecretKey,
	}
}

func resourceObjectStorageBucketRead(
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

func getObjectStorageAndCredentials(
	diags diag.Diagnostics,
	client *openapi.APIClient,
	objectStorageIdOfBucket string) (diag.Diagnostics, openapi.ObjectStorageResponse, S3Credentials) {
	diags, objectStorage := getObjectStorage(diags, client, objectStorageIdOfBucket)
	diags, s3Credentials := getCredentials(diags, client, objectStorageIdOfBucket)
	return diags, objectStorage, s3Credentials
}

func getBucket(diags diag.Diagnostics,
	objectStorage openapi.ObjectStorageResponse,
	s3Credentials S3Credentials,
	bucketName string) (diag.Diagnostics, minio.BucketInfo, error) {

	diags, s3Url := getS3Url(diags, objectStorage)

	minioClient, err := minio.New(s3Url.Host, &minio.Options{
		Creds: credentials.NewStaticV4(
			s3Credentials.AccessKey,
			s3Credentials.SecretKey,
			"",
		),
		Secure: true,
	})
	if err != nil {
		return diags, minio.BucketInfo{}, err
	}

	buckets, err := minioClient.ListBuckets(context.Background())
	if err != nil {
		return diags, minio.BucketInfo{}, err
	}

	for _, bucket := range buckets {
		if bucket.Name == bucketName {
			return diags, bucket, nil
		}
	}
	return diags, minio.BucketInfo{}, nil
}

func getS3Url(
	diags diag.Diagnostics,
	objectStorage openapi.ObjectStorageResponse) (diag.Diagnostics, url.URL) {
	s3Url, err := url.Parse(objectStorage.S3Url)
	if err != nil {
		return diag.FromErr((err)), url.URL{}
	}
	return diags, *s3Url
}

func resourceObjectStorageBucketUpdate(
	ctx context.Context,
	data *schema.ResourceData,
	m interface{},
) diag.Diagnostics {

	var diags diag.Diagnostics
	client := m.(*openapi.APIClient)
	resourceFileBucketName := data.Get("name").(string)
	bucketName := getBucketNameFromId(data.Id())
	diags = checkIfBucketNameChanged(diags, bucketName, resourceFileBucketName)

	objectStorageIdOfBucket := getObjectStorageIdFromId(data.Id())
	resourceFileobjectStorageId := data.Get("object_storage_id").(string)
	diags = checkIfObjectStorageIdChanged(diags, objectStorageIdOfBucket, resourceFileobjectStorageId)

	diags, objectStorage, s3Credentials := getObjectStorageAndCredentials(diags, client, objectStorageIdOfBucket)

	diags, bucketInfo, err := getBucket(diags, objectStorage, s3Credentials, bucketName)
	if err != nil {
		return diag.FromErr(err)
	}

	var publicSharingLink string
	isPublicSharing := false

	if data.HasChange("public_sharing") {
		isPublicSharing = data.Get("public_sharing").(bool)
	}

	if isPublicSharing {
		diags, publicSharingLink, err = enablePublicSharing(diags, objectStorage, s3Credentials, bucketName)
		if err != nil {
			return diag.FromErr(err)
		}
	} else {
		diags = disablePublicSharing(diags, objectStorage, s3Credentials, bucketName)
		publicSharingLink = ""
	}

	bucket := Bucket{
		bucketName,
		objectStorageIdOfBucket,
		objectStorage.S3TenantId,
		bucketInfo.CreationDate,
		isPublicSharing,
		publicSharingLink,
	}
	return AddObjectStorageBucketToData(bucket, data, diags)
}

func disablePublicSharing(
	diags diag.Diagnostics,
	objectStorage openapi.ObjectStorageResponse,
	s3Credentials S3Credentials,
	bucketName string) diag.Diagnostics {
	diags, s3Url := getS3Url(diags, objectStorage)

	client, err := minio.New(s3Url.Host, &minio.Options{
		Creds: credentials.NewStaticV4(
			s3Credentials.AccessKey,
			s3Credentials.SecretKey,
			"",
		),
		Secure: true,
	})
	if err != nil {
		return diag.FromErr(err)
	}

	err = client.SetBucketPolicy(
		context.Background(),
		bucketName,
		ACCESS_FORBIDEN_BUCKET_POLICY_JSON_OBJECT,
	)
	if err != nil {
		return diag.FromErr(err)
	}
	return diags

}

func resourceObjectStorageBucketDelete(
	ctx context.Context,
	d *schema.ResourceData,
	m interface{},
) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*openapi.APIClient)
	bucketName := d.Get("name").(string)
	objectStorageIdOfBucket := d.Get("object_storage_id").(string)

	diags, objectStorage, s3Credentials := getObjectStorageAndCredentials(diags, client, objectStorageIdOfBucket)
	diags = deleteBucket(diags, objectStorage, s3Credentials, bucketName)

	d.SetId("")
	return diags
}

func deleteBucket(diags diag.Diagnostics, objectStorage openapi.ObjectStorageResponse, s3Credentials S3Credentials, bucketName string) diag.Diagnostics {
	diags, s3Url := getS3Url(diags, objectStorage)

	minioClient, err := minio.New(s3Url.Host, &minio.Options{
		Creds: credentials.NewStaticV4(
			s3Credentials.AccessKey,
			s3Credentials.SecretKey,
			"",
		),
		Secure: true,
	})
	if err != nil {
		return diag.FromErr(err)
	}
	err = minioClient.RemoveBucket(context.Background(), bucketName)
	if err != nil {
		return diag.FromErr(err)
	}
	return diags
}

func AddObjectStorageBucketToData(
	bucket Bucket,
	d *schema.ResourceData,
	diags diag.Diagnostics,
) diag.Diagnostics {
	id := generateBucketId(bucket.ObjectStorageId, bucket.Name)

	if err := d.Set("id", id); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("name", bucket.Name); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("object_storage_id", bucket.ObjectStorageId); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("creation_date", bucket.CreationDate.String()); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("public_sharing", bucket.IsPublicSharing); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("public_sharing_link", bucket.PublicSharingLink); err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func generateBucketId(objectStorageId string, bucketName string) string {
	return objectStorageId + "/" + bucketName
}


func getBucketNameFromId(bucketId string) string {
	bucketName := strings.Split(bucketId, "/")
	return bucketName[1]
}

func getObjectStorageIdFromId(bucketId string) string {
	bucketName := strings.Split(bucketId, "/")
	return bucketName[0]
}

func checkIfBucketNameChanged(diags diag.Diagnostics, bucketName string, fileResourceBucketName string) diag.Diagnostics {
	if(bucketName != fileResourceBucketName) {
		return diag.FromErr(errors.New("it is not possible to update the bucket name, please create instead a new bucket"))
	}
	return diags
}

func checkIfObjectStorageIdChanged(diags diag.Diagnostics, bucketName string, fileResourceBucketName string) diag.Diagnostics {
	if(bucketName != fileResourceBucketName) {
		return diag.FromErr(errors.New("it is not possible to update the object storage id"))
	}
	return diags
}