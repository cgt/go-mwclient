package mwclient

import (
	"net/url"
	"testing"
)

func TestUrlEncode(t *testing.T) {
	params := url.Values{
		"a":     {"blah"},
		"b":     {""},
		"token": {"gibberishhere"},
		"s":     {"10"},
		"z":     {"green"},
	}

	if encoded := urlEncode(params); encoded != "a=blah&b=&s=10&z=green&token=gibberishhere" {
		t.Errorf("urlEncode returns '%s' for params", encoded)
	}
}
