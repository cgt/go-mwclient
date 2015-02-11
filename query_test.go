package mwclient

import (
	"fmt"
	"net/http"
	"testing"

	"cgt.name/pkg/go-mwclient/params"
)

func TestQuery(t *testing.T) {
	reqCount := 0 // incremented on each request to queryHandler

	queryHandler := func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			panic("Bad HTTP form")
		}

		if reqCount == 0 {
			if value := r.Form.Get("continue"); value != "" {
				t.Fatalf("'continue' value not empty in first req: continue=%s",
					value)
			}
			fmt.Fprintf(w, `{"continue":{"fkcontinue":"sendthisback","continue":"-||"}}`)
		} else if reqCount == 1 {
			if value := r.Form.Get("continue"); value != "-||" {
				t.Fatalf("'continue' key has different value than '-||': continue=%s",
					value)
			}
			if r.Form.Get("fkcontinue") != "sendthisback" {
				t.Fatalf("client did not return fkcontinue parameter")
			}
			fmt.Fprintf(w, "{}") // no continue element
		} else {
			panic("reqCount somehow got a different value than 0 or 1")
		}

		reqCount++
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
	}

	server, client := setup(queryHandler)
	defer server.Close()

	q := client.NewQuery(params.Values{})
	for q.Next() {
		continue
	}
	if err := q.Err(); err != nil {
		t.Fatalf("q.Err() != nil: %v", err)
	}
}
