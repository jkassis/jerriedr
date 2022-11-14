package main

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"

	"net/http"

	"github.com/google/uuid"
	"github.com/jkassis/jerrie/core"
	"github.com/jkassis/jerriedr/cmd/kube"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const requestFormat = ` { "UUID": "%s", "Fn": "/%s/Backup", "Body": {} }`

const FLAG_SERVICE = "service"

type HostService struct {
	Host string
	Port int
}

type PodService struct {
	PodName      string
	PodNamespace string
	LocalPort    int
}

func init() {
	// A general configuration object (feed with flags, conf files, etc.)
	v := viper.New()

	// CLI Command with flag parsing
	c := &cobra.Command{
		Use:   "servicebackup",
		Short: "Backup services using http backup reqeusts",
		Long: `This command is equivalent to the following curl call against the requested services

curl -d '{ "UUID": "<UUID>", "Fn": "/v1/Backup", "Body": {} }' -H 'Content-Type: application/json' http://<hostport>/raft/leader/read

eg..
curl -d '{ "UUID": "9db4caec-a449-4082-a1c3-ac82b4d25444", "Fn": "/v1/Backup", "Body": {} }' -H 'Content-Type: application/json' http://dockie-0.dockie-int.fg.svc.cluster.local:10000/raft/leader/read
`,
		Run: func(cmd *cobra.Command, args []string) {
			CMDBackupRemote(v)
		},
	}

	// kube
	CMDKubeConfig(c, v)
	CMDProtocolConfig(c, v)
	CMDVersionConfig(c, v)

	// services
	c.PersistentFlags().StringP(FLAG_SERVICE, "s", "", "a <host>:<port> that responds to requests at '<host>:<port>/<version>/backup' by placing backup files in /var/data/single/<host>-<port>-server-0/backup/<timestamp>.bak")
	v.BindPFlag(FLAG_SERVICE, c.PersistentFlags().Lookup(FLAG_SERVICE))

	MAIN.AddCommand(c)
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

func CMDBackupRemote(v *viper.Viper) {
	start := time.Now()
	core.Log.Warnf("BackupRemote: starting")

	// for each service
	serviceSpecs := v.GetStringSlice(FLAG_SERVICE)
	BackupFromServiceSpecs(v, serviceSpecs)

	duration := time.Since(start)
	core.Log.Warnf("BackupRemote: took %s", duration.String())
}

func BackupFromServiceSpecs(v *viper.Viper, serviceSpecs []string) {
	if len(serviceSpecs) == 0 {
		core.Log.Fatalf("No services specified.")
	}

	// get kube client
	kubeClient, kubeErr := KubeClientGet(v)

	// get services
	podServices := make([]*PodService, 0)
	hostServices := make([]*HostService, 0)
	for _, serviceSpecString := range serviceSpecs {
		// cast and validate
		serviceSpec := ServiceSpec(serviceSpecString)
		if !serviceSpec.IsValid() {
			core.Log.Fatalf("invalid service spec: %s", serviceSpec)
		}

		// is the host address in a kube cluster?
		if serviceSpec.IsKube() {
			// yes. make sure we have a kube client
			if kubeErr != nil {
				core.Log.Fatalf("kube client initialization failed: %v", kubeErr)
			}

			// forward a local port
			port, err := serviceSpec.PortGet()
			if err != nil {
				core.Log.Fatalf(err.Error())
			}
			forwardedPort, err := kubeClient.PortForward(&kube.PortForwardRequest{
				LocalPort:    0,
				PodName:      serviceSpec.PodGet(),
				PodNamespace: serviceSpec.NamespaceGet(),
				PodPort:      port,
			})
			if err != nil {
				core.Log.Fatalf("could not port forward to kube service %s: %v", serviceSpecString, err)
			}

			podServices = append(podServices, &PodService{PodName: serviceSpec.PodGet(), PodNamespace: serviceSpec.NamespaceGet(), LocalPort: int(forwardedPort.Local)})
		} else {
			// no.
			host := serviceSpec.HostGet()
			port, err := serviceSpec.PortGet()
			if err != nil {
				core.Log.Errorf("could not parse service hostport: %v", serviceSpec)
			}
			hostServices = append(hostServices, &HostService{Host: host, Port: port})
		}
	}

	// get http protocol and backup protocol version
	protocol := v.GetString(FLAG_PROTOCOL)
	core.Log.Infof("got protocol: %s", protocol)
	version := v.GetString(FLAG_VERSION)
	core.Log.Infof("got version: %s", version)

	// establish a WaitGroup
	wg := sync.WaitGroup{}

	request := func(req *http.Request, serviceName string) {
		// make the request
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			core.Log.Error(err)
			return
		}

		// get response and report
		defer resp.Body.Close()
		resBody, err := io.ReadAll(resp.Body)
		if err != nil {
			core.Log.Error(err)
		}
		core.Log.Warnf("BackupRemote: %s: %d %s", serviceName, resp.StatusCode, resBody)
		if err != nil {
			core.Log.Error(err)
		}
	}

	// start backups on hostServices
	for _, service := range hostServices {
		wg.Add(1)
		go func(service *HostService) {
			defer wg.Done()
			serviceName := service.Host + ":" + strconv.Itoa(service.Port)
			core.Log.Warnf("Running remote backup for %s", serviceName)
			reqBody := strings.NewReader(fmt.Sprintf(requestFormat, uuid.NewString(), version))
			req, err := http.NewRequest("POST", fmt.Sprintf("%s://%s:%d/raft/leader/read", protocol, service.Host, service.Port), reqBody)
			if err != nil {
				core.Log.Fatalln(err)
			}
			request(req, serviceName)
		}(service)
	}

	// start backups on podServices
	for _, service := range podServices {
		wg.Add(1)
		go func(service *PodService) {
			defer wg.Done()
			serviceName := service.PodNamespace + "/" + service.PodName + ":" + strconv.Itoa(service.LocalPort)
			core.Log.Warnf("Running remote backup for %s", serviceName)
			reqBody := strings.NewReader(fmt.Sprintf(requestFormat, uuid.NewString(), version))
			req, err := http.NewRequest("POST", fmt.Sprintf("%s://%s:%d/raft/leader/read", protocol, "localhost", service.LocalPort), reqBody)
			if err != nil {
				core.Log.Fatalln(err)
			}
			request(req, serviceName)
		}(service)
	}

	wg.Wait()
}
