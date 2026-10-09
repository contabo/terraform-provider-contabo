package contabo

import (
	"context"
	"testing"
)

func TestIsPublicSharingPolicy(t *testing.T) {
	cases := []struct {
		name   string
		policy string
		want   bool
	}{
		{
			name:   "empty policy",
			policy: "",
			want:   false,
		},
		{
			name:   "policy set by enablePublicSharing",
			policy: `{"Version": "2012-10-17","Statement": [{"Action": ["s3:GetObject"],"Effect": "Allow","Principal": {"AWS": ["*"]},"Resource": ["arn:aws:s3:::my-bucket/*"],"Sid": ""}]}`,
			want:   true,
		},
		{
			name:   "policy with string values",
			policy: `{"Version":"2012-10-17","Statement":[{"Action":"s3:GetObject","Effect":"Allow","Principal":"*","Resource":"arn:aws:s3:::my-bucket/*"}]}`,
			want:   true,
		},
		{
			name:   "policy set by disablePublicSharing",
			policy: ACCESS_FORBIDEN_BUCKET_POLICY_JSON_OBJECT,
			want:   false,
		},
		{
			name:   "policy for another bucket",
			policy: `{"Version": "2012-10-17","Statement": [{"Action": ["s3:GetObject"],"Effect": "Allow","Principal": {"AWS": ["*"]},"Resource": ["arn:aws:s3:::other-bucket/*"],"Sid": ""}]}`,
			want:   false,
		},
		{
			name:   "deny statement",
			policy: `{"Version": "2012-10-17","Statement": [{"Action": ["s3:GetObject"],"Effect": "Deny","Principal": {"AWS": ["*"]},"Resource": ["arn:aws:s3:::my-bucket/*"],"Sid": ""}]}`,
			want:   false,
		},
		{
			name:   "invalid json",
			policy: `{`,
			want:   false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := isPublicSharingPolicy(c.policy, "my-bucket"); got != c.want {
				t.Errorf("isPublicSharingPolicy() = %v, want %v", got, c.want)
			}
		})
	}
}

func TestResourceObjectStorageBucketImportIdValidation(t *testing.T) {
	cases := []struct {
		id      string
		wantErr bool
	}{
		{id: "b3e7c2a1-1234-5678-9abc-def012345678/my-bucket", wantErr: false},
		{id: "my-bucket", wantErr: true},
		{id: "/my-bucket", wantErr: true},
		{id: "b3e7c2a1-1234-5678-9abc-def012345678/", wantErr: true},
		{id: "a/b/c", wantErr: true},
	}

	for _, c := range cases {
		t.Run(c.id, func(t *testing.T) {
			d := resourceObjectStorageBucket().TestResourceData()
			d.SetId(c.id)
			_, err := resourceObjectStorageBucketImport(context.Background(), d, nil)
			if (err != nil) != c.wantErr {
				t.Errorf("resourceObjectStorageBucketImport(%q) error = %v, wantErr %v", c.id, err, c.wantErr)
			}
		})
	}
}
