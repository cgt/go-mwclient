package mwclient

import "net/url"

// GetPageID gets the pageid of a page specified by its name.
func (w *Client) GetPageID(pageName string) (string, error) {
	params := url.Values{
		"action": {"query"},
		"prop":   {"info"},
		"titles": {pageName},
	}

	resp, err := w.Get(params)
	if err != nil {
		return "", err
	}

	var id string
	for k, _ := range resp.GetPath("query", "pages").MustMap() {
		// There should only be one item in the map.
		id = k
	}
	return id, nil
}
