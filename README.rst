===========
go-mwclient
===========

go-mwclient is a library for interacting with the MediaWiki API. The package is
actually called "mwclient", but I chose to prefix the repository's name with
"go-" to avoid confusion with similarly named libraries.

::

    import "github.com/cgt/go-mwclient" // imports "mwclient"

Documentation is available at: http://godoc.org/github.com/cgt/go-mwclient

This library's API is subject to breaking change at any time for the time being.

Example
=======

::

    package main

    import (
        "fmt"
        "net/url"

        "github.com/cgt/go-mwclient"
    )

    func main() {
        // Make a Client object and specify the wiki's API URL and your user agent.
        w := mwclient.NewDefault("https://da.wikipedia.org/w/api.php", "Username's wikibot")

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
