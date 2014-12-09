package mwclient

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"sort"
)

// DumpCookies exports the cookies stored in the client.
func (w *Client) DumpCookies() []*http.Cookie {
	return w.cjar.Cookies(w.APIURL)
}

// LoadCookies imports cookies into the client.
func (w *Client) LoadCookies(cookies []*http.Cookie) {
	w.cjar.SetCookies(w.APIURL, cookies)
}

// GetPageID gets the pageid of a page specified by its name.
func (w *Client) GetPageID(pageName string) (string, error) {
	params := url.Values{
		"action":       {"query"},
		"prop":         {"info"},
		"titles":       {pageName},
		"indexpageids": {""},
		"rawcontinue":  {""},
	}

	resp, err := w.Get(params)
	if err != nil {
		return "", err
	}

	pageIDs, err := resp.GetPath("query", "pageids").Array()
	if err != nil {
		return "", err
	}
	id := pageIDs[0].(string)
	if id == "-1" {
		return "", fmt.Errorf("page '%s' not found", pageName)
	}
	return id, nil
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
