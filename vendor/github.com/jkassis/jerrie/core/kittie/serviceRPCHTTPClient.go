package kittie

import (
	"bytes"
	"encoding/json"
	"errors"
	fmt "fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/google/uuid"
	"github.com/jkassis/jerrie/core"
)

// ServiceRPCHTTPClient is a client for accessing JSONRPCAPIs
type ServiceRPCHTTPClient struct {
	HTTPAPIRootURLs []string
	HTTPClient      *core.HTTPClient
}

// RandHTTPAPIRootURL returns a random root url
func (c *ServiceRPCHTTPClient) RandHTTPAPIRootURL() string {
	return c.HTTPAPIRootURLs[core.RandInt(0, len(c.HTTPAPIRootURLs))]
}

// Post sends an http request
func (c *ServiceRPCHTTPClient) Post(fn string, rpcReqBody []byte) ([]byte, error) {
	// encode he RPCReq
	rpcReq := &RPCReq{
		UUID: uuid.New(),
		Fn:   fn,
		Body: (*json.RawMessage)(&rpcReqBody),
	}
	req, err := json.Marshal(rpcReq)
	if err != nil {
		return nil, err
	}

	// make the httpReq
	httpReq, err := http.NewRequest("POST", c.RandHTTPAPIRootURL(), bytes.NewBuffer(req))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	// get the response
	var httpRes *http.Response
	httpRes, err = c.HTTPClient.Do(httpReq)

	// clear the response
	defer func() {
		if httpRes != nil && httpRes.Body != nil {
			io.Copy(ioutil.Discard, httpRes.Body)
			httpRes.Body.Close()
		}
	}()

	// http protocol error?
	if err != nil {
		return nil, err
	}

	// got a response?
	if httpRes == nil || httpRes.Body == nil {
		// no. it's required
		err = errors.New("empty callback response")
		core.Log.Info(err)
		return nil, err
	}

	// yes. copy it out
	resBuf := bytes.NewBuffer(nil)
	io.Copy(resBuf, httpRes.Body)

	// was it ok?
	switch httpRes.StatusCode {
	case http.StatusInternalServerError:
		return nil, errors.New(string(resBuf.Bytes()))
	case http.StatusOK:
		return resBuf.Bytes(), nil
	default:
		return resBuf.Bytes(), fmt.Errorf("Got unexpected status code %d", httpRes.StatusCode)
	}
}
