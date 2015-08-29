package mwclient

import "net/http"

// DumpCookies exports the cookies stored in the client.
func (w *Client) DumpCookies() []*http.Cookie {
	return w.httpc.Jar.Cookies(w.apiURL)
}

// LoadCookies imports cookies into the client.
func (w *Client) LoadCookies(cookies []*http.Cookie) {
	w.httpc.Jar.SetCookies(w.apiURL, cookies)
}
