JAVAOPT = '-Dio.swagger.parser.util.RemoteUrl.trustAll=true -Dio.swagger.v3.parser.util.RemoteUrl.trustAll=true'
ifndef VERSION
	VERSION = 'v0.0.1'
endif
ifndef OUTPUTLOCATION
	OUTPUTLOCATION = /local/openapi/
endif
ifndef OPENAPIURL
	OPENAPIURL = https://api.contabo.com/api-v1.yaml
endif

ifndef OPENAPIVOLUME
	OPENAPIVOLUME = "$(CURDIR):/local"
endif

.PHONY: test-acc
test-acc:
	TF_ACC=1 go test -v ./...

.PHONY: build
build: generate-api-clients build-only

.PHONY: generate-api-clients
generate-api-clients:
	rm -rf openapi
	-docker volume rm -f openapivolume
	docker run --rm -v $(OPENAPIVOLUME) --env JAVA_OPTS=$(JAVAOPT) openapitools/openapi-generator-cli:v5.2.1 generate \
	--skip-validate-spec \
	--input-spec $(OPENAPIURL) \
	--generator-name  go \
	--output $(OUTPUTLOCATION)
	docker container create --name generate-openapi-client2 -v openapivolume:/openapi alpine bash
	docker cp generate-openapi-client2:/openapi/ .
	docker rm generate-openapi-client2

.PHONY: build-only
build-only:
	go mod tidy
	go mod download
	go build -o terraform-provider-contabo_$(VERSION)

.PHONY: build-only-debug
build-only-debug:
	go mod tidy
	go mod download
	go build -gcflags="all=-N -l"  -o terraform-provider-contabo_$(VERSION)

.PHONY: doc-preview
doc-preview:
	@echo "Preview your markdown documentation on this page: https://registry.terraform.io/tools/doc-preview"
