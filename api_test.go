package main

import (
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestUpdateItemMovesOnlyWhenSectionChanges(t *testing.T) {
	var paths []string
	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("Authorization") != "Bearer secret" {
			t.Fatal("authorization bearer absent")
		}
		paths = append(paths, r.Method+" "+r.URL.Path)
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{}`)),
			Header:     make(http.Header),
		}, nil
	})}

	c := &client{baseURL: "http://koffan.test/api/v1", token: "secret", http: httpClient}
	if err := c.updateItem(7, "Pommes", 2, 4, 3); err != nil {
		t.Fatal(err)
	}

	want := []string{"PUT /api/v1/items/7", "POST /api/v1/items/7/move"}
	if !reflect.DeepEqual(paths, want) {
		t.Fatalf("appels = %v, attendu %v", paths, want)
	}
}
