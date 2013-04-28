package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"sync"

	"github.com/tent/tent-client-go"
)

type request struct {
	req     *http.Request
	res     *http.Response
	reqBody *bytes.Buffer
	resBody *bytes.Buffer
}

type roundTripRecorder struct {
	roundTripper http.RoundTripper
	requests     []*request
	mtx          sync.Mutex
}

func (r *roundTripRecorder) RoundTrip(req *http.Request) (*http.Response, error) {
	reqBuf, resBuf := &bytes.Buffer{}, &bytes.Buffer{}
	if req.Body != nil {
		req.Body = readCloser{req.Body, io.TeeReader(req.Body, reqBuf)}
	}
	res, err := r.roundTripper.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	log := &request{req, res, reqBuf, resBuf}
	res.Body = readCloser{res.Body, io.TeeReader(res.Body, resBuf)}
	r.mtx.Lock()
	r.requests = append(r.requests, log)
	r.mtx.Unlock()
	return res, nil
}

type readCloser struct {
	io.Closer
	io.Reader
}

var excludeHeaders = map[string]bool{
	"Host":              true,
	"Content-Length":    true,
	"Transfer-Encoding": true,
	"Trailer":           true,
	"User-Agent":        true,
}

func requestMarkdown(r *request) string {
	buf := &bytes.Buffer{}

	// request headers
	buf.Write([]byte("```text\n"))
	buf.WriteString(r.req.Method)
	buf.WriteByte(' ')
	buf.WriteString(r.req.URL.RequestURI())
	buf.WriteString(" HTTP/1.1\n")
	r.req.Header.WriteSubset(buf, excludeHeaders)
	buf.Write([]byte("\n```\n"))

	// request body
	if r.req.Body != nil {
		buf.Write([]byte("```json\n"))
		json.Indent(buf, r.reqBody.Bytes(), "", "  ")
		buf.Write([]byte("\n```\n"))
	}

	// response headers
	buf.Write([]byte("\n```text\n"))
	buf.WriteString(r.res.Proto)
	buf.WriteByte(' ')
	buf.WriteString(r.res.Status)
	buf.Write([]byte("\n"))
	r.res.Header.WriteSubset(buf, excludeHeaders)
	buf.Write([]byte("\n```\n"))

	// response body
	resBody := r.resBody.Bytes()
	if len(resBody) > 0 {
		buf.Write([]byte("```json\n"))
		json.Indent(buf, resBody, "", "  ")
		buf.Write([]byte("\n```\n"))
	}

	return buf.String()
}

func getRequests() []*request {
	t := tent.HTTP.Transport.(*roundTripRecorder)
	reqs := t.requests
	t.requests = t.requests[:0]
	return reqs
}
