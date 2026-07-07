package httpx

import (
	"encoding/json"
	"errors"
	"net/http"
)

type Response struct {
	Data  any    `json:"data,omitempty"`
	Error string `json:"error,omitempty"`
}

func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(Response{Data: data})
}

func Error(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(Response{Error: message})
}

func DecodeJSON(r *http.Request, dst any) error {
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return err
	}
	if decoder.More() {
		return errors.New("unexpected extra json content")
	}
	return nil
}

func DecodeJSONLoose(r *http.Request, dst any) error {
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(dst); err != nil {
		return err
	}
	if decoder.More() {
		return errors.New("unexpected extra json content")
	}
	return nil
}
