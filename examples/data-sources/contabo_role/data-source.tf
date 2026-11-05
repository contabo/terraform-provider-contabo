# Built-in role for read-only S3 access.
data "contabo_role" "s3_read" {
  name = "S3 Read"
}
