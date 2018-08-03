package mwclient

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"cgt.name/pkg/go-mwclient/params"
)

func TestEdit(t *testing.T) {
	resp := `{"edit":{"result":"Success","pageid":42,"title":"PAGE",
	"contentmodel":"wikitext","oldrevid":7936766,"newrevid":7950155,
	"newtimestamp":"2015-02-12T17:13:01Z"}}`

	httpHandler := func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			panic("Bad HTTP form")
		}

		if r.Method != "POST" {
			t.Fatalf("edit requests must be posted. Method: %v", r.Method)
		}
		if v := r.Form.Get("action"); v != "edit" {
			t.Fatalf("action != edit: action=%s", v)
		}
		if v := r.Form.Get("token"); v != "VALIDTOKEN" {
			t.Fatalf("token != VALIDTOKEN: token=%s", v)
		}

		fmt.Fprint(w, resp)
	}

	server, client := setup(httpHandler)
	defer server.Close()

	client.Tokens[CSRFToken] = "VALIDTOKEN"
	err := client.Edit(params.Values{})
	if err != nil {
		t.Fatalf("edit request returned error: %v", err)
	}
}

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

func TestEditCaptchaImage(t *testing.T) {
	resp := `{
	"edit": {
		"captcha": {
			"type": "image",
			"mime": "image/png",
			"id": "1",
			"url": "CAPTCHAURL"
		},
		"result": "Failure"
	}
}`
	httpHandler := func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, resp)
	}

	server, client := setup(httpHandler)
	defer server.Close()

	client.Tokens["csrf"] = "doesn't matter"
	err := client.Edit(params.Values{})
	if err == nil {
		t.Fatalf("error not detected despite edit failure")
	}
	e, ok := err.(CaptchaError)
	if !ok {
		t.Fatalf("error returned, but is not of type CaptchaError: %T", err)
	}

	// Check that CaptchaError fields are correct
	if e.ID != "1" {
		t.Errorf("CaptchaError.ID is not \"1\": ID == %s", e.ID)
	}
	if e.Mime != "image/png" {
		t.Errorf("CaptchaError.Mime is not \"image/png\": Mime == %s", e.Mime)
	}
	if e.Type != "image" {
		t.Errorf("CaptchaError.Type is not \"image\": Type == %s", e.Type)
	}
	if e.URL != "CAPTCHAURL" {
		t.Errorf("CaptchaError.URL is not \"CAPTCHAURL\": URL == %s", e.URL)
	}
	if e.Question != "" {
		t.Errorf("e.Question is not empty string despite image captcha: %s",
			e.Question)
	}
}

func TestEditCaptchaMath(t *testing.T) {
	resp := `{
    "edit": {
        "captcha": {
            "type": "math",
            "mime": "text/tex",
            "id": "1",
            "question": "84 - 3 = "
        },
        "result": "Failure"
    }
}`
	httpHandler := func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, resp)
	}

	server, client := setup(httpHandler)
	defer server.Close()

	client.Tokens["csrf"] = "doesn't matter"
	err := client.Edit(params.Values{})
	if err == nil {
		t.Fatalf("error not detected despite edit failure")
	}
	e, ok := err.(CaptchaError)
	if !ok {
		t.Fatalf("error returned, but is not of type CaptchaError: %T", err)
	}

	// Check that CaptchaError fields are correct
	if e.ID != "1" {
		t.Errorf("CaptchaError.ID is not \"1\": ID == %s", e.ID)
	}
	if e.Mime != "text/tex" {
		t.Errorf("CaptchaError.Mime is not \"text/tex\": Mime == %s", e.Mime)
	}
	if e.Type != "math" {
		t.Errorf("CaptchaError.Type is not \"math\": Type == %s", e.Type)
	}
	if e.Question != "84 - 3 = " {
		t.Errorf("CaptchaError.Question is not \"84  - 3 = \": Question == %s",
			e.Question)
	}
	if e.URL != "" {
		t.Errorf("e.URL is not empty string despite math captcha: %s", e.URL)
	}
}

func TestHandleGetPagesReturnsPagesEvenIfWarning(t *testing.T) {
	jsonResp := []byte(`
{
  "warnings": {
    "main": {
      "warnings": "Unrecognized parameter: foo."
    }
  },
  "batchcomplete": true,
  "query": {
    "pages": [
      {
        "pageid": 15580374,
        "ns": 0,
        "title": "Main Page",
        "revisions": [
          {
            "timestamp": "2018-06-26T14:19:36Z",
            "slots": {
              "main": {
                "contentmodel": "wikitext",
                "contentformat": "text/x-wiki",
                "content": "...snip..."
              }
            }
          }
        ]
      }
    ]
  }
}
`)

	var resp getPagesResponse
	err := json.Unmarshal(jsonResp, &resp)
	if err != nil {
		panic(err)
	}
	titles := []string{"Main Page"}

	pages, err := handleGetPages(titles, resp)

	if pages == nil {
		t.Error("expected non-nil pages, got nil")
	}
	if err == nil {
		t.Error("expected non-nil error, got nil")
	}
}

func TestHandleGetPagesReturnsBothWarningsAndPageErrors(t *testing.T) {
	jsonResp := []byte(`
{
  "warnings": {
    "main": {
      "warnings": "Unrecognized parameter: foo."
    }
  },
  "batchcomplete": true,
  "query": {
    "pages": [
      {
        "ns": 0,
        "title": "DoesNotExist",
        "missing": true
      }
    ]
  }
}
`)
	var resp getPagesResponse
	err := json.Unmarshal(jsonResp, &resp)
	if err != nil {
		panic(err)
	}
	titles := []string{"DoesNotExist"}

	pages, err := handleGetPages(titles, resp)

	if err == nil {
		t.Error("expected error, got nil")
	} else if _, ok := err.(APIWarnings); !ok {
		t.Errorf("expected APIWarnings error, got %#v", err)
	}

	if pages == nil {
		t.Error("expected non-nil pages, got nil")
	} else {
		page := pages[titles[0]]
		if page.Error == nil {
			t.Error("expected page-specific error, got nil")
		}
	}
}
