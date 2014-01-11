package mwclient

import (
	"fmt"
	"net/url"
)

// GetToken returns a specified token (and an error if this is not possible).
// If the token is not already available in the Client.Tokens map,
// it will attempt to retrieve it via the API.
// tokenName should be "edit" (or whatever), not "edittoken".
func (w *Client) GetToken(tokenName string) (string, error) {
	if _, ok := w.Tokens[tokenName]; ok {
		return w.Tokens[tokenName], nil
	}

	parameters := url.Values{
		"action": {"tokens"},
		"type":   {tokenName},
	}

	resp, err, apiok := ErrorCheck(w.Get(parameters))
	if err != nil {
		return "", err
	}
	if !apiok {
		// Check for errors
		if err, ok := resp.CheckGet("error"); ok {
			newError := fmt.Errorf("%s: %s", err.Get("code").MustString(), err.Get("info").MustString())
			return "", newError
		}

		// Check for warnings
		if warnings, ok := resp.CheckGet("warnings"); ok {
			newError := fmt.Errorf(warnings.GetPath("tokens", "*").MustString())
			return "", newError
		}
	}

	token, err := resp.GetPath("tokens", tokenName+"token").String()
	if err != nil {
		// This really shouldn't happen.
		return "", fmt.Errorf("Error occured while converting token to string: %s", err)
	}
	w.Tokens[tokenName] = token
	return token, nil
}

// GetPage gets the content of a page specified by its pageid and returns it as a string.
func (w *Client) GetPage(pageid string) (string, error) {
	parameters := url.Values{
		"action":  {"query"},
		"prop":    {"revisions"},
		"rvprop":  {"content"},
		"pageids": {pageid},
	}

	resp, err := w.Get(parameters)
	if err != nil {
		return "", err
	}

	if _, ok := resp.GetPath("query", "pages", pageid).CheckGet("missing"); ok {
		return "", fmt.Errorf("API could not retrieve page with pageid %s.", pageid)
	}

	content, err := resp.GetPath("query", "pages", pageid).Get("revisions").GetIndex(0).Get("*").String()
	if err != nil {
		// I don't know when this would ever happen, but just to be safe...
		return "", fmt.Errorf("Unable to assert page content to string: %s", err)
	}
	return content, nil

}
