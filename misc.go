package mwclient

import (
	"net/http"
	"sort"
	"strings"

	"cgt.name/pkg/go-mwclient/params"
)

// DumpCookies exports the cookies stored in the client.
func (w *Client) DumpCookies() []*http.Cookie {
	return w.cjar.Cookies(w.APIURL)
}

// LoadCookies imports cookies into the client.
func (w *Client) LoadCookies(cookies []*http.Cookie) {
	w.cjar.SetCookies(w.APIURL, cookies)
}

// GetPageIDs gets the pageids of the pages passed as arguments.
// GetPageIDs return a map[string]string where the key is the page name and
// the value is the page ID.
// If a page could not be found by the API, it will not be in the map,
// so check for existence when using the map.
func (w *Client) GetPageIDs(pageNames ...string) (IDs map[string]string, err error) {
	p := params.Values{
		"action":       "query",
		"prop":         "info",
		"indexpageids": "",
		"rawcontinue":  "",
	}

	if len(pageNames) == 0 {
		return nil, ErrNoArgs
	} else if len(pageNames) == 1 {
		p.Add("titles", pageNames[0])
	} else {
		// Normalize the title parameter's values to help
		// MediaWiki cache responses to queries.
		sort.Strings(pageNames)
		p.Add("titles", strings.Join(pageNames, "|"))
	}

	resp, err := w.Get(p)
	if err != nil {
		return nil, err
	}

	pageIDs, err := resp.GetPath("query", "pageids").StringArray()
	if err != nil {
		return nil, err
	}

	IDs = make(map[string]string)
	for _, id := range pageIDs {
		if id == "-1" {
			continue
		}
		name, ok := resp.GetPath("query", "pages", id).CheckGet("title")
		if !ok {
			continue
		}
		if nameStr, err := name.String(); err == nil {
			IDs[nameStr] = id
		}
	}

	return IDs, nil
}
