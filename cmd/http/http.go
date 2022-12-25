package http

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func Post(reqURL, contentType, body string) (resBodyString string, err error) {
	// make the request
	var req *http.Request
	{
		req, err = http.NewRequestWithContext(context.Background(), "POST", reqURL, strings.NewReader(body))
		if err != nil {
			return "", fmt.Errorf("HttpPost: could not create request: %v", err)
		}
		req.Close = true
		req.Header.Set("Content-Type", "application/json")
	}

	var resBody []byte
	{
		var res *http.Response
		res, err = http.DefaultClient.Do(req)
		if err != nil {
			return "", fmt.Errorf("http.Post error: %v", err)
		}
		defer res.Body.Close()

		resBody, err = io.ReadAll(res.Body)
		if err != nil {
			return "", fmt.Errorf("error while reading res.Body from POST: %s: %v", reqURL, err)
		}

		if res.StatusCode != http.StatusOK {
			return "", fmt.Errorf("POST to %s: %d [%s]: %s", reqURL, res.StatusCode, res.Status, resBody)
		}
	}

	return string(resBody), nil
}

func Get(reqURL, contentType string) (resBodyString string, err error) {
	// make the request
	var req *http.Request
	{
		req, err = http.NewRequestWithContext(context.Background(), "GET", reqURL, nil)
		if err != nil {
			return "", fmt.Errorf("HTTPGet: could not create request: %v", err)
		}
		req.Close = true
		req.Header.Set("Content-Type", "application/json")
	}

	var resBody []byte
	{
		var res *http.Response
		res, err = http.DefaultClient.Do(req)
		if err != nil {
			return "", fmt.Errorf("http.Get error: %v", err)
		}
		defer res.Body.Close()

		resBody, err = io.ReadAll(res.Body)
		if err != nil {
			return "", fmt.Errorf("error while reading res.Body from GET: %s: %v", reqURL, err)
		}

		if res.StatusCode != http.StatusOK {
			return "", fmt.Errorf("GET to %s: %d [%s]: %s", reqURL, res.StatusCode, res.Status, resBody)
		}
	}

	return string(resBody), nil
}
