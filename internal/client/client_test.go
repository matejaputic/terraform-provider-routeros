// SPDX-License-Identifier: MPL-2.0
package client

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestConfiguration(t *testing.T) {
	for _, c := range []Config{{HostURL: "api://router", Timeout: time.Second}, {HostURL: "https://user:pass@router", Timeout: time.Second}, {HostURL: "https://router", CA: "bad", Timeout: time.Second}, {HostURL: "https://router", CA: "bad", Insecure: true, Timeout: time.Second}, {HostURL: "https://router"}} {
		if _, e := New(c); e == nil {
			t.Fatalf("accepted invalid config: %#v", c)
		}
	}
}
func TestRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		if !ok || u != "user" || p != "secret" {
			t.Error("missing authentication")
		}
		if r.URL.Query().Get(".id") != "*2" {
			t.Error("query not encoded")
		}
		if r.URL.EscapedPath() != "/rest/ip/address/a%2Fb" {
			t.Errorf("path %s", r.URL.EscapedPath())
		}
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()
	c, e := New(Config{HostURL: srv.URL, Username: "user", Password: "secret", Timeout: time.Second})
	if e != nil {
		t.Fatal(e)
	}
	var out map[string]bool
	if e = c.Request(context.Background(), "GET", "/ip/address", "a/b", url.Values{".id": []string{"*2"}}, nil, &out); e != nil || !out["ok"] {
		t.Fatalf("response %v, %v", out, e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if e = c.Request(ctx, "GET", "/ip/address", "", nil, nil, nil); e == nil {
		t.Fatal("ignored cancellation")
	}
}
func TestStatusAndRedirect(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Error("followed redirect with credentials") }))
	defer target.Close()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target.URL, http.StatusFound) }))
	defer srv.Close()
	c, _ := New(Config{HostURL: srv.URL, Timeout: time.Second})
	e := c.Request(context.Background(), "GET", "ip/address", "", nil, nil, nil)
	var se *StatusError
	if !errors.As(e, &se) || se.Code != 302 {
		t.Fatalf("expected typed 302, got %v", e)
	}
}
func TestResponseHandling(t *testing.T) {
	for _, tc := range []struct {
		body    string
		status  int
		wantErr bool
	}{{"", 204, true}, {"no json", 200, true}, {`{"ok":true}`, 200, false}, {"sensitive body", 401, true}} {
		t.Run(http.StatusText(tc.status)+tc.body, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(tc.status); w.Write([]byte(tc.body)) }))
			defer srv.Close()
			c, _ := New(Config{HostURL: srv.URL, Timeout: time.Second})
			var out map[string]any
			e := c.Request(context.Background(), "GET", "ip/address", "", nil, nil, &out)
			if (e != nil) != tc.wantErr {
				t.Fatalf("error %v", e)
			}
		})
	}
}
