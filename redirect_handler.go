package main

import "net/http"

func RedirectHandler(httpHttpsRedirect bool, h http.Handler) http.Handler {
	return func(rw http.ResponseWriter, req *http.Request) {
		// if the request is HTTPS or we dont care about redirecting, then go ahead and pass the request along
		if req.Header.Get("X-Forwarded-Proto") == "https" || !httpHttpsRedirect {
			h(rw, req)
			return
		}

		http.Redirect(rw, req, "https://"+req.Host+req.URL.String(), http.StatusMovedPermanently)
	}
}
