package kube

import (
	"bufio"
	"io"
	"io/ioutil"
	"sync"

	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

// StreamAllToLog streams stdout, stderr, and an errChan to logs
func StreamAllToLog(prefix string, readers ...io.Reader) {
	wg := sync.WaitGroup{}
	doOne := func(r io.Reader) {
		defer wg.Done()
		err := StreamOneToLog(prefix, r)
		if err != nil {
			if err != io.EOF {
				logrus.Error(err)
				return
			}
		}
	}

	wg.Add(len(readers))
	for _, r := range readers {
		go doOne(r)
	}
	wg.Wait()
}

// ReadAll reads all data from multiple readers
func ReadAll(readers ...io.Reader) (results [][]byte, err error) {
	results = make([][]byte, len(readers))
	eg := errgroup.Group{}
	for i, r := range readers {
		var i = i
		var r = r
		eg.Go(func() error {
			var err error
			var result []byte
			result, err = ioutil.ReadAll(r)
			if err != nil {
				return err
			}
			results[i] = result
			return nil
		})
	}
	err = eg.Wait()
	if err != nil {
		return nil, err
	}

	return results, nil
}

// StreamOneToLog streams the input to logs with a given prefix
func StreamOneToLog(prefix string, input io.Reader) error {
	inputBuf := bufio.NewReader(input)
	line := make([]byte, 0)
	for {
		partial, isPrefix, err := inputBuf.ReadLine()
		if err != nil {
			if err != io.EOF {
				return err
			}
			return nil
		}
		line = append(line, partial...)
		if isPrefix {
			continue
		}
		logrus.Info(prefix + string(line))
		line = make([]byte, 0)
	}
}
