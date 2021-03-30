package webber

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	userAgentString    = "paperchain-webber/0.1.0 ( info@paperchain.io )"
	defaultTimeoutInMs = 1000

	ContentTypeApplicationJSON = "application/json"
	ContentTypeFormEncoded     = "application/x-www-form-urlencoded"
)

var (
	errUnsuccessfulResponse = errors.New("The response was unsuccessful")
)

type (
	Request struct {
		URI         string
		Headers     map[string]string
		Method      string
		ContentType string

		TimeoutInMs int
		EnableGzip  bool

		params  map[string]string
		payload interface{}
	}
	Response struct {
		*http.Response
		Data []byte
	}
)

func NewResponse(r *http.Response) *Response {
	return &Response{r, nil}
}

func (r *Request) withURI(uri string) *Request {
	r.URI = uri
	return r
}

func (r *Request) Get(params map[string]string) (*Response, error) {
	r.Method = http.MethodGet
	r.params = params

	return r.Do()
}

func (r *Request) Post(payload interface{}) (*Response, error) {
	r.Method = http.MethodPost
	r.payload = payload

	return r.Do()
}

var (
	netTransport = &http.Transport{
		Dial: (&net.Dialer{
			Timeout: defaultTimeoutInMs * time.Second,
		}).Dial,
		TLSHandshakeTimeout: defaultTimeoutInMs * time.Second,
		MaxIdleConns:        20,
		MaxIdleConnsPerHost: 20,
	}

	client = &http.Client{
		Timeout:   time.Second * defaultTimeoutInMs,
		Transport: netTransport,
	}
)

func (r *Request) Do() (*Response, error) {
	u, err := r.getURLString()
	if err != nil {
		return nil, err
	}

	body, err := prepareRequestBody(r.payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(r.Method, u, body)
	if err != nil {
		return nil, err
	}

	for k, v := range r.Headers {
		req.Header.Set(k, v)
	}
	req.Header.Set("User-Agent", userAgentString)

	if r.ContentType != "" {
		req.Header.Set("Content-Type", r.ContentType)
	}

	httpResponse, err := client.Do(req)
	res := NewResponse(httpResponse)
	if err != nil || !(httpResponse.StatusCode >= 200 && httpResponse.StatusCode < 300) {
		return res, errUnsuccessfulResponse
	}

	return res, nil
}

func (r *Response) Read(uncompress bool) error {
	var err error

	// If a request fails gets rejected early it may have a nil *http.Response
	// object in that case any *http.Response attributes like request Header,
	// Body etc. won't be available.
	if r.Response == nil {
		return nil
	}

	defer r.Body.Close()
	if contentEncoding := r.Header.Get("Content-Encoding"); uncompress && (contentEncoding == "gzip" || contentEncoding == "agzip") {
		reader, err := gzip.NewReader(r.Body)
		defer reader.Close()
		if err != nil {
			return err
		}

		r.Data, err = ioutil.ReadAll(reader)
		if err != nil {
			return err
		}
	} else {
		r.Data, err = ioutil.ReadAll(r.Body)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *Request) getURLString() (string, error) {
	pu, err := url.Parse(r.URI)
	if err != nil {
		return "", err
	}

	if r.params != nil && len(r.params) > 0 {
		q := pu.Query()
		for k, v := range r.params {
			q.Add(k, v)
		}
		pu.RawQuery = q.Encode()
	}

	return pu.String(), nil
}

func prepareRequestBody(b interface{}) (io.Reader, error) {
	switch b.(type) {
	case string:
		// treat is as text
		return strings.NewReader(b.(string)), nil
	case io.Reader:
		// treat is as text
		return b.(io.Reader), nil
	case []byte:
		//treat as byte array
		return bytes.NewReader(b.([]byte)), nil
	case nil:
		return nil, nil
	default:
		// try to jsonify it
		j, err := json.Marshal(b)
		if err == nil {
			return bytes.NewReader(j), nil
		}
		return nil, err
	}
}
