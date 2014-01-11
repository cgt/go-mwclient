package mwclient

import (
	"bytes"
	"net/url"
	"sort"
)

// URLEncode is a slightly modified version of Values.Encode() from net/url.
// It encodes url.Values into URL encoded form, sorted by key, with the exception
// of the key "maxlag", which will be appended to the end instead of being subject
// to regular sorting. This is done because that's what the MediaWiki API wants.
func URLEncode(v url.Values) string {
	if v == nil {
		return ""
	}
	var buf bytes.Buffer
	keys := make([]string, 0, len(v))
	for k := range v {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	maxlag := false
	for _, k := range keys {
		if k == "maxlag" {
			maxlag = true
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
	if maxlag {
		buf.WriteString("&" + url.QueryEscape("maxlag") + "=" + url.QueryEscape(v["maxlag"][0]))
	}
	return buf.String()
}
