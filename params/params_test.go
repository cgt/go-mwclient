// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package params

import "testing"

type EncodeQueryTest struct {
	m        Values
	expected string
}

var encodeQueryTests = []EncodeQueryTest{
	{nil, ""},
	{Values{"q": "puppies", "oe": "utf8"}, "oe=utf8&q=puppies"},
	{Values{"x": "c", "a": "b", "token": "t"}, "a=b&x=c&token=t"},
}

func TestEncodeQuery(t *testing.T) {
	for _, tt := range encodeQueryTests {
		if q := tt.m.Encode(); q != tt.expected {
			t.Errorf(`EncodeQuery(%+v) = %q, want %q`, tt.m, q, tt.expected)
		}
	}
}

func TestQueryValues(t *testing.T) {
	v := Values{
		"foo": "bar",
		"bar": "1",
	}
	if len(v) != 2 {
		t.Errorf("got %d keys in Query values, want 2", len(v))
	}
	if g, e := v.Get("foo"), "bar"; g != e {
		t.Errorf("Get(foo) = %q, want %q", g, e)
	}
	// Case sensitive:
	if g, e := v.Get("Foo"), ""; g != e {
		t.Errorf("Get(Foo) = %q, want %q", g, e)
	}
	if g, e := v.Get("bar"), "1"; g != e {
		t.Errorf("Get(bar) = %q, want %q", g, e)
	}
	if g, e := v.Get("baz"), ""; g != e {
		t.Errorf("Get(baz) = %q, want %q", g, e)
	}
	v.Del("bar")
	if g, e := v.Get("bar"), ""; g != e {
		t.Errorf("second Get(bar) = %q, want %q", g, e)
	}
}
