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
	"github.com/jkassis/jerriedr/cmd/schema"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	v := viper.New()

	c := &cobra.Command{
		Use:   "servicesnapshottake",
		Short: "Trigger a remote service to take a snapshot.",
		Long:  ``,
		Run: func(cmd *cobra.Command, args []string) {
			CMDSnapshotTake(v)
		},
	}

	// flag configuration
	FlagsAddKubeFlags(c, v)
	schema.FlagsAddServiceFlag(c, v)
	FlagsAddProtocolFlag(c, v)
	FlagsAddAPIVersionFlag(c, v)

	MAIN.AddCommand(c)
}

const requestFormat = ` { "UUID": "%s", "Fn": "/%s/Backup", "Body": {} }`

func CMDSnapshotTake(v *viper.Viper) {
	start := time.Now()
	core.Log.Warnf("CMDSnapshotTake: starting")

	// for each service
	serviceSpecs := v.GetStringSlice(schema.FLAG_SERVICE)
	SnapshotTake(v, serviceSpecs)

	duration := time.Since(start)
	core.Log.Warnf("CMDSnapshotTake: took %s", duration.String())
}

func SnapshotTake(v *viper.Viper, serviceSpecs []string) {
	if len(serviceSpecs) == 0 {
		core.Log.Fatalf("No services specified.")
	}

	// get http protocol and backup protocol version
	protocol := v.GetString(FLAG_PROTOCOL)
	core.Log.Infof("got protocol: %s", protocol)

	version := v.GetString(FLAG_VERSION)
	core.Log.Infof("got version: %s", version)

	// parse service specs
	hostServices, podServices, err := schema.ParseServiceSpecs(serviceSpecs)
	if err != nil {
		core.Log.Fatalf("CMDSnapshotTake: %v", err)
	}

	// subroutine to do an http request
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
		core.Log.Warnf("CMDSnapshotTake: %s: %d %s", serviceName, resp.StatusCode, resBody)
		if err != nil {
			core.Log.Error(err)
		}
	}

	// establish a WaitGroup
	wg := sync.WaitGroup{}

	// get kube client
	kubeClient, kubeErr := KubeClientGet(v)

	// start backups on podServices
	for _, service := range podServices {
		wg.Add(1)
		go func(service *schema.PodService) {
			defer wg.Done()

			// yes. make sure we have a kube client
			if kubeErr != nil {
				core.Log.Fatalf("kube client initialization failed: %v", kubeErr)
			}

			// forward a local port
			forwardedPort, err := kubeClient.PortForward(&kube.PortForwardRequest{
				LocalPort:    0,
				PodName:      service.PodName,
				PodNamespace: service.PodNamespace,
				PodPort:      service.PodPort,
			})
			if err != nil {
				core.Log.Fatalf("could not port forward to kube service %s: %v", service.Spec, err)
			}

			localPort := forwardedPort.Local

			serviceName := service.PodNamespace + "/" + service.PodName + ":" + strconv.Itoa(int(localPort))
			core.Log.Warnf("Running remote backup for %s", serviceName)
			reqBody := strings.NewReader(fmt.Sprintf(requestFormat, uuid.NewString(), version))
			req, err := http.NewRequest("POST", fmt.Sprintf("%s://%s:%d/raft/leader/read", protocol, "localhost", localPort), reqBody)
			if err != nil {
				core.Log.Fatalln(err)
			}
			request(req, serviceName)
		}(service)
	}

	// start backups on hostServices
	for _, service := range hostServices {
		wg.Add(1)
		go func(service *schema.HostService) {
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

	wg.Wait()
}
