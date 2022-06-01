# terraform-provider-contabo

`terraform-provider-contabo` is a [terraform](https://www.terraform.io/) provider for managing your products from [Contabo](https://contabo.com) like Cloud VPS, VDS and S3 compatible Object Storage using the [Contabo APIs](https://api.contabo.com/) via terraform cli.

## Getting Started

* [Link to terraform page](https://registry.terraform.io/providers/contabo/contabo/latest)
* [Documentation link to terraform page](https://registry.terraform.io/providers/contabo/contabo/latest/docs)

1. Install [terraform cli](https://learn.hashicorp.com/tutorials/terraform/install-cli)
2. Copy the example `examples/main.tf.example` as `.tf` file to you project directory
3. Run terraform

    ```sh
    terraform init
    terraform plan
    # CAUTION:  with example main.tf you are about to order and pay an object storage
    terraform apply
    ```

## Local Development

1. Install [terraform cli](https://learn.hashicorp.com/tutorials/terraform/install-cli)
2. `git clone https://github.com/contabo/terraform-provider-contabo.git`
3. `make build` in order to create provider binary
4. create `~/.terraformrc` with following content

    ```terraform
    provider_installation {

      dev_overrides {
        "contabo/contabo" = "/PATH/TO/YOUR/BINARY/BUILD"
      }

      direct {}
    }
    ```

5. Then change to the `examples` directory and copy `main.tf.example` to `main.tf` and fill in the provider config.
6. In the same directory execute

    ```sh
    terraform plan
    # CAUTION:  with example main.tf you are about to order and pay a Cloud VPS instance
    terraform apply
    ```

### Acceptance Testing

In order to run acceptance tests run:

```sh
make test-acc
```

**CAUTION**: running acceptance testing will work with actual resources which will usually cost money
