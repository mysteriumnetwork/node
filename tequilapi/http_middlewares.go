package tequilapi

import "net/http"

type corsHandler struct {
	originalHandler http.Handler
}

func (wrapper corsHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	resp.Header().Set("Access-Control-Allow-Origin", "*")
	resp.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	wrapper.originalHandler.ServeHTTP(resp, req)
}

//ApplyCors wraps original handler by adding cors headers to response BEFORE original ServeHTTP method is called
func ApplyCors(original http.Handler) http.Handler {
	return corsHandler{original}
}
