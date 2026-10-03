// SPDX-License-Identifier: MPL-2.0
package client

import (
	"context"
	"encoding/pem"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestVersionOrdering(t *testing.T) {
	versions := []string{"7.25beta2", "7.25beta10", "7.25rc1", "7.25", "7.25.1"}
	for i := 1; i < len(versions); i++ {
		a, e := ParseVersion(versions[i-1])
		if e != nil {
			t.Fatal(e)
		}
		b, e := ParseVersion(versions[i])
		if e != nil {
			t.Fatal(e)
		}
		n, e := a.Compare(b)
		if e != nil || n >= 0 {
			t.Fatal("incorrect version ordering")
		}
	}
	a, _ := ParseVersion("7.25_ab434")
	b, _ := ParseVersion("7.25_ab435")
	n, e := a.Compare(b)
	if e != nil || n >= 0 {
		t.Fatal("nightly ordering")
	}
	b, _ = ParseVersion("7.25beta5")
	if _, e = a.Compare(b); e == nil {
		t.Fatal("invented nightly/release ordering")
	}
	for _, raw := range []string{"garbage", "7.25beta", "7.25_abx", "7.24.5 secret", "7.999999999999999999999999"} {
		if _, e := ParseVersion(raw); e == nil {
			t.Fatal("accepted invalid version")
		}
	}
	if ok, e := CheckRuntimeVersion("7.24.5 (stable)"); !ok || e != nil {
		t.Fatal(e)
	}
	if _, e := CheckRuntimeVersion("6.49.1"); e == nil {
		t.Fatal("unsupported major accepted")
	}
}
func TestVerifiedTLSAndExplicitOptIn(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{}`)) }))
	defer srv.Close()
	configs := []Config{{HostURL: srv.URL, Timeout: time.Second}, {HostURL: srv.URL, Timeout: time.Second, Insecure: true}, {HostURL: srv.URL, Timeout: time.Second, CA: string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: srv.Certificate().Raw}))}}
	for i, cfg := range configs {
		c, e := New(cfg)
		if e != nil {
			t.Fatal(e)
		}
		e = c.Request(context.Background(), "GET", "system/resource", "", nil, nil, nil)
		c.Close()
		if (e != nil) != (i == 0) {
			t.Fatalf("TLS case %d: %v", i, e)
		}
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type brokenBody struct{}

func (brokenBody) Read([]byte) (int, error) { return 0, errors.New("private response content") }
func (brokenBody) Close() error             { return nil }
func TestBoundedResponsesAndRedactedReadErrors(t *testing.T) {
	for _, body := range []io.ReadCloser{brokenBody{}, io.NopCloser(strings.NewReader(strings.Repeat("x", (4<<20)+1)))} {
		c, _ := New(Config{HostURL: "http://localhost", Timeout: time.Second})
		c.http.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) { return &http.Response{StatusCode: 200, Body: body}, nil })
		err := c.Request(context.Background(), "GET", "ip/address", "", nil, nil, nil)
		if err == nil || strings.Contains(err.Error(), "private") {
			t.Fatal("unbounded/leaky response")
		}
	}
}
func TestCancellationAndUnsafeMutationPaths(t *testing.T) {
	c, _ := New(Config{HostURL: "http://localhost", Password: "sentinel-password", Timeout: time.Second})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := c.Request(ctx, "GET", "ip/address", "", nil, nil, nil)
	if !errors.Is(err, context.Canceled) || strings.Contains(err.Error(), "sentinel") {
		t.Fatal("lost cancellation or secret leak")
	}
	for _, p := range []string{"../system/reset", "ip//address", "ip/address?secret", "ip/address#fragment"} {
		if e := c.Request(context.Background(), "GET", p, "", nil, nil, nil); e == nil {
			t.Fatal("unsafe path")
		}
	}
	for _, m := range []string{http.MethodPatch, http.MethodDelete} {
		if e := c.Request(context.Background(), m, "ip/address", "", nil, nil, nil); e == nil {
			t.Fatal("collection mutation without ID")
		}
	}
}
