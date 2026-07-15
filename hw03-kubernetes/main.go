package main

import "net/http"

func main() {
	mux := http.NewServeMux()
	ok := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status": "OK"}`))
	}
	mux.HandleFunc("GET /health", ok)
	mux.HandleFunc("GET /health/", ok)
	http.ListenAndServe(":8000", mux)
}
