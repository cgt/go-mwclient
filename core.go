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
)

// If you modify this package, please change the user agent.
const DefaultUserAgent = "go-mwclient (https://github.com/cgtdk/go-mwclient) by meta:User:Cgtdk"

type (
	Client struct {
		httpc             *http.Client
		cjar              *cookiejar.Jar
		ApiUrl            *url.URL
		format, UserAgent string
		Tokens            map[string]string
		Maxlag            Maxlag
	}

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
func New(inUrl string, maxlagOn bool, maxlagTimeout string, maxlagRetries int) *Client {
	cjar, _ := cookiejar.New(nil)
	apiurl, err := url.Parse(inUrl)
	if err != nil {
		panic(err) // Yes, this is bad, but so is using bad URLs and I don't want two return values.
	}

	return &Client{
		httpc:     &http.Client{nil, nil, cjar},
		cjar:      cjar,
		ApiUrl:    apiurl,
		format:    "json",
		UserAgent: DefaultUserAgent,
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
func NewDefault(inUrl string) *Client {
	return New(inUrl, false, "-1", 0)
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
			req, err = http.NewRequest(httpMethod, w.ApiUrl.String(), strings.NewReader(URLEncode(params)))
		} else {
			req, err = http.NewRequest(httpMethod, fmt.Sprintf("%s?%s", w.ApiUrl.String(), URLEncode(params)), nil)
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
		for _, cookie := range w.cjar.Cookies(w.ApiUrl) {
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
			num, _ := strconv.Atoi(resp.Header.Get("Retry-After"))
			return nil, MaxLagError{
				string(body),
				num,
			}
		}

		js, err := simplejson.NewJson(body)
		if err != nil {
			log.Printf("Error during JSON parsing: %s\n", err)
			return nil, err
		}

		return js, nil
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

		return nil, fmt.Errorf("The API is busy. Tried to perform request %d times unsuccessfully.", w.Maxlag.Retries)
	}

	// If maxlag is not enabled, just do the request regularly.
	return callf()
}

// Get wraps the w.call method to make it do a GET request.
func (w *Client) Get(params url.Values) (*simplejson.Json, error) {
	return w.call(params, false)
}

// GetCheck wraps the w.call method to make it do a GET request
// and checks for API errors/warnings using the ErrorCheck function.
// The returned boolean will be true if no API errors or warnings are found.
func (w *Client) GetCheck(params url.Values) (*simplejson.Json, error, bool) {
	return ErrorCheck(w.call(params, false))
}

// Post wraps the w.call method to make it do a POST request.
func (w *Client) Post(params url.Values) (*simplejson.Json, error) {
	return w.call(params, true)
}

// PostCheck wraps the w.call method to make it do a POST request
// and checks for API errors/warnings using the ErrorCheck function.
// The returned boolean will be true if no API errors or warnings are found.
func (w *Client) PostCheck(params url.Values) (*simplejson.Json, error, bool) {
	return ErrorCheck(w.call(params, true))
}

// ErrorCheck checks for API errors and warnings, and returns false as its third
// return value if any are found. Otherwise it returns true.
// ErrorCheck does not modify the json and err parameters, but merely passes them through,
// so it can be used to wrap the Post and Get methods.
func ErrorCheck(json *simplejson.Json, err error) (*simplejson.Json, error, bool) {
	apiok := true

	if _, ok := json.CheckGet("error"); ok {
		apiok = false
	}

	if _, ok := json.CheckGet("warnings"); ok {
		apiok = false
	}

	return json, err, apiok
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
			} else {
				return errors.New(lgResult)
			}
		}

		return nil
	}

	return loginFunc("")
}

// Logout logs out. It does not take into account whether or not a user is actually
// logged in (because it is irrelevant). Always returns true.
func (w *Client) Logout() bool {
	w.Get(url.Values{"action": {"logout"}})
	return true
}
