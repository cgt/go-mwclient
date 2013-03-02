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

const DefaultUserAgent = "go-mwclient (https://github.com/cgtdk/go-mwclient) by meta:User:Cgtdk"

type API struct {
	Client    *http.Client
	Jar       *cookiejar.Jar
	ApiUrl    string
	Format    string
	UserAgent string
}

func NewAPI(url string) *API {
	cjar := cookiejar.NewJar(false)
	httpclient := &http.Client{nil, nil, cjar}
	return &API{httpclient, cjar, url, "json", DefaultUserAgent}
}

func (c *API) Get(params url.Values) (*simplejson.Json, error) {
	// Ensure API returns JSON
	params.Set("format", c.Format)

	req, err := http.NewRequest("GET", fmt.Sprintf("%s?%s", c.ApiUrl, params.Encode()), nil)
	if err != nil {
		log.Printf("Error creation of request: %s\n", err)
		return nil, err
	}
	req.Header.Set("User-Agent", c.UserAgent)

	return c.handleResponse(req)
}

func (c *API) Post(params url.Values) (*simplejson.Json, error) {
	// Ensure API returns JSON
	params.Set("format", c.Format)

	req, err := http.NewRequest("POST", c.ApiUrl, strings.NewReader(params.Encode()))
	if err != nil {
		log.Printf("Error creation of request: %s\n", err)
		return nil, err
	}
	req.Header.Set("User-Agent", c.UserAgent)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	return c.handleResponse(req)
}

// handleResponse contains response handling code common to both API.Get() and API.Post().
func (c *API) handleResponse(req *http.Request) (*simplejson.Json, error) {
	urlValue, err := url.Parse(c.ApiUrl)
	if err != nil {
		log.Printf("Unable to parse URL %s: %s\n", c.ApiUrl, err)
		return nil, err
	}

	for _, cookie := range c.Jar.Cookies(urlValue) {
		req.AddCookie(cookie)
	}

	resp, err := c.Client.Do(req)
	defer resp.Body.Close()
	if err != nil {
		log.Printf("Error during GET: %s\n", err)
		return nil, err
	}
	c.Jar.SetCookies(req.URL, resp.Cookies())

	jsonBuf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading from resp.Body: %s\n", err)
		return nil, err
	}

	js, err := simplejson.NewJson(jsonBuf)
	if err != nil {
		log.Printf("Error during JSON parsing: %s\n", err)
		return nil, err
	}

	// Check for MediaWiki API errors
	if apiErr, ok := resp.Header["Mediawiki-Api-Error"]; ok {
		return js, errors.New(apiErr[0])
	}
	return js, nil
}

func (c *API) Login(username, password, token string) (bool, error) {
	v := url.Values{}
	v.Set("action", "login")
	v.Set("lgname", username)
	v.Set("lgpassword", password)
	if token != "" {
		v.Set("lgtoken", token)
	}

	resp, err := c.Post(v)
	if err != nil {
		return false, err
	}

	if lgResult, _ := resp.Get("login").Get("result").String(); lgResult != "Success" {
		if lgResult == "NeedToken" {
			lgToken, _ := resp.Get("login").Get("token").String()
			return c.Login(username, password, lgToken)
		} else {
			return false, errors.New(lgResult)
		}
	}

	return true, nil
}
