package schema

import "fmt"

func ServiceSetNew() *ServiceSet {
	sss := &ServiceSet{}
	sss.Services = make([]*Service, 0)
	return sss
}

type ServiceSet struct {
	Services []*Service
}

func (as *ServiceSet) ServiceAdd(serviceSpec string) error {
	service := ServiceNew()
	err := service.Parse(serviceSpec)
	if err != nil {
		return err
	}
	as.Services = append(as.Services, service)
	return nil
}

func (as *ServiceSet) ServiceAddAll(serviceSpecs []string) error {
	for _, serviceSpec := range serviceSpecs {
		err := as.ServiceAdd(serviceSpec)
		if err != nil {
			return err
		}
	}
	return nil
}

func (as *ServiceSet) ServiceGetByServiceName(serviceName string) (a *Service, err error) {
	for _, service := range as.Services {
		if service.KubeName == serviceName {
			return service, nil
		}
	}

	return nil, fmt.Errorf("could not find service for serviceName '%s' have only these... %v", serviceName, as.Services)
}
