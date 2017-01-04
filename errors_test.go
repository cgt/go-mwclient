package mwclient

import (
	"testing"

	"github.com/antonholmquist/jason"
)

type ErrorType int

const (
	Eror ErrorType = iota
	Warn
	None
)

func TestExtractAPIErrors(t *testing.T) {
	var errtests = []struct {
		jsonInput  []byte
		testType   ErrorType
		warnAmount int
	}{
		{
			[]byte(`{"servedby":"mw1197","error":{"code":"nouser",
			"info":"The user parameter must be set"}}`),
			Eror,
			0,
		},
		{
			[]byte(`{"servedby":"mw1204","error":{"code":"notoken",
			"info":"The token parameter must be set"}}`),
			Eror,
			0,
		},
		{
			[]byte(`{"batchcomplete":true,"warnings":{"query":{"warnings":"Unrecognized value for parameter \"list\": invalidmodule."}}}`),
			Warn,
			1,
		},
		{
			[]byte(`{"query":{"pages":{"709377":{"pageid":709377,"ns":2,"title":
			"Bruger:Cgtdk","contentmodel":"wikitext","pagelanguage":"da",
			"touched":"2014-01-27T10:06:57Z","lastrevid":7257075,"counter":"",
			"length":695}}}}`),
			None,
			0,
		},
	}

	for i, errtest := range errtests {
		j, err := jason.NewObjectFromBytes(errtest.jsonInput)
		if err != nil {
			panic("Invalid test data: bad JSON input")
		}

		err = extractAPIErrors(j)

		switch errtest.testType {
		case Eror:
			if _, ok := err.(APIError); !ok {
				t.Errorf("(test:%d) expected APIError, got: %v", i, err)
			}
		case Warn:
			e, ok := err.(APIWarnings)
			if !ok {
				t.Errorf("(test:%d) expected APIWarnings, got: %v", i, err)
			}
			if len(e) != errtest.warnAmount {
				t.Errorf("(test:%d) expected %d warnings, got %d: %v", i,
					errtest.warnAmount, len(e), err)
			}
		case None:
			if err != nil {
				t.Errorf("(test:%d) expected nil, got !nil: %v", i, err)
			}
		}
	}
}
