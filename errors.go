package mwclient

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/bitly/go-simplejson"
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

// extractAPIErrors extracts API errors or warnings from a given
// *simplejson.Json object. If it finds an error, it will return an APIError.
// Otherwise it will look for warnings, and if it finds any it will return
// it/them in an APIWarning.
func extractAPIErrors(resp *simplejson.Json) error {
	if e, ok := resp.CheckGet("error"); ok { // Check for errors
		code, ok1 := e.CheckGet("code")
		info, ok2 := e.CheckGet("info")
		if !(ok1 && ok2) {
			return errors.New("'error' object in API response is broken and stupid")
		}
		return APIError{
			Code: code.MustString(),
			Info: info.MustString(),
		}
	} else if w, ok := resp.CheckGet("warnings"); ok { // Check for warnings
		warnings := APIWarnings{}

		wmap, err := w.Map()
		if err != nil {
			return errors.New("'warnings' object in API response is broken and stupid")
		}
		for module, v := range wmap {
			info := v.(map[string]interface{})["*"].(string)

			if strings.Contains(info, "\n") {
				// There can be multiple warnings in one warning info field.
				// If so, they are separated by a newline.
				// Split the warning string into two warnings and add them separately.
				for _, warn := range strings.Split(info, "\n") {
					warnings = append(warnings, APIWarnings{{Module: module, Info: warn}}...)
				}
			} else {
				warnings = append(warnings, APIWarnings{{module, info}}...)
			}
		}

		return warnings
	}

	return nil
}
