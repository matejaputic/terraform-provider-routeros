// SPDX-License-Identifier: MPL-2.0
package client

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestOperationContextNotStoredAndDefaultTransportNotAssumed(t *testing.T) {
	previous := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Error("global transport reused")
		return nil, errors.New("not allowed")
	})
	defer func() { http.DefaultTransport = previous }()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/slow" {
			<-r.Context().Done()
			return
		}
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()
	c, e := New(Config{HostURL: srv.URL, Timeout: time.Second})
	if e != nil {
		t.Fatal(e)
	}
	defer c.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if e := c.Request(ctx, "GET", "slow", "", nil, nil, nil); !errors.Is(e, context.DeadlineExceeded) {
		t.Fatal("operation deadline lost")
	}
	if e := c.Request(context.Background(), "GET", "ok", "", nil, nil, nil); e != nil {
		t.Fatal("stored canceled operation context")
	}
}
