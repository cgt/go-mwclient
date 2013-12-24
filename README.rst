===========
go-mwclient
===========

go-mwclient is a library for interacting with the MediaWiki API. The package is
actually called "mwclient", but I chose to prefix the repository's name with
"go-" to avoid confusion with similarly named libraries.

::

    import "github.com/cgtdk/go-mwclient" // imports "mwclient"

This library is still under development (albeit it is sporadic), so do not
consider the API stable.

Documentation is available at: http://godoc.org/github.com/cgtdk/go-mwclient

Example
=======

::

	package main

	import (
		"fmt"
		"github.com/cgtdk/go-mwclient"
		"net/url"
	)

	func main() {
		// Make a Wiki object and specify the wiki's API URL.
		w := mwclient.NewWiki("https://da.wikipedia.org/w/api.php")

		// Log in.
		err := w.Login("USERNAME", "PASSWORD")
		if err != nil {
			fmt.Println(err)
		}

		// Specify parameters to send.
		parameters := url.Values{
			"action":  {"query"},
			"list":    {"recentchanges"},
			"rclimit": {"2"},
			"rctype":  {"edit"},
		}

		// Make the request.
		resp, err := w.Get(parameters)
		if err != nil {
			fmt.Println(err)
		}

		// Print the *simplejson.Json object.
		fmt.Println(resp)
	}

Legal information
=================
The go-mwclient project (i.e., all of its source code, documentation, and other
files) is placed in the public domain via Creative Commons CC0. See
the COPYING file or http://creativecommons.org/publicdomain/zero/1.0/ for
details.

Licenses for third party code can be found in the ATTRIBUTION file.
