package contabo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
)

type ApiError struct {
	StatusCode uint16 `json:"statusCode"`
	Message    string `json:"message"`
}

// HandleResponseErrors turns a failed API response into an error diagnostic
// that names the request, the HTTP status and the API's error message.
func HandleResponseErrors(
	diags diag.Diagnostics,
	httpResp *http.Response,
) diag.Diagnostics {
	return append(diags, apiErrorDiagnostic(httpResp, nil))
}

// apiErrorDiags is HandleResponseErrors for call sites that also have the
// error returned by the API client. Without an HTTP response (DNS failure,
// timeout, TLS error) the error is the only detail there is.
func apiErrorDiags(httpResp *http.Response, err error) diag.Diagnostics {
	return diag.Diagnostics{apiErrorDiagnostic(httpResp, err)}
}

func apiErrorDiagnostic(httpResp *http.Response, err error) diag.Diagnostic {
	if httpResp == nil {
		detail := "The Contabo API request failed without an HTTP response."
		if err != nil {
			detail = fmt.Sprintf("The Contabo API request failed without an HTTP response: %s", err)
		}
		return diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Unexpected API error, no http response",
			Detail:   detail,
		}
	}

	request := ""
	if httpResp.Request != nil && httpResp.Request.URL != nil {
		request = fmt.Sprintf("%s %s: ", httpResp.Request.Method, httpResp.Request.URL.Path)
	}

	return diag.Diagnostic{
		Severity: diag.Error,
		Summary:  fmt.Sprintf("API error, status code: %d", httpResp.StatusCode),
		Detail: fmt.Sprintf(
			"%sAPI error, status code: %d, details: %s",
			request, httpResp.StatusCode, apiErrorMessage(httpResp, err),
		),
	}
}

// apiErrorMessage returns the message of a Contabo API error body, the raw
// body when it is not the usual error object, or the client error.
func apiErrorMessage(httpResp *http.Response, err error) string {
	body := readResponseBody(httpResp)
	var apiError ApiError
	if jsonErr := json.Unmarshal(body, &apiError); jsonErr == nil && apiError.Message != "" {
		return apiError.Message
	}
	if text := strings.TrimSpace(string(body)); text != "" {
		return text
	}
	if err != nil {
		return err.Error()
	}
	return http.StatusText(httpResp.StatusCode)
}

// readResponseBody reads the body and puts it back, so that the response can
// be inspected more than once. The generated API client has already read the
// body into memory, so this never touches the network.
func readResponseBody(httpResp *http.Response) []byte {
	if httpResp == nil || httpResp.Body == nil {
		return nil
	}
	body, _ := io.ReadAll(httpResp.Body)
	_ = httpResp.Body.Close()
	httpResp.Body = io.NopCloser(bytes.NewReader(body))
	return body
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
