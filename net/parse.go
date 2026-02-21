package net

import (
	"encoding/json"
	"net/http"
)

func ParseData(r *http.Request) map[string]interface{} {
	decoder := r.Header.Get("Content-Type")
	var data map[string]interface{}
	if decoder == "application/json" {
		json.NewDecoder(r.Body).Decode(&data)
	} else if decoder == "form-data" {
		r.ParseForm()
		for key, value := range r.Form {
			data[key] = value[0]
		}
	} else {
		data := r.URL.Query()
		for key, value := range data {
			data[key] = value
		}
	}
	return data
}
