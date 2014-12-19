package mwclient

import (
	"bytes"
	"net/http"
	"net/url"
	"sort"
	"strings"
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
	params := url.Values{
		"action":       {"query"},
		"prop":         {"info"},
		"indexpageids": {""},
		"rawcontinue":  {""},
	}

	if len(pageNames) == 0 {
		return nil, ErrNoArgs
	} else if len(pageNames) == 1 {
		params.Add("titles", pageNames[0])
	} else {
		// Normalize the title parameter's values to help
		// MediaWiki cache responses to queries.
		sort.Strings(pageNames)
		params.Add("titles", strings.Join(pageNames, "|"))
	}

	resp, err := w.Get(params)
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

// urlEncode is a slightly modified version of Values.Encode() from net/url.
// It encodes url.Values into URL encoded form, sorted by key, with the exception
// of the key "token", which will be appended to the end instead of being subject
// to regular sorting. This is done in accordance with MW API guidelines to
// ensure that an action will not be executed if the query string has been cut
// off for some reason.
func urlEncode(v url.Values) string {
	if v == nil {
		return ""
	}
	var buf bytes.Buffer
	keys := make([]string, 0, len(v))
	for k := range v {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	token := false
	for _, k := range keys {
		if k == "token" {
			token = true
			continue
		}
		vs := v[k]
		prefix := url.QueryEscape(k) + "="
		for _, v := range vs {
			if buf.Len() > 0 {
				buf.WriteByte('&')
			}
			buf.WriteString(prefix)
			buf.WriteString(url.QueryEscape(v))
		}
	}
	if token {
		buf.WriteString("&" + url.QueryEscape("token") + "=" + url.QueryEscape(v["token"][0]))
	}
	return buf.String()
}
