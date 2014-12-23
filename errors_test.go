package mwclient

import (
	"testing"

	"cgt.name/pkg/multierror"
	"github.com/bitly/go-simplejson"
)

func TestExtractAPIErrors(t *testing.T) {
	var errtests = []struct {
		jsonInput []byte
		errAmount uint8
	}{
		{
			[]byte(`{"servedby":"mw1197","error":{"code":"nouser","info":"The user parameter must be set"}}`),
			1,
		},
		{
			[]byte(`{"servedby":"mw1204","error":{"code":"notoken","info":"The token parameter must be set"}}`),
			1,
		},
		{
			[]byte(`{"warnings":{"tokens":{"*":"Action 'deleteglobalaccount' is not allowed for the current user"}},"tokens":[]}`),
			1,
		},
		{
			[]byte(`{"warnings":{"tokens":{"*":"Action 'deleteglobalaccount' is not allowed for the current user\nAction 'setglobalaccountstatus' is not allowed for the current user"}},"tokens":[]}`),
			2,
		},
		{
			[]byte(`{"query":{"pages":{"709377":{"pageid":709377,"ns":2,"title":"Bruger:Cgtdk","contentmodel":"wikitext","pagelanguage":"da","touched":"2014-01-27T10:06:57Z","lastrevid":7257075,"counter":"","length":695}}}}`),
			0,
		},
	}

	for i, errtest := range errtests {
		js, err := simplejson.NewJson(errtest.jsonInput)
		if err != nil {
			t.Fatalf("Invalid JSON for test %d: %s", i, err)
		}

		_, err = extractAPIErrors(js)
		if errtest.errAmount > 0 {
			if uint8(len(err.(*multierror.MultiError).Errors)) != errtest.errAmount {
				t.Errorf("(test:%d) %d errors returned, expected %d: %s", i, len(err.(*multierror.MultiError).Errors), errtest.errAmount, err)
			} else {
				t.Logf("(test:%d) OK", i)
			}
		} else {
			if err != nil {
				t.Errorf("(test:%d) >0 errors returned, expected nil: %v", i, err)
			} else {
				t.Logf("(test:%d) OK", i)
			}
		}
	}
}
