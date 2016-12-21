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

func TestQueryValues_Add(t *testing.T) {
	v := Values{
		"foo": "a",
		"bar": "",
		// baz added later
	}

	v.Add("foo", "b")
	if g := v.Get("foo"); g != "a|b" {
		t.Errorf("Expected a|b, got %v", g)
	}

	v.Add("bar", "a")
	if g := v.Get("bar"); g != "|a" {
		t.Errorf("Expected |a, got %v", g)
	}

	v.Add("baz", "a")
	if g := v.Get("baz"); g != "a" {
		t.Errorf("Expected a, got %v", g)
	}

	v.Add("baz", "b")
	if g := v.Get("baz"); g != "a|b" {
		t.Errorf("Expected a|b, got %v", g)
	}
}

func TestQueryValues_AddRange_Append(t *testing.T) {
	v := make(Values)
	v.Add("foo", "bar")
	v.AddRange("foo", "quux", "x", "baz")

	if g := v.Get("foo"); g != "bar|quux|x|baz" {
		t.Errorf("expected bar|quux|x|baz, got %v", g)
	}
}

func TestQueryValues_AddRange_Empty(t *testing.T) {
	v := make(Values)
	v.AddRange("foo", "bar", "quux", "x")

	if g := v.Get("foo"); g != "bar|quux|x" {
		t.Errorf("expected bar|quux|x, got %v", g)
	}
}

// TestQueryValues_Add_Eq_AddRange tests that successive calls to Add
// have the same result as one call with the same parameters to AddRange.
func TestQueryValues_Add_Eq_AddRange(t *testing.T) {
	var (
		a = make(Values)
		b = make(Values)
	)

	a.Add("foo", "bar")
	a.Add("foo", "quux")
	a.Add("foo", "x")

	b.AddRange("foo", "bar", "quux", "x")

	ae := a.Encode()
	be := b.Encode()
	if ae != be {
		t.Errorf("a != b. a='%s', b='%s'", ae, be)
	}
}
