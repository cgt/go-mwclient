package mwclient

import (
	"encoding/json"
	"errors"
	"fmt"

	"cgt.name/pkg/go-mwclient/params"
)

var ErrEditNoChange = errors.New("edit successful, but did not change page")

// Edit takes a params.Values containing parameters for an edit action and
// attempts to perform the edit. Edit will return nil if no errors are detected.
// If the edit was successful, but did not result in a change to the page
// (i.e., the new text was identical to the current text)
// then ErrEditNoChange is returned.
// The p (params.Values) argument should contain parameters from:
//	https://www.mediawiki.org/wiki/API:Edit#Parameters
// Edit will set the 'action' and 'token' parameters automatically, but if the token
// field in p is non-empty, Edit will not override it.
// Edit does not check p for sanity.
// p example:
//	params.Values{
//		"pageid":   "709377",
//		"text":     "Complete new text for page",
//		"summary":  "Take that, page!",
//		"notminor": "",
//	}
func (w *Client) Edit(p params.Values) error {
	// If edit token not set, obtain one from API or cache
	if p["token"] == "" {
		csrfToken, err := w.GetToken(CSRFToken)
		if err != nil {
			return fmt.Errorf("unable to obtain csrf token: %s", err)
		}
		p["token"] = csrfToken
	}

	p.Set("action", "edit")

	resp, err := w.Post(p)
	if err != nil {
		return err
	}

	editResult, err := resp.GetPath("edit", "result").String()
	if err != nil {
		return fmt.Errorf("unable to assert 'result' field to type string\n")
	}

	if editResult != "Success" {
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

	if _, ok := resp.Get("edit").CheckGet("nochange"); ok {
		return ErrEditNoChange
	}

	return nil
}

// getPage gets the content of a page and the timestamp of its most recent revision.
// The page is specified either by its name or by its ID.
// If the isName parameter is true, then the pageIDorName parameter will be
// assumed to be a page name and vice versa.
func (w *Client) getPage(pageIDorName string, isName bool) (content string, timestamp string, err error) {
	p := params.Values{
		"action":       "query",
		"prop":         "revisions",
		"rvprop":       "content|timestamp",
		"indexpageids": "",
		"continue":     "",
	}

	if isName {
		p.Set("titles", pageIDorName)
	} else {
		p.Set("pageids", pageIDorName)
	}

	resp, err := w.Get(p)
	if err != nil {
		return "", "", err
	}

	pageIDs, err := resp.GetPath("query", "pageids").Array()
	if err != nil {
		return "", "", err
	}
	id := pageIDs[0].(string)
	if id == "0" || id == "-1" {
		return "", "", fmt.Errorf("page missing (title/id: %s)", pageIDorName)
	}

	revs, ok := resp.GetPath("query", "pages", id).CheckGet("revisions")
	if !ok {
		return "", "", fmt.Errorf("path query.pages.%s.revisions missing from response", id)
	}
	revs = revs.GetIndex(0)

	content, err = revs.Get("*").String()
	if err != nil {
		return "", "", fmt.Errorf("unable to assert page content to string: %s", err)
	}

	timestamp, err = revs.Get("timestamp").String()
	if err != nil {
		return "", "", fmt.Errorf("unable to assert timestamp to string: %s", err)
	}

	return content, timestamp, nil
}

// GetPageByName gets the content of a page (specified by its name) and
// the timestamp of its most recent revision.
func (w *Client) GetPageByName(pageName string) (content string, timestamp string, err error) {
	return w.getPage(pageName, true)
}

// GetPageByID gets the content of a page (specified by its id) and
// the timestamp of its most recent revision.
func (w *Client) GetPageByID(pageID string) (content string, timestamp string, err error) {
	return w.getPage(pageID, false)
}

// These consts represents MW API token names.
// They are meant to be used with the GetToken method like so:
// 	ClientInstance.GetToken(mwclient.CSRFToken)
const (
	CSRFToken                   = "csrf"
	DeleteGlobalAccountToken    = "deleteglobalaccount"
	PatrolToken                 = "patrol"
	RollbackToken               = "rollback"
	SetGlobalAccountStatusToken = "setglobalaccountstatus"
	UserRightsToken             = "userrights"
	WatchToken                  = "watch"
)

// GetToken returns a specified token (and an error if this is not possible).
// If the token is not already available in the Client.Tokens map,
// it will attempt to retrieve it via the API.
// tokenName should be "edit" (or whatever), not "edittoken".
// The token consts (e.g., mwclient.CSRFToken) should be used
// as the tokenName argument.
func (w *Client) GetToken(tokenName string) (string, error) {
	if _, ok := w.Tokens[tokenName]; ok {
		return w.Tokens[tokenName], nil
	}

	p := params.Values{
		"action":   "query",
		"meta":     "tokens",
		"type":     tokenName,
		"continue": "",
	}

	resp, err := w.Get(p)
	if err != nil {
		return "", err
	}

	token, err := resp.GetPath("query", "tokens", tokenName+"token").String()
	if err != nil {
		// This really shouldn't happen.
		return "", fmt.Errorf("error occured while converting token to string: %s", err)
	}
	w.Tokens[tokenName] = token
	return token, nil
}
