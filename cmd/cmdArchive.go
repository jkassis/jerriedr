package main

import (
	"fmt"
	"strings"
)

const FLAG_ARCHIVE = "archive"

type Archive struct {
	Scheme       string
	Path         string
	Host         string
	Spec         string
	PodName      string
	PodNamespace string
	PodContainer string
}

func (s *Archive) Parse(in string) error {
	parts := strings.Split(in, "|")
	s.Scheme = parts[0]
	if s.Scheme == "pod" {
		if len(parts) != 3 {
			return fmt.Errorf("%s must have 3 parts separated by |", in)
		}

		host := parts[1]
		if !strings.Contains(host, "/") {
			return fmt.Errorf("must have <namespace>/<pod>")
		}

		hostParts := strings.Split(host, "/")
		s.PodNamespace = hostParts[0]
		if s.PodNamespace == "" {
			return fmt.Errorf("no namespace")
		}

		s.PodName = hostParts[1]
		if s.PodName == "" {
			return fmt.Errorf("no pod")
		}

		s.Path = parts[2]
		if s.Path == "" {
			return fmt.Errorf("must have <path>")
		}
	} else if s.Scheme == "host" {
		if len(parts) != 3 {
			return fmt.Errorf("%s must have 3 parts separated by |", in)
		}
		s.Host = parts[1]
		if s.Host == "" {
			return fmt.Errorf("must have <host>")
		}

		s.Path = parts[2]
		if s.Path == "" {
			return fmt.Errorf("must have <path>")
		}
	} else if s.Scheme == "local" {
		if len(parts) != 2 {
			return fmt.Errorf("%s must have 2 parts separated by |", in)
		}
		s.Path = parts[1]
		if s.Path == "" {
			return fmt.Errorf("must have <path>")
		}
	} else {
		return fmt.Errorf("<archiveSpec> had invalid scheme: %s", s.Scheme)
	}
	return nil
}

func (s *Archive) IsPod() bool {
	return s.Scheme == "pod"
}

func (s *Archive) IsHost() bool {
	return s.Scheme == "host"
}

func (s *Archive) IsLocal() bool {
	return s.Scheme == "local"
}
