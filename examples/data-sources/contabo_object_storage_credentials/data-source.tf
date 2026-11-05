data "contabo_object_storage_credentials" "eu" {
  object_storage_id = contabo_object_storage.eu.id
}

# For example for an S3 backend or the aws provider.
output "s3_access_key" {
  value = data.contabo_object_storage_credentials.eu.access_key
}

output "s3_secret_key" {
  value     = data.contabo_object_storage_credentials.eu.secret_key
  sensitive = true
}
