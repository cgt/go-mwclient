package mwclient

import (
	"encoding/json"
	"fmt"
	"net/url"
)

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

// Edit takes a map[string]string containing parameters for an edit action and
// attempts to perform the edit. Edit will return nil if no errors are detected.
// The editcfg map[string]string argument should contain parameters from:
//	https://www.mediawiki.org/wiki/API:Edit#Parameters
// Edit will set the 'action' and 'token' parameters automatically, but if the token
// field in editcfg is non-empty, Edit will not override it.
// Edit does not check editcfg for sanity.
// editcfg example:
//	map[string]string{
//		"pageid":   "709377",
//		"text":     "Complete new text for page",
//		"summary":  "Take that, page!",
//		"notminor": "",
//	}
func (w *Client) Edit(editcfg map[string]string) error {
	// If edit token not set, obtain one from API or cache
	if editcfg["token"] == "" {
		editToken, err := w.GetToken("edit")
		if err != nil {
			return fmt.Errorf("unable to obtain edit token: %s", err)
		}
		editcfg["token"] = editToken
	}

	params := url.Values{}
	for k, v := range editcfg {
		params.Set(k, v)
	}
	params.Set("action", "edit")

	resp, err := w.Post(params)
	if err != nil {
		return err
	}

	if resp.GetPath("edit", "result").MustString() != "Success" {
		if captcha, ok := resp.Get("edit").CheckGet("captcha"); ok {
			captchaBytes, err := captcha.Encode()
			if err != nil {
				return fmt.Errorf("error occured while creating error message: %s", err)
			}
			var captchaerr captchaError
			err = json.Unmarshal(captchaBytes, &captchaerr)
			if err != nil {
				return fmt.Errorf("error occured while creating error message: %s", err)
			}
			return captchaerr
		}

		return fmt.Errorf("unrecognized response: %v", resp.Get("edit"))
	}

	return nil
}

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
		return "", "", fmt.Errorf("API could not retrieve page with pageid %s", pageID)
	}

	rv := resp.GetPath("query", "pages", pageID).Get("revisions").GetIndex(0)

	content, err := rv.Get("*").String()
	if err != nil {
		// I don't know when this would ever happen, but just to be safe...
		return "", "", fmt.Errorf("unable to assert page content to string: %s", err)
	}

	timestamp, err := rv.Get("timestamp").String()
	if err != nil {
		return "", "", fmt.Errorf("unable to assert timestamp to string: %s", err)
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

	resp, err := w.Get(parameters)
	if err != nil {
		return "", err
	}

	token, err := resp.GetPath("tokens", tokenName+"token").String()
	if err != nil {
		// This really shouldn't happen.
		return "", fmt.Errorf("error occured while converting token to string: %s", err)
	}
	w.Tokens[tokenName] = token
	return token, nil
}
