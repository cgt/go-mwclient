package mwclient

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func setup(handler func(w http.ResponseWriter, r *http.Request)) (*httptest.Server, *Client) {
	server := httptest.NewServer(http.HandlerFunc(handler))
	client := NewDefault(server.URL, "go-mwclient test")

	return server, client
}

func TestLogin(t *testing.T) {
	loginHandler := func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			t.Fatal("Bad test parameters")
		}

		// This is really difficult to read. Sorry...
		// Possible errors: NoName, NeedToken, WrongToken, EmptyPass, WrongPass, NotExists
		if r.PostForm.Get("lgname") == "" {
			fmt.Fprint(w, `{"login":{"result":"NoName"}}`)
		} else if r.PostForm.Get("lgtoken") == "" {
			fmt.Fprint(w, `{"login":{"result":"NeedToken","token":"7aaaf636d99d46cf2656561c5d099ad7","cookieprefix":"dawiki","sessionid":"e9bffbfa38636ac9f550b5d37fb25d80"}}`)
		} else {
			if r.PostForm.Get("lgtoken") != "7aaaf636d99d46cf2656561c5d099ad7" {
				fmt.Fprint(w, `{"login":{"result":"WrongToken"}}`)
			} else {
				if r.PostForm.Get("lgpassword") == "" {
					fmt.Fprint(w, `{"login":{"result":"EmptyPass"}}`)
				} else {
					if r.PostForm.Get("lgname") == "username" {
						if r.PostForm.Get("lgpassword") == "password" {
							// success
							fmt.Fprint(w, `{"login":{"result":"Success","lguserid":1,"lgusername":"username","lgtoken":"7aaaf636d99d46cf2656561c5d099ad7","cookieprefix":"dawiki","sessionid":"e9bffbfa38636ac9f550b5d37fb25d80"}}`)
						} else {
							fmt.Fprint(w, `{"login":{"result":"WrongPass"}}`)
						}
					} else {
						fmt.Fprint(w, `{"login":{"result":"NotExists"}}`)
					}
				}
			}
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
	}

	server, client := setup(loginHandler)
	defer server.Close()

	// good
	if err := client.Login("username", "password"); err != nil {
		t.Error("passed good login. expected no error, but received error:", err)
	}

	// bad
	if err := client.Login("", ""); err == nil {
		t.Error("passed empty user. expected NoName error, but no error received")
	} else {
		t.Log(err)
	}

	if err := client.Login("username", ""); err == nil {
		t.Error("passed empty password. expected EmptyPass error, but no error received")
	} else {
		t.Log(err)
	}

	if err := client.Login("badusername", ""); err == nil {
		t.Error("passed bad user. expected NotExists error, but no error received")
	} else {
		t.Log(err)
	}

	if err := client.Login("username", "badpassword"); err == nil {
		t.Error("passed bad password. expected WrongPass error, but no error received")
	} else {
		t.Log(err)
	}
}
