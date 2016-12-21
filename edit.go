package mwclient

import (
	"encoding/json"
	"errors"
	"fmt"

	"cgt.name/pkg/go-mwclient/params"
)

// ErrEditNoChange is returned by Client.Edit() when an edit did not change
// a page but was otherwise successful.
var ErrEditNoChange = errors.New("edit successful, but did not change page")

// ErrPageNotFound is returned when a page is not found.
// See GetPage[s]ByName().
var ErrPageNotFound = errors.New("wiki page not found")

// Edit takes a params.Values containing parameters for an edit action and
// attempts to perform the edit. Edit will return nil if no errors are detected.
// If the edit was successful, but did not result in a change to the page
// (i.e., the new text was identical to the current text)
// then ErrEditNoChange is returned.
// The p (params.Values) argument should contain parameters from:
//	https://www.mediawiki.org/wiki/API:Edit#Parameters
// Edit will set the 'action' and 'token' parameters automatically, but if the
// token field in p is non-empty, Edit will not override it.
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

	p["action"] = "edit"

	resp, err := w.Post(p)
	if err != nil {
		return err
	}

	editResult, err := resp.GetString("edit", "result")
	if err != nil {
		return fmt.Errorf("unable to assert 'result' field to type string\n")
	}

	if editResult != "Success" {
		if captcha, err := resp.GetObject("edit", "captcha"); err == nil {
			captchaBytes, err := captcha.Marshal()
			if err != nil {
				return fmt.Errorf("error occured while creating error message: %s", err)
			}
			var captchaerr CaptchaError
			err = json.Unmarshal(captchaBytes, &captchaerr)
			if err != nil {
				return fmt.Errorf("error occured while creating error message: %s", err)
			}
			return captchaerr
		}

		edit, _ := resp.GetValue("edit")
		return fmt.Errorf("unrecognized response: %v", edit)
	}

	if nochange, err := resp.GetBoolean("edit", "nochange"); err == nil && nochange {
		return ErrEditNoChange
	}

	return nil
}

// getPage gets the content of a page and the timestamp of its most recent revision.
// The page is specified either by its name or by its ID.
// If the isName parameter is true, then the pageIDorName parameter will be
// assumed to be a page name and vice versa.
func (w *Client) getPage(pageIDorName string, isName bool) (content string, timestamp string, err error) {
	pages, err := w.getPages(isName, pageIDorName)
	if err != nil {
		return "", "", err
	}

	page := pages[pageIDorName]
	return page.Content, page.Timestamp, page.Error
}

// TODO: refactor getPages to be compatible with formatversion=2

// getPages is just like getPage, but performs a multi-query so that
// only one network call will be used to get the contents of many pages.
// Maps the input name onto a BriefRevision result.
func (w *Client) getPages(areNames bool, pageIDsOrNames ...string) (pages map[string]BriefRevision, err error) {
	if len(pageIDsOrNames) == 0 {
		return nil, ErrNoArgs
	}

	pages = make(map[string]BriefRevision, len(pageIDsOrNames))

	p := params.Values{
		"action":       "query",
		"prop":         "revisions",
		"rvprop":       "content|timestamp",
		"indexpageids": "",
		"continue":     "",
	}

	for _, identifier := range pageIDsOrNames {
		if areNames {
			p.Add("titles", identifier)
		} else {
			p.Add("pageids", identifier)
		}
	}

	resp, err := w.Get(p)
	if err != nil {
		return nil, err
	}

	// make sure we can properly map input page names
	// to output names in the output map.
	// reversed normalized titles
	// canonical -> inputted
	denormalizedNames := make(map[string]string)
	if normalizations, err := resp.GetObjectArray("query", "normalized"); err == nil {
		for _, fix := range normalizations {
			from, err := fix.GetString("from")
			if err != nil {
				return nil, err
			}
			to, err := fix.GetString("to")
			if err != nil {
				return nil, err
			}
			denormalizedNames[to] = from
		}
	}

	pageIDs, err := resp.GetStringArray("query", "pageids")
	if err != nil {
		return nil, err
	}

	for _, id := range pageIDs { // fill the pages
		page := BriefRevision{PageID: id}

		entry, err := resp.GetObject("query", "pages", id)
		if err != nil {
			return nil, fmt.Errorf("API error: expected page to be in pages array")
		}

		if _, err := entry.GetValue("missing"); err == nil {
			page.Error = ErrPageNotFound
			title, err := entry.GetString("title")
			if err != nil {
				return nil, err
			}
			pages[title] = page
			continue
		}

		revs, err := entry.GetObjectArray("revisions")
		if err != nil {
			return nil, fmt.Errorf("API error: revision list not returned")
		}

		rev := revs[0]

		page.Content, err = rev.GetString("*")
		if err != nil {
			return nil, fmt.Errorf("unable to assert page content to string: %s", err)
		}

		page.Timestamp, err = rev.GetString("timestamp")
		if err != nil {
			return nil, fmt.Errorf("unable to assert timestamp to string: %s", err)
		}

		trueTitle, err := entry.GetString("title")
		if err != nil {
			return nil, fmt.Errorf("API error: page entry does not have title field")
		}

		if inputted, ok := denormalizedNames[trueTitle]; ok {
			pages[inputted] = page
		} else {
			pages[trueTitle] = page
		}
	}

	return pages, nil
}

// GetPageByName gets the content of a page (specified by its name) and
// the timestamp of its most recent revision.
func (w *Client) GetPageByName(pageName string) (content string, timestamp string, err error) {
	return w.getPage(pageName, true)
}

// GetPagesByName gets the contents of multiple pages (specified by their names).
// Returns a map of input page names to BriefRevisions.
func (w *Client) GetPagesByName(pageNames ...string) (pages map[string]BriefRevision, err error) {
	return w.getPages(true, pageNames...)
}

// GetPageByID gets the content of a page (specified by its id) and
// the timestamp of its most recent revision.
func (w *Client) GetPageByID(pageID string) (content string, timestamp string, err error) {
	return w.getPage(pageID, false)
}

// GetPagesByID gets the content of pages (specified by id).
// Returns a map of input page names to BriefRevisions.
func (w *Client) GetPagesByID(pageIDs ...string) (pages map[string]BriefRevision, err error) {
	return w.getPages(false, pageIDs...)
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
	LoginToken                  = "login"
)

// GetToken returns a specified token (and an error if this is not possible).
// If the token is not already available in the Client.Tokens map,
// it will attempt to retrieve it via the API.
// tokenName should be "edit" (or whatever), not "edittoken".
// The token consts (e.g., mwclient.CSRFToken) should be used
// as the tokenName argument.
func (w *Client) GetToken(tokenName string) (string, error) {
	// Always obtain a fresh login token
	if tokenName != LoginToken {
		if tok, ok := w.Tokens[tokenName]; ok {
			return tok, nil
		}
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

	token, err := resp.GetString("query", "tokens", tokenName+"token")
	if err != nil {
		// This really shouldn't happen.
		return "", fmt.Errorf("error occured while converting token to string: %s", err)
	}
	if tokenName != LoginToken {
		w.Tokens[tokenName] = token
	}
	return token, nil
}
