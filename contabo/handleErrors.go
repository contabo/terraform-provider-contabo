package contabo

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
)

type ApiError struct {
	StatusCode uint16 `json:"statusCode"`
	Message    string `json:"message"`
}

func HandleResponseErrors(
	diags diag.Diagnostics,
	httpResp *http.Response,
) diag.Diagnostics {
	var apiError ApiError
	var responseBody []byte
	var err error

	if httpResp == nil {
		return append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Unexpected API error, no http response",
			Detail:   "Unexpected API error, no http response",
		})
	}

	responseBody, err = io.ReadAll(httpResp.Body)
	if err != nil {
		log.Panic("Error while parsing response error")
	}

	err = json.Unmarshal(responseBody, &apiError)

	var errorMessage string
	if err != nil {
		errorMessage = err.Error() + string(responseBody)
	} else {
		errorMessage = apiError.Message
	}

	return append(diags, diag.Diagnostic{
		Severity: diag.Error,
		Summary:  fmt.Sprintf("API error, status code: %d", apiError.StatusCode),
		Detail: fmt.Sprintf(
			"API error, status code: %d, details: %s", apiError.StatusCode, errorMessage),
	})
}

func MultipleDataObjectsError(
	diags diag.Diagnostics,
) diag.Diagnostics {
	return append(diags, diag.Diagnostic{
		Severity: diag.Error,
		Summary:  "API response had multiple data objects.",
		Detail:   "The API response for a specific object contained multiple objects.",
	})
}

func NoDataError(
	diags diag.Diagnostics,
) diag.Diagnostics {
	return append(diags, diag.Diagnostic{
		Severity: diag.Error,
		Summary:  "API response returned empty data.",
		Detail:   "The API response returned empty data.",
	})
}

func HandleMissingDataObjectsFilters(
	diags diag.Diagnostics,
	summary string,
	details string,
) diag.Diagnostics {
	return append(diags, diag.Diagnostic{
		Severity: diag.Error,
		Summary:  summary,
		Detail:   details,
	})
}

func HandleDownloadErrors(
	diags diag.Diagnostics,
) diag.Diagnostics {
	return append(diags, diag.Diagnostic{
		Severity: diag.Error,
		Summary:  "Download error, check the url availability and retry",
		Detail:   "Download error, check the url availability and retry",
	})
}
