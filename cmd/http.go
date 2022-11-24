package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

func HTTPPost(reqURL, contentType, body string) (resBodyString string, err error) {
	// make the request
	res, err := http.Post(reqURL, contentType, strings.NewReader(body))
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("error while reading res.Body from POST: %s: %v", reqURL, err)
	}

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("POST to %s: %d [%s]: %s", reqURL, res.StatusCode, res.Status, resBody)
	}

	return string(resBody), nil
}

func HTTPGet(reqURL string) (resBodyString string, err error) {
	// make the request
	res, err := http.Get(reqURL)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("error while reading res.Body from GET: %s: %v", reqURL, err)
	}

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GET  %s: %d [%s]: %s", reqURL, res.StatusCode, res.Status, resBody)
	}

	return string(resBody), nil
}
