package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const FLAG_SERVICE = "service"

func CMDServiceConfig(c *cobra.Command, v *viper.Viper) {
	c.PersistentFlags().StringP(FLAG_SERVICE, "s", "", "a <host>:<port> that responds to requests at '<host>:<port>/<version>/backup' by placing backup files in /var/data/single/<host>-<port>-server-0/backup/<timestamp>.bak")
	v.BindPFlag(FLAG_SERVICE, c.PersistentFlags().Lookup(FLAG_SERVICE))
}

type HostService struct {
	Dir  string
	Host string
	Port int
	Spec string
}

type PodService struct {
	Dir          string
	PodName      string
	PodNamespace string
	PodPort      int
	Spec         string
}

// ServiceSpec is a string that represents a service... local or in kube
// servicespec => <podSpec> | <hostSpec>
// podSpec => pod|<ns>/<pod>:<port>|<path>
// hostSpec => host|<hostname>:<port>|<path>
//
// Better...
// podSpec => pod://<ns>/<pod>:<port>/<path>
// hostSpec => host://<hostname>:<port>/<path>
type ServiceSpec string

func (s ServiceSpec) IsKube() bool {
	return strings.HasPrefix(string(s), "pod|")
}

func (s ServiceSpec) IsHost() bool {
	return strings.HasPrefix(string(s), "host|")
}

func (s ServiceSpec) IsValid() bool {
	// TODO could get more complex with the validation
	parts := strings.Split(string(s), "|")
	if len(parts) < 3 {
		return false
	}

	host := parts[1]

	if s.IsKube() {
		return strings.Contains(host, "/") && strings.Contains(host, ":")
	} else {
		return strings.Contains(host, ":")
	}
}

func (s ServiceSpec) HostGet() string {
	parts := strings.Split(string(s), "|")
	parts = strings.Split(parts[1], ":")
	return parts[0]
}

func (s ServiceSpec) PodGet() string {
	parts := strings.Split(string(s), "|")
	parts = strings.Split(parts[1], "/")
	parts = strings.Split(parts[1], ":")
	return parts[0]
}

func (s ServiceSpec) NamespaceGet() string {
	parts := strings.Split(string(s), "|")
	parts = strings.Split(parts[1], "/")
	return parts[0]
}

func (s ServiceSpec) DirGet() string {
	parts := strings.Split(string(s), "|")
	return parts[2]
}

func (s ServiceSpec) PortGet() (int, error) {
	parts := strings.Split(string(s), "|")
	parts = strings.Split(parts[1], "/")
	parts = strings.Split(parts[1], ":")
	port, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, fmt.Errorf("bad port in %s: %v", string(s), err)
	}
	return port, nil
}

func parseServiceSpecs(serviceSpecs []string) ([]*HostService, []*PodService, error) {
	// get services
	podServices := make([]*PodService, 0)
	hostServices := make([]*HostService, 0)
	for _, serviceSpecString := range serviceSpecs {
		// cast and validate
		serviceSpec := ServiceSpec(serviceSpecString)
		if !serviceSpec.IsValid() {
			return nil, nil, fmt.Errorf("invalid service spec: %s", serviceSpec)
		}

		// is the host address in a kube cluster?
		if serviceSpec.IsKube() {
			podPort, err := serviceSpec.PortGet()
			if err != nil {
				return nil, nil, fmt.Errorf("could not parse service port: %v", serviceSpec)
			}
			podServices = append(podServices,
				&PodService{
					PodName:      serviceSpec.PodGet(),
					PodNamespace: serviceSpec.NamespaceGet(),
					PodPort:      podPort,
					Spec:         serviceSpecString,
					Dir:          serviceSpec.DirGet(),
				})
		} else if serviceSpec.IsHost() {
			// no.
			port, err := serviceSpec.PortGet()
			if err != nil {
				return nil, nil, fmt.Errorf("could not parse service hostport: %v", serviceSpec)
			}
			hostServices = append(hostServices,
				&HostService{
					Host: serviceSpec.HostGet(),
					Port: port,
					Spec: serviceSpecString,
					Dir:  serviceSpec.DirGet(),
				})
		} else {
			return nil, nil, fmt.Errorf("serviceSpec type must be pod|host: %v", serviceSpec)
		}
	}

	return hostServices, podServices, nil
}

// ArchiveSpec is a string that represents an archive... local or in kube
// archiveSpec => <podSpec> | <hostSpec>
// podSpec => pod|<ns>/<pod>|<path>
// hostSpec => host|<hostname>|<path>
//
// Better...
// podSpec => pod://<ns>/<pod>/<path>
// hostSpec => host://<hostname>/<path>
type ArchiveSpec string

func (s ArchiveSpec) IsKube() bool {
	return strings.HasPrefix(string(s), "kube|")
}

func (s ArchiveSpec) IsValid() bool {
	// TODO could get more complex with the validation
	if s.IsKube() {
		return strings.Contains(string(s), "/") && strings.Contains(string(s), ":")
	} else {
		return strings.Contains(string(s), ":")
	}
}

func (s ArchiveSpec) HostGet() string {
	parts := strings.Split(string(s), ":")
	return parts[0]
}

func (s ArchiveSpec) PodGet() string {
	tail := string(s)[5:]
	parts := strings.Split(tail, "/")
	parts = strings.Split(parts[1], ":")
	return parts[0]
}

func (s ArchiveSpec) NamespaceGet() string {
	tail := string(s)[5:]
	parts := strings.Split(tail, "/")
	return parts[0]
}

func (s ArchiveSpec) DirGet() (int, error) {
	parts := strings.Split(string(s), ":")
	port, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, fmt.Errorf("bad port in %s: %v", string(s), err)
	}
	return port, nil
}
