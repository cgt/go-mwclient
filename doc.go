/*
Package mwclient provides functionality for interacting with the MediaWiki API.

go-mwclient is intended for users who are already familiar with (or are
willing to learn) the MediaWiki API. It is intended to make dealing with
the API more convenient, but not to hide it.

Basic usage

In the example below, basic usage of go-mwclient is shown.

	wiki, err := mwclient.New("https://wiki.example.com/w/api.php", "my user agent")
	if err != nil {
		// Malformed URL or empty user agent
		panic(err)
	}

	parameters := params.Values{
		"action":   "query",
		"list":     "recentchanges",
	}
	response, err := wiki.Get(parameters)
	if err != nil {
		panic(err)
	}

Create a new Client object with the New() constructor, and then you are
ready to start making requests to the API. If you wish to make requests
to multiple MediaWiki sites, you must create a Client for each of them.

go-mwclient offers a few methods for making arbitrary requests to
the API: Get, GetRaw, Post, and PostRaw (see documentation for the
methods for details). They all offer the same basic interface: pass
a params.Values (from the cgt.name/pkg/go-mwclient/params package),
receive a response and an error.

params.Values is similar to (and a fork of) the standard library's
net/url.Values. The reason why params.Values is used instead is that
url.Values is based on a map[string][]string because it must allow
multiple keys with the same name. However, the literal syntax for such
a map is rather cumbersome, and the MediaWiki API actually does not use
multiple keys when multiple values for the same key is desired. Instead,
one key is used and the values are separated by pipes (|). Therefore,
the decision to use params.Values (which is based on map[string]string)
instead was made.

For convenience, go-mwclient offers several methods for making common
requests (login, edit, etc.), but these methods are implemented using
the same interface.

Error handling

If an API call fails, for whatever reason, it will return an error. Many
things can go wrong during an API call: the network could be down, the
API could return an unexpected response (if the API was changed), or
perhaps there's an error in your API request.

If the error is an API error or warning (and you used the "non-Raw" Get
and Post methods), then the error/warning(s) will be parsed and returned
in either an mwclient.APIError or an mwclient.APIWarnings object, both
of which implement the error interface.

For more information about API errors and warnings, please see
https://www.mediawiki.org/wiki/API:Errors_and_warnings.

If maxlag is enabled, it may be that the API has rejected the requests
and the amount of retries (3 by default) have been tried unsuccessfully.
In that case, the error will be the variable mwclient.ErrAPIBusy.

Other methods than the core ones (i.e., other methods than Get and Post)
may return other errors.
*/
package mwclient // import "cgt.name/pkg/go-mwclient"
