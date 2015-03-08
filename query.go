package mwclient

import (
	"fmt"

	"cgt.name/pkg/go-mwclient/params"
	"github.com/bitly/go-simplejson"
)

// Query provides a simple interface to deal with query continuations.
// A Query should be instantiated through the NewQuery method on the Client type.
// Once you have instantiated a Query, call the Next method to retrieve the
// first set of results from the API.
// If Next returns false, then either you have received all the results for the
// query or an error occured. If an error occurs, it will be available through
// the Err method.
// If Next returns true, then there are more results to be retrieved and another
// call to Next will retrieve the next results.
//
// The following example will retrieve all the pages that are in the category
// "Soap":
//	p := params.Values{
//		"list": "categorymembers",
//		"cmtitle": "Category:Soap",
//	}
//	q := w.NewQuery(p) // w being an instantiated Client
//	for q.Next() {
//		fmt.Println(q.Resp())
//	}
//	if q.Err() != nil {
//		// handle the error
//	}
// See https://www.mediawiki.org/wiki/API:Query for more details on how to
// query the MediaWiki API.
type Query struct {
	mwc    *Client
	params params.Values
	resp   *simplejson.Json
	err    error
}

// Err returns the first error encountered by the Next method.
func (q Query) Err() error {
	return q.err
}

// Resp returns the API response retrieved by the Next method.
func (q Query) Resp() *simplejson.Json {
	return q.resp
}

// NewQuery instantiates a new query with the given parameters.
func (w *Client) NewQuery(p params.Values) *Query {
	p.Set("action", "query")
	p.Set("continue", "")

	return &Query{
		mwc:    w,
		params: p,
		resp:   nil,
		err:    nil,
	}
}

// Next retrieves the next set of results from the API and makes them available
// through the Resp method. Next returns true if new results are available
// through Resp or false if there were no more results to request or if an
// error occurred.
func (q *Query) Next() (done bool) {
	if q.resp == nil {
		// first call to Next
		q.resp, q.err = q.mwc.Get(q.params)
		return q.err == nil
	}

	cont, ok := q.resp.CheckGet("continue")
	if !ok {
		return false
	}
	cm, err := cont.Map()
	if err != nil {
		q.err = fmt.Errorf("response processing error: %v", err)
		return false
	}
	for k, v := range cm {
		value, ok := v.(string)
		if !ok {
			q.err = fmt.Errorf("response processing error: %v", err)
			return false
		}
		q.params.Set(k, value)
	}

	q.resp, q.err = q.mwc.Get(q.params)
	return q.err == nil
}
