=============
 go-mwclient
=============

go-mwclient is a library for interacting with the MediaWiki API. The package's
actual name is "mwclient", but it is called "go-mwclient" to avoid confusion
with similarly named libraries using other languages.

The canonical import path for this package is

::

    import cgt.name/pkg/go-mwclient // imports "mwclient"

Relevant links:
 - GoDoc: http://godoc.org/cgt.name/pkg/go-mwclient
 - MediaWiki API docs: https://www.mediawiki.org/wiki/API:Main_page

Installation
============

::

    go get cgt.name/pkg/go-mwclient

API stability
==============
At this time the public API is not guaranteed to be stable. If I discover a
better way of doing something that breaks backwards compatibility, I will
break it.

Example
=======

::

    package main

    import (
        "fmt"
        "net/url"

        "cgt.name/pkg/go-mwclient"
    )

    func main() {
        // Make a Client object and specify the wiki's API URL and your user agent.
        w, err := mwclient.New("https://en.wikipedia.org/w/api.php", "Username's wikibot")
        if err != nil {
            panic(err)
        }

        // Log in.
        err = w.Login("USERNAME", "PASSWORD")
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
To the extent possible under law, the author(s) have dedicated all copyright and
related and neighboring rights to this software to the public domain worldwide.
This software is distributed without any warranty.

You should have received a copy of the CC0 Public Domain Dedication along with
this software. If not, see http://creativecommons.org/publicdomain/zero/1.0/.
