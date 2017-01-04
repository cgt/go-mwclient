package mwclient

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/antonholmquist/jason"
)

// APIError represents a MediaWiki API error.
type APIError struct {
	Code, Info string
}

func (e APIError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Info)
}

// APIWarnings represents a collection of MediaWiki API warnings.
type APIWarnings []struct {
	Module, Info string
}

func (w APIWarnings) Error() string {
	var buf bytes.Buffer

	amount := len(w) // amount of warnings
	if amount == 1 {
		buf.WriteString("1 warning: ")
	} else {
		buf.WriteString(fmt.Sprintf("%d warnings: ", len(w)))
	}

	for _, warn := range w {
		buf.WriteString(fmt.Sprintf("[%s: %s] ", warn.Module, warn.Info))
	}

	return buf.String()
}

// CaptchaError represents the error returned by the API when it requires the
// client to solve a CAPTCHA to perform the action requested.
type CaptchaError struct {
	Type     string `json:"type"`
	Mime     string `json:"mime"`
	ID       string `json:"id"`
	URL      string `json:"url"`
	Question string `json:"question"`
}

func (e CaptchaError) Error() string {
	if e.URL != "" {
		return fmt.Sprintf("API requires solving a CAPTCHA of type %s (%s) with ID %s at URL %s",
			e.Type, e.Mime, e.ID, e.URL)
	} else if e.Question != "" {
		return fmt.Sprintf("API requires solving a CAPTCHA of type %s (%s) with ID %s: %s",
			e.Type, e.Mime, e.ID, e.Question)
	} else {
		// Unknown CAPTCHA type
		return fmt.Sprintf("API requires solving a CAPTCHA of type %s (%s) with ID %s",
			e.Type, e.Mime, e.ID)
	}
}

// maxLagError is returned by the callf closure in the Client.call method when
// there is too much lag on the MediaWiki site. maxLagError contains a message
// from the server in the format "Waiting for $host: $lag seconds lagged\n" and
// an integer specifying how many seconds to wait before trying the request again.
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

// extractAPIErrors extracts API errors or warnings from a given
// *jason.Object. If it finds an error, it will return an APIError.
// Otherwise it will look for warnings, and if it finds any it will return
// it/them in an APIWarning.
// extractAPIErrors is not compatible with MWAPI formatversion=1.
func extractAPIErrors(resp *jason.Object) error {
	if e, err := resp.GetObject("error"); err == nil {
		code, err1 := e.GetString("code")
		info, err2 := e.GetString("info")
		if !(err1 == nil && err2 == nil) {
			return fmt.Errorf("extractAPIErrors: 'error' object does not contain expected 'code' and 'info': %v", e)
		}
		return APIError{
			Code: code,
			Info: info,
		}
	}

	if w, err := resp.GetObject("warnings"); err == nil {
		return extractWarnings(w)
	}

	return nil
}

func extractWarnings(resp *jason.Object) error {
	var warnings APIWarnings
	for module, warningValue := range resp.Map() {
		warning, err := warningValue.Object()
		if err != nil {
			return fmt.Errorf("extractWarnings: %v: %v", err, warningValue)
		}

		info, err := warning.GetString("warnings")
		if err != nil {
			return fmt.Errorf("extractWarnings: %v: %v", err, warning)
		}
		warnings = append(warnings, APIWarnings{{module, info}}...)
	}

	return warnings
}
