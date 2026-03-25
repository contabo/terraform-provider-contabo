package contabo

import (
	"context"
	"errors"
	"time"

	"contabo.com/openapi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	uuid "github.com/satori/go.uuid"
)

const maxNumberOfRetries = 30
const sleepInterval = 2000

func pollInstance(diags diag.Diagnostics,
	client *openapi.APIClient,
	instanceId int64,
) {
	numberOfRetries := 0
	for numberOfRetries < maxNumberOfRetries {
		if isInstanceRunning(diags, client, instanceId) {
			return
		}
		time.Sleep(sleepInterval * time.Millisecond)
		numberOfRetries += 1
	}

	err := errors.New("Polling instance has failed.")
	diag.FromErr(err)

}

func isInstanceRunning(
	diags diag.Diagnostics,
	client *openapi.APIClient,
	instanceId int64,
) bool {
	res, httpResp, err := client.InstancesApi.
		RetrieveInstance(context.Background(), instanceId).
		XRequestId(uuid.NewV4().String()).
		Execute()

	if err != nil {

		HandleResponseErrors(diags, httpResp)
		return false
	}

	if res.Data != nil && len(res.Data) == 1 {
		if res.Data[0].Status == "running" {
			return true
		}
	}

	return false
}
