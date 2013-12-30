package mwclient

import (
	"fmt"
	"net/url"
)

// GetToken returns a specified token (and an error if this is not possible).
// If the token is not already available in the Wiki.Tokens map,
// it will attempt to retrieve it via the API.
// tokenName should be "edit" (or whatever), not "edittoken".
func (w *Wiki) GetToken(tokenName string) (string, error) {
	if _, ok := w.Tokens[tokenName]; ok {
		return w.Tokens[tokenName], nil
	}

	parameters := url.Values{
		"action": {"tokens"},
		"type":   {tokenName},
	}

	resp, err, apiok := ErrorCheck(w.Get(parameters))
	if err != nil {
		return "", err
	}
	if !apiok {
		// Check for errors
		if err, ok := resp.CheckGet("error"); ok {
			newError := fmt.Errorf("%s: %s", err.Get("code").MustString(), err.Get("info").MustString())
			return "", newError
		}

		// Check for warnings
		if warnings, ok := resp.CheckGet("warnings"); ok {
			newError := fmt.Errorf(warnings.GetPath("tokens", "*").MustString())
			return "", newError
		}
	}

	token, err := resp.GetPath("tokens", tokenName+"token").String()
	if err != nil {
		// This really shouldn't happen.
		return "", fmt.Errorf("Error occured while converting token to string: %s", err)
	}
	w.Tokens[tokenName] = token
	return token, nil
}
