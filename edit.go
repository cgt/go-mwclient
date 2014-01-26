package mwclient

import (
	"fmt"
	"net/url"
)

// GetPage gets the content of a page specified by its pageid and the timestamp
// of its most recent revision, and returns the content, the timestamp, and an error.
func (w *Client) GetPage(pageID string) (string, string, error) {
	parameters := url.Values{
		"action":  {"query"},
		"prop":    {"revisions"},
		"rvprop":  {"content|timestamp"},
		"pageids": {pageID},
	}

	resp, err := w.Get(parameters)
	if err != nil {
		return "", "", err
	}

	// Check if API could find the page
	if _, ok := resp.GetPath("query", "pages", pageID).CheckGet("missing"); ok {
		return "", "", fmt.Errorf("API could not retrieve page with pageid %s.", pageID)
	}

	rv := resp.GetPath("query", "pages", pageID).Get("revisions").GetIndex(0)

	content, err := rv.Get("*").String()
	if err != nil {
		// I don't know when this would ever happen, but just to be safe...
		return "", "", fmt.Errorf("Unable to assert page content to string: %s", err)
	}

	timestamp, err := rv.Get("timestamp").String()
	if err != nil {
		return "", "", fmt.Errorf("Unable to assert timestamp to string: %s", err)
	}

	return content, timestamp, nil
}

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
