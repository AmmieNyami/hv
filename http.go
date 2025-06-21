package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
)

type Response struct {
	ErrorCode   int    `json:"error_code"`
	ErrorString string `json:"error_string"`
	Data        any    `json:"data"`
}

func WriteResponseHttp(response Response, code int, w http.ResponseWriter) {
	jsonBytes, _ := json.Marshal(response)
	json := string(jsonBytes)

	h := w.Header()
	h.Del("Content-Length")
	h.Set("Content-Type", "application/json; charset=utf-8")
	h.Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(code)
	fmt.Fprintln(w, json)
}

func Method(f http.HandlerFunc, acceptsMethods ...string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		acceptsMethods := append([]string(nil), acceptsMethods...)
		acceptsMethods = append(acceptsMethods, "OPTIONS")
		allowedMethodsString := ""
		for i, m := range acceptsMethods {
			allowedMethodsString += m
			if i != len(acceptsMethods)-1 {
				allowedMethodsString += ", "
			}
		}

		// Set CORS headers
		origin := r.Header.Get("Origin")
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		w.Header().Set("Access-Control-Allow-Methods", allowedMethodsString)
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			return
		}

		if !slices.Contains(acceptsMethods, r.Method) {
			w.Header().Set("Allow", allowedMethodsString)
			WriteResponseHttp(Response{
				ErrorCode:   -1,
				ErrorString: "Invalid method",
				Data:        nil,
			}, http.StatusMethodNotAllowed, w)
			return
		}

		f(w, r)
	}
}
