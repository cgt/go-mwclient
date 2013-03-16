// Package mwclient provides methods for interacting with the MediaWiki API.
package mwclient

import (
	"code.google.com/p/cookiejar"
	"errors"
	"fmt"
	simplejson "github.com/bitly/go-simplejson"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
)

// If you modify this package, please change the user agent.
const DefaultUserAgent = "go-mwclient (https://github.com/cgtdk/go-mwclient) by meta:User:Cgtdk"

type Wiki struct {
	client            *http.Client
	cjar              *cookiejar.Jar
	ApiUrl            *url.URL
	format, UserAgent string
}

// NewWiki returns an initialized Wiki object. If the provided API url is an
// invalid URL (as defined by the net/url package), then it will panic
// with the error from url.Parse().
func NewWiki(inUrl string) *Wiki {
	cjar := cookiejar.NewJar(false)
	apiurl, err := url.Parse(inUrl)
	if err != nil {
		panic(err) // Yes, this is bad, but so is using bad URLs and I don't want two return values.
	}
	return &Wiki{
		&http.Client{nil, nil, cjar},
		cjar,
		apiurl,
		"json",
		DefaultUserAgent,
	}
}

// call makes a GET or POST request to the Mediawiki API (depending on whether
// the post argument is true or false (if true, it will POST).
func (w *Wiki) call(params url.Values, post bool) (*simplejson.Json, error) {
	params.Set("format", w.format)

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
		req, err = http.NewRequest(httpMethod, w.ApiUrl.String(), strings.NewReader(params.Encode()))
	} else {
		req, err = http.NewRequest(httpMethod, fmt.Sprintf("%s?%s", w.ApiUrl.String(), params.Encode()), nil)
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
	resp, err := w.client.Do(req)
	defer resp.Body.Close()
	if err != nil {
		log.Printf("Error during %s: %s\n", httpMethod, err)
		return nil, err
	}

	// Set any new cookies
	w.cjar.SetCookies(req.URL, resp.Cookies())

	jsonResp, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading from resp.Body: %s\n", err)
		return nil, err
	}

	js, err := simplejson.NewJson(jsonResp)
	if err != nil {
		log.Printf("Error during JSON parsing: %s\n", err)
		return nil, err
	}

	return js, nil
}

// Get wraps the w.call method to make it do a GET request.
func (w *Wiki) Get(params url.Values) (*simplejson.Json, error) {
	return w.call(params, false)
}

// Post wraps the w.call method to make it do a POST request.
func (w *Wiki) Post(params url.Values) (*simplejson.Json, error) {
	return w.call(params, true)
}

// Login attempts to login using the provided username and password.
func (w *Wiki) Login(username, password string) error {

	// By using a closure, we avoid requiring the public Login method to have a token parameter.
	var loginFunc func(token string) error

	loginFunc = func(token string) error {
		v := url.Values{}
		v.Set("action", "login")
		v.Set("lgname", username)
		v.Set("lgpassword", password)
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
func (w *Wiki) Logout() bool {
	w.Get(url.Values{"action": {"logout"}})
	return true
}
