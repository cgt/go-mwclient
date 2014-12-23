package mwclient

import (
	"errors"
	"fmt"
	"strings"

	"cgt.name/pkg/multierror"
	"github.com/bitly/go-simplejson"
)

// APIError represents a generic API error described by an error code
// and a string containing information about the error.
type APIError struct {
	Code, Info string
}

func (e APIError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Info)
}

// APIWarning represents a generic API warning described by the name of the module
// from which the warning originates and a string containing information about the warning.
type APIWarning struct {
	Module, Info string
}

func (e APIWarning) Error() string {
	return fmt.Sprintf("%s: %s", e.Module, e.Info)
}

// captchaError represents the error returned by the API when it requires the client
// to solve a CAPTCHA to perform the action requested.
type captchaError struct {
	Type string `json:"type"`
	Mime string `json:"mime"`
	ID   string `json:"id"`
	URL  string `json:"url"`
}

func (e captchaError) Error() string {
	return fmt.Sprintf("API requires solving a CAPTCHA of type %s (%s) with ID %s at URL %s", e.Type, e.Mime, e.ID, e.URL)
}

// maxLagError is returned by the callf closure in the Client.call method when there is too much
// lag on the MediaWiki site. maxLagError contains a message from the server in the format
// "Waiting for $host: $lag seconds lagged\n" and an integer specifying how many seconds to wait
// before trying the request again.
type maxLagError struct {
	Message string
	Wait    int
}

func (e maxLagError) Error() string {
	return e.Message
}

// ErrAPIBusy is the error returned by an API call function when maxlag is
// enabled, and the API responds that it is busy for each of the in
// Client.Maxlag.Retries specified amount of retries.
var ErrAPIBusy = errors.New("the API is too busy. Try again later")

// ErrNoArgs is returned by API call methods that take variadic arguments when
// no arguments are passed.
var ErrNoArgs = errors.New("no arguments passed")

// extractAPIErrors extracts API errors and warnings from a given *simplejson.Json object
// and returns them together in a multierror.Multierror object.
func extractAPIErrors(json *simplejson.Json) (*simplejson.Json, error) {
	// Check if there are any errors or warnings
	var isAPIErrors, isAPIWarnings bool
	if _, ok := json.CheckGet("error"); ok {
		isAPIErrors = true
	}
	if _, ok := json.CheckGet("warnings"); ok {
		isAPIWarnings = true
	}
	// If there are no errors or warnings, return with nil error.
	if !isAPIErrors && !isAPIWarnings {
		return json, nil
	}

	// There are errors/warnings, extract and return them.
	var apiErrors multierror.Errors
	if isAPIErrors {
		// Extract error code
		errorCode, err := json.GetPath("error", "code").String()
		if err != nil {
			return json, fmt.Errorf("unable to assert error code field to type string")
		}

		// Extract error info
		errorInfo, err := json.GetPath("error", "info").String()
		if err != nil {
			return json, fmt.Errorf("unable to assert error info field to type string")
		}

		apiErrors = append(apiErrors, APIError{errorCode, errorInfo})
	}

	if isAPIWarnings {
		// Extract warnings
		warningsMap, err := json.Get("warnings").Map()
		if err != nil {
			return nil, fmt.Errorf("unable to assert 'warnings' field to type map[string]interface{}\n")
		}

		for k, v := range warningsMap {
			warning := v.(map[string]interface{})["*"]

			if strings.Contains(warning.(string), "\n") {
				// There can be multiple warnings in one warning info field.
				// If so, they are separated by a newline.
				// Split the warning string into two warnings and add them separately.
				for _, warn := range strings.Split(warning.(string), "\n") {
					apiErrors = append(apiErrors, APIWarning{k, warn})
				}
			} else {
				apiErrors = append(apiErrors, APIWarning{k, warning.(string)})
			}
		}
	}

	return json, apiErrors.Err()
}
