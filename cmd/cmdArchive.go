package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const FLAG_ARCHIVE = "archive"

func CMDArchiveConfig(c *cobra.Command, v *viper.Viper) {
	c.PersistentFlags().StringP(FLAG_ARCHIVE, "s", "", "a <host> that responds to requests at '<host>/<version>/backup' by placing backup files in /var/data/single/<pod>-<port>-server-0/backup/<timestamp>.bak")
	v.BindPFlag(FLAG_ARCHIVE, c.PersistentFlags().Lookup(FLAG_ARCHIVE))
}

type Archive struct {
	Scheme       string
	Path         string
	Host         string
	Spec         string
	PodName      string
	PodNamespace string
	PodContainer string
}

func (s Archive) Parse(in string) error {
	parts := strings.Split(in, "|")
	if len(parts) < 3 {
		return fmt.Errorf("<archiveSpec> equires 3 parts separated by |")
	}

	s.Scheme = parts[0]
	if s.Scheme == "pod" {
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
		s.Host = parts[1]
		if s.Host == "" {
			return fmt.Errorf("must have <host>")
		}

		s.Path = parts[2]
		if s.Path == "" {
			return fmt.Errorf("must have <path>")
		}
	} else if s.Scheme == "local" {
		s.Path = parts[2]
		if s.Path == "" {
			return fmt.Errorf("must have <path>")
		}
	} else {
		return fmt.Errorf("<archiveSpec> had invalid scheme: %s", s.Scheme)
	}
	return nil
}

func (s Archive) IsKube() bool {
	return s.Host != ""
}

func (s Archive) IsHost() bool {
	return s.PodName != ""
}

func (s Archive) IsLocal() bool {
	return s.Host != ""
}
