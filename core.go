// Package mwclient provides methods for interacting with the MediaWiki API.
package mwclient

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"
	"time"

	simplejson "github.com/bitly/go-simplejson"
	"github.com/joeshaw/multierror"
)

// If you modify this package, please change the user agent.
const DefaultUserAgent = "go-mwclient (https://github.com/cgtdk/go-mwclient)"

type (
	// Client represents the API client.
	Client struct {
		httpc             *http.Client
		cjar              *cookiejar.Jar
		APIURL            *url.URL
		format, UserAgent string
		Tokens            map[string]string
		Maxlag            Maxlag
	}

	// Maxlag contains maxlag configuration for Client.
	// See https://www.mediawiki.org/wiki/Manual:Maxlag_parameter
	Maxlag struct {
		On      bool   // If true, Client.Call will set the maxlag parameter.
		Timeout string // The maxlag parameter to send to the server.
		Retries int    // Specifies how many times to retry a request before returning with an error.
	}
)

// MaxLagError is returned by the callf closure in the Client.call method when there is too much
// lag on the MediaWiki site. MaxLagError contains a message from the server in the format
// "Waiting for $host: $lag seconds lagged\n" and an integer specifying how many seconds to wait
// before trying the request again.
type MaxLagError struct {
	Message string
	Wait    int
}

func (e MaxLagError) Error() string {
	return e.Message
}

// New returns an initialized Client object. If the provided API url is an
// invalid URL (as defined by the net/url package), then it will panic
// with the error from url.Parse().
func New(inURL, userAgent string, maxlagOn bool, maxlagTimeout string, maxlagRetries int) *Client {
	cjar, _ := cookiejar.New(nil)
	apiurl, err := url.Parse(inURL)
	if err != nil {
		panic(err) // Yes, this is bad, but so is using bad URLs and I don't want two return values.
	}

	var ua string // user agent
	if userAgent == "" || userAgent == " " {
		ua = fmt.Sprintf("Unidentified client (%s)", DefaultUserAgent)
	} else {
		ua = fmt.Sprintf("%s (%s)", userAgent, DefaultUserAgent)
	}

	return &Client{
		httpc:     &http.Client{nil, nil, cjar},
		cjar:      cjar,
		APIURL:    apiurl,
		format:    "json",
		UserAgent: ua,
		Tokens:    map[string]string{},
		Maxlag: Maxlag{
			On:      maxlagOn,
			Timeout: maxlagTimeout,
			Retries: maxlagRetries,
		},
	}
}

// NewDefault is a wrapper for New that passes nil as inMaxlag.
// NewDefault is meant for user clients (as opposed to bot clients); use New for bots.
func NewDefault(inURL, userAgent string) *Client {
	return New(inURL, userAgent, false, "-1", 0)
}

// call makes a GET or POST request to the Mediawiki API (depending on whether
// the post argument is true or false (if true, it will POST).
// call supports the maxlag parameter and will respect it if it is turned on
// in the Client it operates on.
func (w *Client) call(params url.Values, post bool) (*simplejson.Json, error) {
	callf := func() (*simplejson.Json, error) {
		params.Set("format", w.format)

		if w.Maxlag.On {
			if params.Get("maxlag") == "" {
				// User has not set maxlag param manually. Use configured value.
				params.Set("maxlag", w.Maxlag.Timeout)
			}
		}

		// Make a POST or GET request depending on the "post" parameter.
		var httpMethod string
		if post {
			httpMethod = "POST"
		} else {
			httpMethod = "GET"
		}

		var req *http.Request
		var err error
		if post {
			req, err = http.NewRequest(httpMethod, w.APIURL.String(), strings.NewReader(URLEncode(params)))
		} else {
			req, err = http.NewRequest(httpMethod, fmt.Sprintf("%s?%s", w.APIURL.String(), URLEncode(params)), nil)
		}
		if err != nil {
			log.Printf("Unable to make request: %s\n", err)
			return nil, err
		}

		// Set headers on request
		req.Header.Set("User-Agent", w.UserAgent)
		if post {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}

		// Set any old cookies on the request
		for _, cookie := range w.cjar.Cookies(w.APIURL) {
			req.AddCookie(cookie)
		}

		// Make the request
		resp, err := w.httpc.Do(req)
		defer resp.Body.Close()
		if err != nil {
			log.Printf("Error during %s: %s\n", httpMethod, err)
			return nil, err
		}

		// Set any new cookies
		w.cjar.SetCookies(req.URL, resp.Cookies())

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Error reading from resp.Body: %s\n", err)
			return nil, err
		}

		// Handle maxlag
		if resp.Header.Get("X-Database-Lag") != "" {
			retryAfter, _ := strconv.Atoi(resp.Header.Get("Retry-After"))
			return nil, MaxLagError{
				string(body),
				retryAfter,
			}
		}

		js, err := simplejson.NewJson(body)
		if err != nil {
			log.Printf("Error during JSON parsing: %s\n", err)
			return nil, err
		}

		return ExtractAPIErrors(js, err)
	}

	if w.Maxlag.On {
		for tries := 0; tries < w.Maxlag.Retries; tries++ {
			reqResp, err := callf()

			// Logic for handling maxlag errors. If err is nil or a different error,
			// they are passed through in the else.
			if lagerr, ok := err.(MaxLagError); ok {
				// If there are no tries left, don't wait needlessly.
				if tries < w.Maxlag.Retries-1 {
					time.Sleep(time.Duration(lagerr.Wait) * time.Second)
				}
				continue
			} else {
				return reqResp, err
			}
		}

		return nil, fmt.Errorf("the API is busy. Tried to perform request %d times unsuccessfully", w.Maxlag.Retries)
	}

	// If maxlag is not enabled, just do the request regularly.
	return callf()
}

// ExtractAPIErrors extracts API errors and warnings from a given *simplejson.Json object
// and returns them in a multierror.Errors object.
func ExtractAPIErrors(json *simplejson.Json, err error) (*simplejson.Json, error) {
	// This shouldn't happen, but just in case...
	if err != nil {
		return nil, err
	}

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
			return json, fmt.Errorf("API returned malformed response. Unable to assert error code field to type string")
		}

		// Extract error info
		errorInfo, err := json.GetPath("error", "info").String()
		if err != nil {
			return json, fmt.Errorf("API returned malformed response. Unable to assert error info field to type string")
		}

		apiErrors = append(apiErrors, fmt.Errorf("%s: %s", errorCode, errorInfo))
	}

	if isAPIWarnings {
		// Extract warnings
		for k, v := range json.Get("warnings").MustMap() {
			apiErrors = append(apiErrors, fmt.Errorf("%s: %s", k, v.(map[string]interface{})["*"]))
		}
	}

	return json, apiErrors.Err()
}

// Get wraps the w.call method to make it do a GET request.
func (w *Client) Get(params url.Values) (*simplejson.Json, error) {
	return w.call(params, false)
}

// Post wraps the w.call method to make it do a POST request.
func (w *Client) Post(params url.Values) (*simplejson.Json, error) {
	return w.call(params, true)
}

// Login attempts to login using the provided username and password.
func (w *Client) Login(username, password string) error {

	// By using a closure, we avoid requiring the public Login method to have a token parameter.
	var loginFunc func(token string) error

	loginFunc = func(token string) error {
		v := url.Values{
			"action":     {"login"},
			"lgname":     {username},
			"lgpassword": {password},
		}
		if token != "" {
			v.Set("lgtoken", token)
		}

		resp, err := w.Post(v)
		if err != nil {
			return err
		}

		if lgResult, _ := resp.Get("login").Get("result").String(); lgResult != "Success" {
			if lgResult == "NeedToken" {
				lgToken, _ := resp.Get("login").Get("token").String()
				return loginFunc(lgToken)
			}
			return errors.New(lgResult)
		}

		return nil
	}

	return loginFunc("")
}

// Logout logs out. It does not take into account whether or not a user is actually
// logged in (because it is irrelevant).
func (w *Client) Logout() {
	w.Get(url.Values{"action": {"logout"}})
}
