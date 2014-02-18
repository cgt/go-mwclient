// Package mwclient provides functionality for interacting with the MediaWiki API.
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
)

// If you modify this package, please change the user agent.
const DefaultUserAgent = "go-mwclient (https://github.com/cgt/go-mwclient)"

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

// New returns a pointer to an initialized Client object. If the provided API URL
// is invalid (as defined by the net/url package), then it will panic with the
// error from url.Parse(). New disables maxlag by default. To enable it,
// simply set Client.Maxlag.On to true.
// The default timeout is 5 seconds and the default amount of retries is 5.
func New(inURL, userAgent string) (*Client, error) {
	cjar, _ := cookiejar.New(nil)
	apiurl, err := url.Parse(inURL)
	if err != nil {
		return nil, err
	}

	if userAgent == "" || userAgent == " " {
		return nil, fmt.Errorf("userAgent parameter empty")
	}

	return &Client{
		httpc:     &http.Client{nil, nil, cjar},
		cjar:      cjar,
		APIURL:    apiurl,
		format:    "json",
		UserAgent: fmt.Sprintf("%s (%s)", userAgent, DefaultUserAgent),
		Tokens:    map[string]string{},
		Maxlag: Maxlag{
			On:      false,
			Timeout: "5",
			Retries: 3,
		},
	}, nil
}

// call makes a GET or POST request to the Mediawiki API (depending on whether
// the post argument is true or false (if true, it will POST) and returns the
// JSON response as a []byte.
// call supports the maxlag parameter and will respect it if it is turned on
// in the Client it operates on.
func (w *Client) call(params url.Values, post bool) ([]byte, error) {
	// The main functionality in this method is in a closure to simplify maxlag handling.
	callf := func() ([]byte, error) {
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
			req, err = http.NewRequest(httpMethod, w.APIURL.String(), strings.NewReader(urlEncode(params)))
		} else {
			req, err = http.NewRequest(httpMethod, fmt.Sprintf("%s?%s", w.APIURL.String(), urlEncode(params)), nil)
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

		// Store any new cookies
		w.cjar.SetCookies(req.URL, resp.Cookies())

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Error reading from resp.Body: %s\n", err)
			return nil, err
		}

		// Handle maxlag
		if resp.Header.Get("X-Database-Lag") != "" {
			retryAfter, _ := strconv.Atoi(resp.Header.Get("Retry-After"))
			return nil, maxLagError{
				string(body),
				retryAfter,
			}
		}

		return body, nil

	}

	if w.Maxlag.On {
		for tries := 0; tries < w.Maxlag.Retries; tries++ {
			reqResp, err := callf()

			// Logic for handling maxlag errors. If err is nil or a different error,
			// they are passed through in the else.
			if lagerr, ok := err.(maxLagError); ok {
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

// callJSON wraps the call method and encodes the JSON response
// as a *simplejson.Json object. Furthermore, any API errors/warnings are
// extracted and returned as the error return value (unless an error occurs
// during the API call or the parsing of the JSON response, in which case that
// error will be returned and the *simplejson.Json return value will be nil).
func (w *Client) callJSON(params url.Values, post bool) (*simplejson.Json, error) {
	body, err := w.call(params, post)
	if err != nil {
		return nil, err
	}

	js, err := simplejson.NewJson(body)
	if err != nil {
		return nil, err
	}

	return extractAPIErrors(js, err)
}

// Get performs a GET request with the specified parameters and returns the
// response as a *simplejson.Json object.
// Get will return any API errors and/or warnings (if no other errors occur)
// as the error return value.
func (w *Client) Get(params url.Values) (*simplejson.Json, error) {
	return w.callJSON(params, false)
}

// GetRaw performs a GET request with the specified parameters
// and returns the raw JSON response as a []byte.
// Unlike Get, GetRaw does not check for API errors/warnings.
// GetRaw is useful when you want to decode the JSON into a struct for easier
// and safer use.
func (w *Client) GetRaw(params url.Values) ([]byte, error) {
	return w.call(params, false)
}

// Post performs a POST request with the specified parameters and returns the
// response as a *simplejson.Json object.
// Post will return any API errors and/or warnings (if no other errors occur)
// as the error return value.
func (w *Client) Post(params url.Values) (*simplejson.Json, error) {
	return w.callJSON(params, true)
}

// PostRaw performs a POST request with the specified parameters
// and returns the raw JSON response as a []byte.
// Unlike Post, PostRaw does not check for API errors/warnings.
// PostRaw is useful when you want to decode the JSON into a struct for easier
// and safer use.
func (w *Client) PostRaw(params url.Values) ([]byte, error) {
	return w.call(params, false)
}

// Login attempts to login using the provided username and password.
func (w *Client) Login(username, password string) error {

	// By using a closure, we avoid requiring the public Login method to have
	// a token parameter while also avoiding repeating ourselves.
	// loginFunc must be predefined because it calls itself.
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

// Logout logs out. It does not take into account whether or not a user is actually logged in.
func (w *Client) Logout() {
	w.Get(url.Values{"action": {"logout"}})
}
