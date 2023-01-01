package schema

import (
	"fmt"
	"strconv"

	"golang.org/x/sync/errgroup"
)

func ServiceSetNew() *ServiceSet {
	sss := &ServiceSet{}
	sss.Services = make([]*Service, 0)
	return sss
}

type ServiceSet struct {
	Services []*Service
}

func (as *ServiceSet) ServiceAddBySpec(serviceSpec string) error {
	service := ServiceNew()
	err := service.Parse(serviceSpec)
	if err != nil {
		return err
	}
	as.Services = append(as.Services, service)
	return nil
}

func (as *ServiceSet) ServiceAdd(service *Service) error {
	as.Services = append(as.Services, service)
	return nil
}

func (as *ServiceSet) ServiceAddAll(serviceSpecs []string) error {
	for _, serviceSpec := range serviceSpecs {
		err := as.ServiceAddBySpec(serviceSpec)
		if err != nil {
			return err
		}
	}
	return nil
}

func (as *ServiceSet) ServiceGetByName(serviceName string) (a *Service, err error) {
	for _, service := range as.Services {
		if service.Name == serviceName {
			return service, nil
		}
	}

	serviceNames := make([]string, 0)
	for _, service := range as.Services {
		serviceNames = append(serviceNames, service.Name)
	}
	return nil, fmt.Errorf("could not find service for serviceName '%s' have only these... %v", serviceName, serviceNames)
}

func (as *ServiceSet) DoOncePerEndpoint(fn func(*Service) error) (err error) {
	doneOnce := make(map[string]struct{})
	eg := errgroup.Group{}
	for _, service := range as.Services {
		key := service.Host + ":" + strconv.Itoa(service.Port)
		if _, ok := doneOnce[key]; ok {
			continue
		}
		doneOnce[key] = struct{}{}

		service := service
		eg.Go(func() (err error) {
			return fn(service)
		})
	}

	return eg.Wait()
}
