package mwclient

import (
	"fmt"
	"net/http"
	"testing"
)

func TestGetToken(t *testing.T) {
	resp := `{"batchcomplete":"","query":{"tokens":{"csrftoken":"+\\"}}}`
	httpHandler := func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			panic("Bad HTTP form")
		}

		if v := r.Form.Get("action"); v != "query" {
			t.Fatalf("action != query: action=%s", v)
		}
		if v := r.Form.Get("meta"); v != "tokens" {
			t.Fatalf("meta != tokens: meta=%s", v)
		}
		if v := r.Form.Get("type"); v != CSRFToken {
			t.Fatalf("meta != %s: meta=%s", CSRFToken, v)
		}

		fmt.Fprint(w, resp)
	}

	server, client := setup(httpHandler)
	defer server.Close()

	token, err := client.GetToken(CSRFToken)
	if err != nil {
		t.Fatalf("token request failed: %v", err)
	}
	if token != "+\\" {
		t.Fatalf("received token does not match sent token")
	}
}

func TestGetCachedToken(t *testing.T) {
	client, err := New("http://example.com", "go-mwclient test")
	if err != nil {
		panic(err)
	}
	client.Tokens[CSRFToken] = "tokenvalue"
	gotToken, err := client.GetToken(CSRFToken)
	if err != nil {
		panic(err)
	}
	if gotToken != client.Tokens[CSRFToken] {
		t.Fatalf("got token does not match manually cached token: CSRFToken=%s",
			gotToken)
	}
}
