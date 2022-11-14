package main

import (
	"fmt"
	"strconv"
	"strings"
)

const FLAG_SERVICE = "service"

type HostService struct {
	Host string
	Port int
	Spec string
}

type PodService struct {
	PodName      string
	PodNamespace string
	PodPort      int
	Spec         string
}

// ServiceSpec is a string that represents a service... local or in kube
type ServiceSpec string

func (s ServiceSpec) IsKube() bool {
	return strings.HasPrefix(string(s), "kube|")
}

func (s ServiceSpec) IsValid() bool {
	// TODO could get more complex with the validation
	if s.IsKube() {
		return strings.Contains(string(s), "/") && strings.Contains(string(s), ":")
	} else {
		return strings.Contains(string(s), ":")
	}
}

func (s ServiceSpec) HostGet() string {
	parts := strings.Split(string(s), ":")
	return parts[0]
}

func (s ServiceSpec) PodGet() string {
	tail := string(s)[5:]
	parts := strings.Split(tail, "/")
	parts = strings.Split(parts[1], ":")
	return parts[0]
}

func (s ServiceSpec) NamespaceGet() string {
	tail := string(s)[5:]
	parts := strings.Split(tail, "/")
	return parts[0]
}

func (s ServiceSpec) PortGet() (int, error) {
	parts := strings.Split(string(s), ":")
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
				})
		} else {
			// no.
			host := serviceSpec.HostGet()
			port, err := serviceSpec.PortGet()
			if err != nil {
				return nil, nil, fmt.Errorf("could not parse service hostport: %v", serviceSpec)
			}
			hostServices = append(hostServices, &HostService{Host: host, Port: port, Spec: serviceSpecString})
		}
	}

	return hostServices, podServices, nil
}
