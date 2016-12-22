package mwclient

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"

	"cgt.name/pkg/go-mwclient/params"

	"github.com/antonholmquist/jason"
	"github.com/mrjones/oauth"
)

// If you modify this package, please change DefaultUserAgent.

// DefaultUserAgent is the HTTP User-Agent used by default.
const DefaultUserAgent = "go-mwclient (https://github.com/cgt/go-mwclient)"

type assertType uint8

// These consts are used as enums for the Client type's Assert field.
const (
	// AssertNone is used to disable API assertion
	AssertNone assertType = iota
	// AssertUser is used to assert that the client is logged in
	AssertUser
	// AssertBot is used to assert that the client is logged in as a bot
	AssertBot
)

type (
	// Client represents the API client.
	Client struct {
		httpc  *http.Client
		apiURL *url.URL
		// HTTP user agent
		UserAgent string
		// API token cache.
		// Maps from name of token (e.g., "csrf") to token value.
		// Use GetToken to obtain tokens.
		Tokens map[string]string
		// Maxlag contains maxlag configuration for Client.
		Maxlag Maxlag
		// If Assert is assigned the value of consts AssertUser or AssertBot,
		// the 'assert' parameter will be added to API requests with
		// the value 'user' or 'bot', respectively. To disable such assertions,
		// set Assert to AssertNone (set by default by New()).
		Assert assertType
		debug  io.Writer
	}

	// Maxlag contains maxlag configuration for Client.
	// See https://www.mediawiki.org/wiki/Manual:Maxlag_parameter
	Maxlag struct {
		// If true, API requests will set the maxlag parameter.
		On bool
		// The maxlag parameter to send to the server.
		Timeout string
		// Specifies how many times to retry a request before returning with an error.
		Retries int
		// sleep is used for mocking time.Sleep in tests to avoid prolonging
		// test execution needlessly by actually sleeping.
		sleep sleeper
	}
)

// SetDebug takes an io.Writer to which HTTP requests and responses
// made by Client will be dumped with httputil to as they are sent and
// received. To disable, set to nil (default).
func (w *Client) SetDebug(wr io.Writer) { w.debug = wr }

// SetHTTPTimeout overrides the default HTTP client timeout of 30 seconds.
// This is not related to the maxlag timeout.
func (w *Client) SetHTTPTimeout(timeout time.Duration) {
	w.httpc.Timeout = timeout
}

// sleeper is used for mocking time.Sleep.
type sleeper func(d time.Duration)

// New returns a pointer to an initialized Client object. If the provided API URL
// is invalid (as defined by the net/url package), then it will return nil and
// the error from url.Parse().
//
// The userAgent parameter will be joined with the DefaultUserAgent const and
// used as HTTP User-Agent. If userAgent is an empty string, DefaultUserAgent
// will be used by itself as User-Agent. The User-Agent set by New can be
// overriden by setting the UserAgent field on the returned *Client.
//
// New disables maxlag by default. To enable it, simply set
// Client.Maxlag.On to true. The default timeout is 5 seconds and the default
// amount of retries is 3.
func New(inURL, userAgent string) (*Client, error) {
	cjar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}

	apiurl, err := url.Parse(inURL)
	if err != nil {
		return nil, err
	}

	var ua string
	if userAgent != "" {
		ua = userAgent + " " + DefaultUserAgent
	} else {
		ua = DefaultUserAgent
	}

	return &Client{
		httpc: &http.Client{
			Transport:     nil,
			CheckRedirect: nil,
			Jar:           cjar,
			Timeout:       30 * time.Second,
		},
		apiURL:    apiurl,
		UserAgent: ua,
		Tokens:    map[string]string{},
		Maxlag: Maxlag{
			On:      false,
			Timeout: "5",
			Retries: 3,
			sleep:   time.Sleep,
		},
		Assert: AssertNone,
	}, nil
}

// call makes a GET or POST request to the Mediawiki API depending on whether
// the post argument is true or false (if true, it will POST) and returns
// the response body as an io.ReadCloser. Remember to close it when done with it.
// call supports the maxlag parameter and will respect it if it is turned on
// in the Client it operates on.
func (w *Client) call(p params.Values, post bool) (io.ReadCloser, error) {
	// The main functionality in this method is in a closure to simplify maxlag handling.
	callf := func() (io.ReadCloser, error) {
		p.Set("format", "json")
		if fmtver := p.Get("formatversion"); fmtver == "1" {
			p.Set("utf8", "")
		} else if fmtver == "" {
			p.Set("formatversion", "2")
			// utf8= is implicit in formatversion=2
		}

		if w.Maxlag.On {
			if p.Get("maxlag") == "" {
				// User has not set maxlag param manually. Use configured value.
				p.Set("maxlag", w.Maxlag.Timeout)
			}
		}

		if w.Assert > AssertNone {
			switch w.Assert {
			case AssertUser:
				p.Set("assert", "user")
			case AssertBot:
				p.Set("assert", "bot")
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
			req, err = http.NewRequest(httpMethod, w.apiURL.String(), strings.NewReader(p.Encode()))
		} else {
			req, err = http.NewRequest(httpMethod, fmt.Sprintf("%s?%s", w.apiURL.String(), p.Encode()), nil)
		}
		if err != nil {
			return nil, fmt.Errorf("unable to create HTTP request (method: %s, params: %v): %v",
				httpMethod, p, err)
		}

		// Set headers on request
		req.Header.Set("User-Agent", w.UserAgent)
		if post {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}

		if w.debug != nil {
			reqdump, err := httputil.DumpRequestOut(req, true)
			if err != nil {
				fmt.Fprintf(w.debug, "Err dumping request: %v\n", err)
			} else {
				w.debug.Write(reqdump)
			}
		}

		// Make the request
		resp, err := w.httpc.Do(req)
		if err != nil {
			return nil, fmt.Errorf("error occured during HTTP request: %v", err)
		}

		if w.debug != nil {
			respdump, err := httputil.DumpResponse(resp, true)
			if err != nil {
				fmt.Fprintf(w.debug, "Err dumping response: %v\n", err)
			} else {
				w.debug.Write(respdump)
			}
		}

		// Handle maxlag
		if resp.Header.Get("X-Database-Lag") != "" {
			defer resp.Body.Close()
			retryAfter, err := strconv.Atoi(resp.Header.Get("Retry-After"))
			if err != nil {
				return nil, err
			}

			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return nil, err
			}

			return nil, maxLagError{
				string(body),
				retryAfter,
			}
		}

		return resp.Body, nil
	}

	if w.Maxlag.On {
		for tries := 0; tries < w.Maxlag.Retries; tries++ {
			reqResp, err := callf()

			// Logic for handling maxlag errors. If err is nil or a different error,
			// they are passed through in the else.
			if lagerr, ok := err.(maxLagError); ok {
				// If there are no tries left, don't wait needlessly.
				if tries < w.Maxlag.Retries-1 {
					w.Maxlag.sleep(time.Duration(lagerr.Wait) * time.Second)
				}
				continue
			} else {
				return reqResp, err
			}
		}

		return nil, ErrAPIBusy
	}

	// If maxlag is not enabled, just do the request regularly.
	return callf()
}

// callJSON wraps the call method and encodes the JSON response
// as a *jason.Object. Furthermore, any API errors/warnings are
// extracted and returned as the error return value (unless an error occurs
// during the API call or the parsing of the JSON response, in which case that
// error will be returned and the *jason.Object return value will be nil).
func (w *Client) callJSON(p params.Values, post bool) (*jason.Object, error) {
	body, err := w.call(p, post)
	if err != nil {
		return nil, err
	}
	if body != nil {
		defer body.Close()
	}

	js, err := jason.NewObjectFromReader(body)
	if err != nil {
		return nil, err
	}

	return js, extractAPIErrors(js)
}

// callRaw wraps the call method and reads the response body into a []byte.
func (w *Client) callRaw(p params.Values, post bool) ([]byte, error) {
	body, err := w.call(p, post)
	if err != nil {
		return nil, err
	}
	if body != nil {
		defer body.Close()
	}

	buf, err := ioutil.ReadAll(body)
	if err != nil {
		return nil, err
	}

	return buf, nil
}

// Get performs a GET request with the specified parameters and returns the
// response as a *jason.Object.
// Get will return any API errors and/or warnings (if no other errors occur)
// as the error return value.
func (w *Client) Get(p params.Values) (*jason.Object, error) {
	return w.callJSON(p, false)
}

// GetRaw performs a GET request with the specified parameters
// and returns the raw JSON response as a []byte.
// Unlike Get, GetRaw does not check for API errors/warnings.
// GetRaw is useful when you want to decode the JSON into a struct for easier
// and safer use.
func (w *Client) GetRaw(p params.Values) ([]byte, error) {
	return w.callRaw(p, false)
}

// Post performs a POST request with the specified parameters and returns the
// response as a *jason.Object.
// Post will return any API errors and/or warnings (if no other errors occur)
// as the error return value.
func (w *Client) Post(p params.Values) (*jason.Object, error) {
	return w.callJSON(p, true)
}

// PostRaw performs a POST request with the specified parameters
// and returns the raw JSON response as a []byte.
// Unlike Post, PostRaw does not check for API errors/warnings.
// PostRaw is useful when you want to decode the JSON into a struct for easier
// and safer use.
func (w *Client) PostRaw(p params.Values) ([]byte, error) {
	return w.callRaw(p, true)
}

// Login attempts to login using the provided username and password.
// Do not use Login with OAuth.
func (w *Client) Login(username, password string) error {
	token, err := w.GetToken(LoginToken)
	if err != nil {
		return err
	}
	v := params.Values{
		"action":     "login",
		"lgname":     username,
		"lgpassword": password,
		"lgtoken":    token,
	}
	resp, err := w.Post(v)
	if err != nil {
		return err
	}
	lgResult, err := resp.GetString("login", "result")
	if err != nil {
		return fmt.Errorf("invalid API response: unable to assert login result to string")
	}
	if lgResult != "Success" {
		apierr := APIError{Code: lgResult}
		if reason, err := resp.GetString("login", "reason"); err == nil {
			apierr.Info = reason
		}
		return apierr
	}
	return nil
}

// Logout sends a logout request to the API.
// Logout does not take into account whether or not a user is actually logged in.
// Do not use Logout with OAuth.
func (w *Client) Logout() error {
	_, err := w.GetRaw(params.Values{"action": "logout"})
	return err
}

// OAuth configures OAuth authentication. After calling OAuth, future requests
// will be authenticated. OAuth does not make any API calls, so authentication
// failures will appear in response to the first API call after OAuth has
// been configured. Do not mix use of OAuth with Login/Logout.
func (w *Client) OAuth(consumerToken, consumerSecret, accessToken, accessSecret string) error {
	consumer := oauth.NewConsumer(consumerToken, consumerSecret, oauth.ServiceProvider{})
	access := oauth.AccessToken{
		Token:  accessToken,
		Secret: accessSecret,
	}

	httpc, err := consumer.MakeHttpClient(&access)
	if err != nil {
		return err
	}
	httpc.Jar = w.httpc.Jar
	w.httpc = httpc

	return nil
}
