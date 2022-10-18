package main

import (
	"fmt"
	"net/http"
)

func BootstrapListener() {
	http.ListenAndServe(":3333", &bootstrapHTTPHandler{})
}

type bootstrapHTTPHandler struct{}

func (h *bootstrapHTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("METHOD\n========\n%s\n\n", r.Method)
	fmt.Printf("BODY\n========\n%s\n", r.Body)
}
