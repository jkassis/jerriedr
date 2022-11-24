package schema

import (
	"fmt"
	"strconv"
	"strings"
)

func ServiceNew() *Service {
	service := &Service{}
	return service
}

type Service struct {
	Scheme        string
	Host          string
	KubeContainer string
	KubeName      string
	KubeNamespace string
	Port          int
	Parent        *Service
	Spec          string
}

func (a *Service) Parse(spec string) error {
	parts := strings.Split(spec, "|")
	a.Scheme = parts[0]

	if a.Scheme == "statefulset" {
		err := fmt.Errorf("%s must be statefulset|<kubeNamespace>/<kubeName>(/<container>)?|port", spec)

		if len(parts) != 3 {
			return err
		}

		{
			statefulSet := parts[1]
			if !strings.Contains(statefulSet, "/") {
				return err
			}

			{
				statefulSetParts := strings.Split(statefulSet, "/")
				a.KubeNamespace = statefulSetParts[0]
				if a.KubeNamespace == "" {
					return err
				}

				a.KubeName = statefulSetParts[1]
				if a.KubeName == "" {
					return err
				}

				if len(statefulSetParts) == 3 {
					a.KubeContainer = statefulSetParts[2]
				}
			}
		}

		{
			port, converr := strconv.Atoi(parts[2])
			if converr != nil {
				return err
			}
			a.Port = port
		}
	} else if a.Scheme == "pod" {
		err := fmt.Errorf("%s must be pod|<kubeNamespace>/<kubeName>(/<container>)?|<path>", spec)

		if len(parts) != 3 {
			return err
		}

		{
			pod := parts[1]
			if !strings.Contains(pod, "/") {
				return err
			}

			{
				podParts := strings.Split(pod, "/")
				a.KubeNamespace = podParts[0]
				if a.KubeNamespace == "" {
					return err
				}

				a.KubeName = podParts[1]
				if a.KubeName == "" {
					return err
				}

				if len(podParts) == 3 {
					a.KubeContainer = podParts[2]
				}
			}
		}

		{
			port, converr := strconv.Atoi(parts[2])
			if converr != nil {
				return err
			}
			a.Port = port
		}
	} else if a.Scheme == "host" {
		err := fmt.Errorf("%s must be host|<hostName>|<port>", spec)

		if len(parts) != 3 {
			return err
		}

		{
			a.Host = parts[1]
			if a.Host == "" {
				return err
			}
		}

		{
			port, converr := strconv.Atoi(parts[2])
			if converr != nil {
				return err
			}
			a.Port = port
		}
	} else if a.Scheme == "local" {
		err := fmt.Errorf("%s must be local|<port>", spec)

		a.Host = "localhost"

		if len(parts) != 2 {
			return err
		}

		{
			port, convErr := strconv.Atoi(parts[1])
			if convErr != nil {
				return err
			}
			a.Port = port
		}
	} else {
		return fmt.Errorf("%s must be <scheme>|<schemeSpec> where <scheme> => statefulset | pod | host | local: %s", spec, a.Scheme)
	}

	a.Spec = spec
	return nil
}

func (a *Service) IsPod() bool {
	return a.Scheme == "pod"
}

func (a *Service) IsHost() bool {
	return a.Scheme == "host"
}

func (a *Service) IsLocal() bool {
	return a.Scheme == "local"
}

func (a *Service) IsStatefulSet() bool {
	return a.Scheme == "statefulset"
}

func (a *Service) PodServiceGet(replica int) (*Service, error) {
	if !a.IsStatefulSet() {
		return nil, fmt.Errorf("%s is not a statefulset spec", a.Spec)
	}
	podName := a.KubeName + "-" + strconv.Itoa(replica)
	return &Service{
		Host:          "",
		KubeContainer: a.KubeContainer,
		KubeName:      podName,
		KubeNamespace: a.KubeNamespace,
		Parent:        a,
		Port:          a.Port,
		Scheme:        "pod",
		Spec:          fmt.Sprintf("pod|%s/%s/%s|%d", a.KubeNamespace, podName, a.KubeContainer, a.Port),
	}, nil
}
