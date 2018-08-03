package mwclient

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/antonholmquist/jason"

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

// BriefRevision contains basic information on a single revision of a page.
type BriefRevision struct {
	Content   string
	Timestamp string
	Error     error
	PageID    string
}

// getPage gets the content of a page and the timestamp of its most recent revision.
// The page is specified either by its name or by its ID.
// If the isName parameter is true, then the pageIDorName parameter will be
// assumed to be a page name and vice versa.
func (w *Client) getPage(pageIDorName string, isName bool) (content string, timestamp string, err error) {
	pages, err := w.getPages(isName, pageIDorName)
	if pages == nil && err != nil {
		return "", "", err
	}
	page := pages[pageIDorName]
	if page.Error != nil {
		return "", "", page.Error
	}
	return page.Content, page.Timestamp, err
}

// getPages is just like getPage, but performs a multi-query so that
// only one API request will be used to get the contents of many pages.
// Maps the input name onto a BriefRevision result.
func (w *Client) getPages(areNames bool, pageIDsOrNames ...string) (pages map[string]BriefRevision, err error) {
	if len(pageIDsOrNames) == 0 {
		return nil, ErrNoArgs
	}

	p := params.Values{
		"action":  "query",
		"prop":    "revisions",
		"rvprop":  "content|timestamp",
		"rvslots": "main",
	}
	if areNames {
		p.AddRange("titles", pageIDsOrNames...)
	} else {
		p.AddRange("pageids", pageIDsOrNames...)
	}

	r, err := w.call(p, false)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	var resp getPagesResponse
	err = json.NewDecoder(r).Decode(&resp)
	if err != nil {
		return nil, err
	}
	return handleGetPages(pageIDsOrNames, resp)
}

func handleGetPages(pageNames []string, resp getPagesResponse) (pages map[string]BriefRevision, err error) {
	// Return warnings as errors along with any data.
	// If a warning is returned, it is possible that the data is wrong.
	// For example, the query could have asked for more than 50 pages,
	// in which case only 50 will be returned and the rest will be left out.
	var warnings error
	if resp.Warnings != nil {
		j, err := jason.NewObjectFromBytes(resp.Warnings)
		if err != nil {
			return nil, fmt.Errorf("error decoding warnings: %v", err)
		}
		warnings = extractWarnings(j)
		if warnings == nil {
			return nil, fmt.Errorf("error decoding warnings: no warnings: %v", resp.Warnings)
		}
	}

	// make sure we can properly map input page names
	// to output names in the output map.
	// reversed normalized titles
	// canonical -> inputted
	normalized := resp.Query.Normalized
	denormalizedNames := make(map[string]string, len(normalized))
	if normalized != nil {
		for _, norm := range normalized {
			denormalizedNames[norm.To] = norm.From
		}
	}

	pages = make(map[string]BriefRevision, len(pageNames))
	for _, entry := range resp.Query.Pages {
		var page BriefRevision

		// Missing and Special errors are not mutually exclusive,
		// but treat them as if they were because it's easier.
		if entry.Missing {
			page.Error = ErrPageNotFound
		} else if entry.Special {
			page.Error = errors.New("special pages not supported for this query")
		}

		if page.Error == nil {
			page.PageID = strconv.Itoa(entry.PageID)

			rev := entry.Revisions[0]
			page.Content = rev.Slots.Main.Content
			page.Timestamp = rev.Timestamp
		}

		var title string
		if inputTitle, ok := denormalizedNames[entry.Title]; ok {
			title = inputTitle
		} else {
			title = entry.Title
		}
		pages[title] = page
	}

	return pages, warnings
}

type getPagesResponse struct {
	Warnings json.RawMessage `json:"warnings"`
	Query    struct {
		Normalized []struct {
			From string `json:"from"`
			To   string `json:"to"`
		} `json:"normalized"`
		Pages []struct {
			Missing   bool   `json:"missing"`
			Special   bool   `json:"special"`
			PageID    int    `json:"pageid"`
			Title     string `json:"title"`
			Revisions []struct {
				Timestamp string `json:"timestamp"`
				Slots     struct {
					Main struct {
						ContentModel  string `json:"contentmodel"`
						ContentFormat string `json:"contentformat"`
						Content       string `json:"content"`
					}
				}
			} `json:"revisions"`
		} `json:"pages"`
	} `json:"query"`
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
