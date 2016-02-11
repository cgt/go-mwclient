package mwclient

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"cgt.name/pkg/go-mwclient/params"
)

func noSleep(d time.Duration) {
	return // the test monster under my bed is keeping me awake
}

func setup(handler func(w http.ResponseWriter, r *http.Request)) (*httptest.Server, *Client) {
	server := httptest.NewServer(http.HandlerFunc(handler))
	client, err := New(server.URL, "go-mwclient test")
	if err != nil {
		panic(err)
	}
	client.Maxlag.sleep = noSleep

	return server, client
}

func TestLoginToken(t *testing.T) {
	loginHandler := func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			panic("Bad HTTP form")
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")

		lgtoken := `b53b3ef3792bdaa1caff44fca1nb240756bc4eeb+\\`
		lgtokenExpected := `b53b3ef3792bdaa1caff44fca1nb240756bc4eeb+\`

		if q := r.URL.Query(); q.Get("action") == "query" && q.Get("meta") == "tokens" {
			// handle token request
			tokenTypes := strings.Split(q["type"][0], "|")
			foundLogin := false
			for _, t := range tokenTypes {
				if t == "login" {
					foundLogin = true
				}
			}
			if !foundLogin {
				t.Errorf("token requested, but not logintoken: %s", q["type"][0])
			}
			_, err := fmt.Fprintf(
				w,
				`{"batchcomplete":"","query":{"tokens":{"logintoken":"%s"}}}`,
				lgtoken)
			if err != nil {
				panic(err)
			}
		} else if r.Method == "POST" && r.PostFormValue("action") == "login" {
			// handle login request
			var errs []string
			fail := false
			if lgname := r.PostFormValue("lgname"); lgname != "username" {
				fail = true
				errs = append(errs,
					fmt.Sprintf(
						"expected \"username\" for lgname, got \"%s\"",
						lgname))
			}
			if lgpw := r.PostFormValue("lgpassword"); lgpw != "password" {
				fail = true
				errs = append(errs,
					fmt.Sprintf(
						"expected \"password\" for lgpassword, got \"%s\"",
						lgpw))
			}
			if lgtok := r.PostFormValue("lgtoken"); lgtok != lgtokenExpected {
				fail = true
				errs = append(errs,
					fmt.Sprintf(
						"expected \"%s\" for lgtoken, got \"%s\"",
						lgtokenExpected,
						lgtok))
			}

			if fail {
				if len(errs) > 1 {
					errMsg := strings.Join(errs, "; ")
					t.Error(errMsg)
				} else if len(errs) == 1 {
					t.Error(errs[0])
				} else {
					panic("TestLoginToken: fail == true, but empty errs")
				}
			}

			fmt.Fprint(
				w,
				`{"login":{"result":"Success","lguserid": 1,
				"lgusername":"username",
				"lgtoken":"32db2c4f4f5dca04a72e0a0913b27c25",
				"cookieprefix":"commonswiki",
				"sessionid":"vaggusqhjuh2m6u1rbchoaphm9ie19l"}}`)
		} else {
			t.Errorf("Unexpected request: %s", r.URL)
		}
	}

	server, client := setup(loginHandler)
	defer server.Close()

	if err := client.Login("username", "password"); err != nil {
		t.Errorf("Login() returned err: %v", err)
	}
}

func TestMaxlagOn(t *testing.T) {
	httpHandler := func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			panic("Bad HTTP form")
		}

		if r.Form.Get("maxlag") == "" {
			t.Fatalf("maxlag param not set. Params: %s", r.Form.Encode())
		}
	}

	server, client := setup(httpHandler)
	defer server.Close()

	p := params.Values{}
	client.Maxlag.On = true
	client.call(p, false)
}

func TestMaxlagOff(t *testing.T) {
	httpHandler := func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			panic("Bad HTTP form")
		}

		if r.Form.Get("maxlag") != "" {
			t.Fatalf("maxlag param set. Params: %s", r.Form.Encode())
		}
	}

	server, client := setup(httpHandler)
	defer server.Close()

	p := params.Values{}
	// Maxlag is off by default
	client.call(p, false)
}

func TestMaxlagRetryFail(t *testing.T) {
	httpHandler := func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			panic("Bad HTTP form")
		}
		if r.Form.Get("maxlag") == "" {
			t.Fatalf("maxlag param not set. Params: %s", r.Form.Encode())
		}

		header := w.Header()
		header.Set("X-Database-Lag", "10") // Value does not matter
		header.Set("Retry-After", "1")     // Value *does* matter
	}

	server, client := setup(httpHandler)
	defer server.Close()

	p := params.Values{}
	client.Maxlag.On = true
	_, err := client.call(p, false)
	if err != ErrAPIBusy {
		t.Fatalf("Expected ErrAPIBusy error from call(), got: %v", err)
	}
}

func TestAssertOff(t *testing.T) {
	httpHandler := func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			panic("Bad HTTP form")
		}

		if r.Form.Get("assert") != "" {
			t.Fatalf("Expected no assert param, found 'assert=%s'", r.Form.Get("assert"))
		}
	}

	server, client := setup(httpHandler)
	defer server.Close()

	p := params.Values{}
	// Assert should be off by default
	client.Get(p)
}

func TestAssertUser(t *testing.T) {
	httpHandler := func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			panic("Bad HTTP form")
		}

		if r.Form.Get("assert") == "" {
			t.Fatalf("Expected assert param, got none or empty")
		}
		if v := r.Form.Get("assert"); v != "user" {
			t.Fatalf("Expected 'assert=user', got 'assert=%s'", v)
		}
	}

	server, client := setup(httpHandler)
	defer server.Close()

	p := params.Values{}
	client.Assert = AssertUser
	client.Get(p)
}

func TestAssertBot(t *testing.T) {
	httpHandler := func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			panic("Bad HTTP form")
		}

		if r.Form.Get("assert") == "" {
			t.Fatalf("Expected assert param, got none or empty")
		}
		if v := r.Form.Get("assert"); v != "bot" {
			t.Fatalf("Expected 'assert=bot', got 'assert=%s'", v)
		}
	}

	server, client := setup(httpHandler)
	defer server.Close()

	p := params.Values{}
	client.Assert = AssertBot
	client.Get(p)
}
