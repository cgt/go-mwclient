# go-mwclient

go-mwclient is a Go package for interacting with the MediaWiki JSON API.
The package's actual name is "mwclient", but it is called "go-mwclient"
to avoid confusion with similarly named libraries written in other
programming languages.

go-mwclient aims to be a thin wrapper around the MediaWiki API that
takes care of the most tedious parts of interacting with the API (such
as authentication and query continuation), but it does not aim to
abstract away all the functionality of the API.

The canonical import path for this package is

    import cgt.name/pkg/go-mwclient // imports "mwclient"

Documentation:
- GoDoc: <http://godoc.org/cgt.name/pkg/go-mwclient>
- MediaWiki API docs: <https://www.mediawiki.org/wiki/API:Main_page>

## Installation

    go get cgt.name/pkg/go-mwclient

## API stability

go-mwclient's API seems to have stabilized. It probably won't change
significantly, if at all.

TODO: v1

## Example

    package main

    import (
        "fmt"

        "cgt.name/pkg/go-mwclient"
    )

    func main() {
        // Initialize a *Client with New(), specifying the wiki's API URL
        // and your HTTP User-Agent. Try to use a meaningful User-Agent.
        w, err := mwclient.New("https://en.wikipedia.org/w/api.php", "myWikibot")
        if err != nil {
            panic(err)
        }

        // Log in.
        err = w.Login("USERNAME", "PASSWORD")
        if err != nil {
            panic(err)
        }

        // Specify parameters to send.
        parameters := map[string]string{
            "action":   "query",
            "list":     "recentchanges",
            "rclimit":  "2",
            "rctype":   "edit",
            "continue": "",
        }

        // Make the request.
        resp, err := w.Get(parameters)
        if err != nil {
            panic(err)
        }

        // Print the *jason.Object
        fmt.Println(resp)
    }

## Dependencies
Other than the Go standard library, go-mwclient depends on the following
third party, open source packages:

- <https://github.com/antonholmquist/jason> (MIT licensed)
- <https://github.com/mrjones/oauth> (MIT licensed)

## Copyright

To the extent possible under law, the author(s) have dedicated all
copyright and related and neighboring rights to this software to the
public domain worldwide. This software is distributed without any
warranty.

You should have received a copy of the CC0 Public
Domain Dedication along with this software. If not, see
http://creativecommons.org/publicdomain/zero/1.0/.

### params package

The `params` package is based on the `net/url` package from the Go
standard library, which is released under a BSD-style license. See
params/LICENSE.

Contributions to the `params` package as part of this project are
released to the public domain via CC0, as noted above and specified in
the COPYING file.
