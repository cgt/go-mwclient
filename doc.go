/*
Package mwclient provides functionality for interacting with the MediaWiki API.

go-mwclient is intended for users who are already familiar with (or are
willing to learn) the MediaWiki API. It is intended to make dealing with
the API more convenient, but not to hide it.

go-mwclient v1 uses version 2 of the MW JSON API.

Basic usage

In the example below, basic usage of go-mwclient is shown.

	// Initialize a *Client with New(), specifying the wiki's API URL
	// and your HTTP User-Agent. Try to use a meaningful User-Agent.
	w, err := mwclient.New("https://en.wikipedia.org/w/api.php", "myWikibot")
	if err != nil {
		panic(err) // Malformed URL
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
methods for details). They all offer the same basic interface: pass a
params.Values map (from the cgt.name/pkg/go-mwclient/params package),
receive a response and an error.

For convenience, go-mwclient offers several methods for making common
requests (login, edit, etc.), but these methods are implemented using
the same interface.

params.Values

params.Values is similar to (and a fork of) the standard library's
net/url.Values. The reason why params.Values is used instead is
that url.Values is based on a map[string][]string, rather than a
map[string]string. This is because url.Values must support multiple keys
with the same name.

The literal syntax for a map[string][]string is rather cumbersome
because the value is a slice rather than just a string, and the
MediaWiki API actually does not use multiple keys when multiple values
for the same key is required. Instead, one key is used and the values
are separated by pipes (|). It is therefore very simple to write
multi-value values in params.Values literals while

params.Values makes it simple to write multi-value values in literals
while avoiding the cumbersome []string literals for the most common case
where the is only value.

See documentation for the params package for more information.

Because of the way type identity works in Go, it is possible for callers
to pass a plain map[string]string rather than a params.Values. It is
only necessary for users to use params.Values directly if they wish to
use params.Values's methods. It makes no difference to go-mwclient.

Error handling

If an API call fails it will return an error. Many things can go wrong
during an API call: the network could be down, the API could return an
unexpected response (if the API was changed), or perhaps there's an
error in your API request.

If the error is an API error or warning (and you used the "non-Raw" Get
and Post methods), then the error/warning(s) will be parsed and returned
in either an APIError or an APIWarnings object, both of which implement
the error interface. The "Raw" request methods do not check for API
errors or warnings.

For more information about API errors and warnings, please see
https://www.mediawiki.org/wiki/API:Errors_and_warnings.

If maxlag is enabled, it may be that the API has rejected the requests
and the amount of retries (3 by default) have been tried unsuccessfully.
In that case, the error will be the variable mwclient.ErrAPIBusy.

Other methods than the core ones (i.e., other methods than Get and Post)
may return other errors.
*/
package mwclient // import "cgt.name/pkg/go-mwclient"
