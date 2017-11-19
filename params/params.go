// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package params is a MediaWiki specific replacement for parts of net/url.
// Specifically, it contains a fork of url.Values (params.Values) that
// is based on map[string]string instead of map[string][]string.
// The purpose of this is that the MediaWiki API does not use multiple keys
// to allow multiple values for a key (e.g., "a=b&a=c"). Instead it uses
// one key with values separated by a pipe (e.g. "a=b|c").
package params // import "cgt.name/pkg/go-mwclient/params"

import (
	"bytes"
	"net/url"
	"sort"
	"strings"
)

// Values maps a string key to a string value.
// It is typically used for query parameters and form values.
// Unlike in the http.Header map, the keys in a Values map
// are case-sensitive.
type Values map[string]string

// Get gets the value associated with the given key.
// If there are no values associated with the key, Get returns
// the empty string.
func (v Values) Get(key string) string {
	if v == nil {
		return ""
	}
	vs, ok := v[key]
	if !ok {
		return ""
	}
	return vs
}

// Set sets the key to value. It replaces any existing
// values.
func (v Values) Set(key, value string) {
	v[key] = value
}

// Add adds the value to key. It appends to any existing
// values associated with key.
func (v Values) Add(key, value string) {
	if current, ok := v[key]; ok {
		v[key] = strings.Join([]string{current, value}, "|")
	} else {
		v[key] = value
	}
}

// AddRange adds multiple values to a key.
// It appends to any existing values associated with key.
func (v Values) AddRange(key string, values ...string) {
	if current, ok := v[key]; ok {
		list := make([]string, 0, 1+len(values))
		list = append(list, current)
		list = append(list, values...)
		v[key] = strings.Join(list, "|")
	} else {
		v[key] = strings.Join(values, "|")
	}
}

// Del deletes the value associated with key.
func (v Values) Del(key string) {
	delete(v, key)
}

// Encode encodes the values into ``URL encoded'' form
// ("bar=baz&foo=quux") sorted by key.
// Encode is a slightly modified version of Values.Encode() from net/url.
// It encodes url.Values into URL encoded form, sorted by key, with the exception
// of the key "token", which will be appended to the end instead of being subject
// to regular sorting. This is done in accordance with MW API guidelines to
// ensure that an action will not be executed if the query string has been cut
// off for some reason.
func (v Values) Encode() string {
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
		prefix := url.QueryEscape(k) + "="
		if buf.Len() > 0 {
			buf.WriteByte('&')
		}
		buf.WriteString(prefix)
		buf.WriteString(url.QueryEscape(v[k]))
	}
	if token {
		buf.WriteString("&token=" + url.QueryEscape(v["token"]))
	}
	return buf.String()
}
