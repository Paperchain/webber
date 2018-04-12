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

func (r *Request) Do() (*Response, error) {
	netTransport := &http.Transport{
		Dial: (&net.Dialer{
			Timeout: defaultTimeoutInMs * time.Second,
		}).Dial,
		TLSHandshakeTimeout: defaultTimeoutInMs * time.Second,
	}

	client := &http.Client{
		Timeout:   time.Second * defaultTimeoutInMs,
		Transport: netTransport,
	}

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
	} else {
		req.Header.Set("Content-Type", ContentTypeApplicationJSON)
	}

	httpResponse, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer httpResponse.Body.Close()

	res := NewResponse(httpResponse)

	if contentEncoding := httpResponse.Header.Get("Content-Encoding"); contentEncoding == "gzip" || contentEncoding == "agzip" {
		reader, err := gzip.NewReader(httpResponse.Body)
		defer reader.Close()
		if err != nil {
			return nil, err
		}

		res.Data, err = ioutil.ReadAll(reader)
		if err != nil {
			return nil, err
		}
	} else {
		res.Data, err = ioutil.ReadAll(httpResponse.Body)
		if err != nil {
			return nil, err
		}
	}

	if !(httpResponse.StatusCode >= 200 && httpResponse.StatusCode < 300) {
		return res, errUnsuccessfulResponse
	}

	return res, nil
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
