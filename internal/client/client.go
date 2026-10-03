// SPDX-License-Identifier: MPL-2.0
package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Config struct {
	HostURL, Username, Password, CA string
	Insecure                        bool
	Timeout                         time.Duration
}
type Client struct {
	base               *url.URL
	http               *http.Client
	username, password string
	Version            string
	Packages           []string // populated once before sharing the configured client
}
type StatusError struct{ Code int }

// RequestError retains cancellation/timeout identity without exposing URL, bodies or credentials.
type RequestError struct{ cause error }

func (e *RequestError) Error() string { return "RouterOS request failed (transport or context error)" }
func (e *RequestError) Unwrap() error { return e.cause }
func (c *Client) Close()              { c.http.CloseIdleConnections() }

func (e *StatusError) Error() string { return fmt.Sprintf("RouterOS HTTP status %d", e.Code) }
func New(c Config) (*Client, error) {
	u, e := url.Parse(c.HostURL)
	if e != nil {
		return nil, fmt.Errorf("invalid hosturl")
	}
	if (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.Hostname() == "" || u.Opaque != "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return nil, fmt.Errorf("hosturl must be an http(s) URL without credentials, query or fragment; api/apis are not supported")
	}
	if c.Insecure && c.CA != "" {
		return nil, fmt.Errorf("ca_certificate and insecure are mutually exclusive")
	}
	if c.Timeout <= 0 {
		return nil, fmt.Errorf("rest_timeout must be positive")
	}
	// Own the transport; a process-wide custom DefaultTransport must not panic
	// configuration or alter TLS/credential behavior for provider aliases.
	t := &http.Transport{Proxy: http.ProxyFromEnvironment, DialContext: (&net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}).DialContext, ForceAttemptHTTP2: true, MaxIdleConns: 100, IdleConnTimeout: 90 * time.Second, TLSHandshakeTimeout: 10 * time.Second, ExpectContinueTimeout: time.Second}
	t.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12, InsecureSkipVerify: c.Insecure} // explicit user opt-in
	if c.CA != "" {
		roots, e := x509.SystemCertPool()
		if e != nil {
			roots = x509.NewCertPool()
		}
		if !roots.AppendCertsFromPEM([]byte(c.CA)) {
			return nil, fmt.Errorf("ca_certificate is not valid PEM")
		}
		t.TLSClientConfig.RootCAs = roots
	}
	u.RawPath = ""
	u.Path = strings.TrimRight(u.Path, "/")
	if !strings.HasSuffix(u.Path, "/rest") {
		u.Path += "/rest"
	}
	return &Client{base: u, username: c.Username, password: c.Password, http: &http.Client{Transport: t, Timeout: c.Timeout, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}
func (c *Client) Request(ctx context.Context, method, path, id string, query url.Values, body any, out any) error {
	if ctx == nil {
		return fmt.Errorf("RouterOS operation context is required")
	}
	if id == "." || id == ".." || strings.ContainsAny(id, "\r\n\x00") {
		return fmt.Errorf("invalid RouterOS item ID")
	}
	for _, segment := range strings.Split(strings.Trim(path, "/"), "/") {
		if segment == "" || segment == "." || segment == ".." || strings.ContainsAny(segment, "?#\\\\") {
			return fmt.Errorf("invalid RouterOS resource path")
		}
	}
	if (method == http.MethodPatch || method == http.MethodDelete) && id == "" {
		return fmt.Errorf("RouterOS mutation requires an ID")
	}
	u := *c.base
	u.Path += "/" + strings.Trim(path, "/")
	u.RawPath = ""
	if id != "" {
		// RouterOS requires literal '*' in internal IDs; it does not decode %2A.
		// Keep every other special character escaped as a single path segment.
		u.RawPath = u.EscapedPath() + "/" + strings.ReplaceAll(url.PathEscape(id), "%2A", "*")
		u.Path += "/" + id
	}
	u.RawQuery = query.Encode()
	var b []byte
	var err error
	if body != nil {
		b, err = json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode RouterOS request")
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, u.String(), bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("construct RouterOS request")
	}
	req.SetBasicAuth(c.username, c.password)
	req.Header.Set("Content-Type", "application/json")
	res, err := c.http.Do(req)
	if err != nil {
		cause := err
		var urlError *url.Error
		if errors.As(err, &urlError) {
			cause = urlError.Err
		}
		return &RequestError{cause: cause}
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return &StatusError{Code: res.StatusCode}
	}
	data, err := io.ReadAll(io.LimitReader(res.Body, (4<<20)+1))
	if err != nil {
		return fmt.Errorf("read RouterOS response")
	}
	if len(data) > 4<<20 {
		return fmt.Errorf("RouterOS response exceeds size limit")
	}
	if out == nil {
		return nil
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return fmt.Errorf("empty RouterOS response where JSON is required")
	}
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return fmt.Errorf("null RouterOS response where JSON object/array is required")
	}
	if err = json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("invalid RouterOS JSON response")
	}
	return nil
}
