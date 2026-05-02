// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tailscale.com/client/tailscale/v2"
)

type TestServer struct {
	t *testing.T

	Method string
	Path   string
	Body   *bytes.Buffer

	ResponseCode        int
	ResponseBody        interface{}
	ResponseByPath      map[string]interface{}   // optional: response body per method+path or path
	ResponseQueueByPath map[string][]interface{} // optional: per method+path, pop first element per request (so same path can return different bodies in sequence)
	ResponseCodeByPath  map[string]int           // optional: response status code per method+path or path; overrides ResponseCode for matching requests
}

func NewTestHarness(t *testing.T) (*tailscale.Client, *TestServer) {
	t.Helper()

	testServer := &TestServer{
		t: t,
	}

	mux := http.NewServeMux()
	mux.Handle("/", testServer)
	svr := &http.Server{
		Handler: mux,
	}

	// Start a listener on a random port
	listener, err := net.Listen("tcp", ":0")
	assert.NoError(t, err)

	go func() {
		_ = svr.Serve(listener)
	}()

	// When the test is over, close the server
	t.Cleanup(func() {
		assert.NoError(t, svr.Close())
	})

	baseURL := fmt.Sprintf("http://localhost:%v", listener.Addr().(*net.TCPAddr).Port)
	parsedBaseURL, err := url.Parse(baseURL)
	require.NoError(t, err)
	client := &tailscale.Client{
		BaseURL: parsedBaseURL,
		APIKey:  "not-a-real-key",
		Tailnet: "example.com",
	}

	return client, testServer
}

func (t *TestServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	t.Method = r.Method
	t.Path = r.URL.Path

	t.Body = bytes.NewBuffer([]byte{})
	_, err := io.Copy(t.Body, r.Body)
	assert.NoError(t.t, err)

	key := r.Method + " " + r.URL.Path
	var body interface{}
	if t.ResponseQueueByPath != nil {
		if q, ok := t.ResponseQueueByPath[key]; ok && len(q) > 0 {
			body = q[0]
			t.ResponseQueueByPath[key] = q[1:]
		}
	}
	if body == nil && t.ResponseByPath != nil {
		if b, ok := t.ResponseByPath[key]; ok {
			body = b
		} else if b, ok := t.ResponseByPath[r.URL.Path]; ok {
			body = b
		}
	}
	if body == nil {
		body = t.ResponseBody
	}
	code := t.ResponseCode
	if t.ResponseCodeByPath != nil {
		if c, ok := t.ResponseCodeByPath[key]; ok {
			code = c
		} else if c, ok := t.ResponseCodeByPath[r.URL.Path]; ok {
			code = c
		}
	}
	w.WriteHeader(code)
	switch b := body.(type) {
	case nil:
		// no body
	case []byte:
		_, err := w.Write(b)
		assert.NoError(t.t, err)
	default:
		assert.NoError(t.t, json.NewEncoder(w).Encode(b))
	}
}