package main

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/jkassis/jerrie/core"
	"github.com/jkassis/jerriedr/cmd/kube"
	"github.com/jkassis/jerriedr/cmd/schema"
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"
)

func EnvSnap(v *viper.Viper, services []*schema.Service) (err error) {
	// establish an errgroup
	eg := errgroup.Group{}

	// get kube client
	kubeClient, kubeErr := KubeClientGet(v)

	// start backups on podServices
	for _, service := range services {
		service := service
		eg.Go(func() error {
			if service.IsStatefulSet() {
				service, err = service.PodServiceGet(0)
			}

			core.Log.Warnf("running remote backup for %s", service.Spec)

			var reqBody, reqURL string
			{
				reqBody = fmt.Sprintf(
					` { "UUID": "%s", "Fn": "%s", "Body": {} }`,
					uuid.NewString(),
					service.BackupURL)

				if service.IsPod() {
					// yes. make sure we have a kube client
					if kubeErr != nil {
						return fmt.Errorf("kube client initialization failed: %v", kubeErr)
					}

					// forward a local port
					forwardedPort, err := kubeClient.PortForward(&kube.PortForwardRequest{
						LocalPort:    0,
						PodName:      service.KubeName,
						PodNamespace: service.KubeNamespace,
						PodPort:      service.Port,
					})
					if err != nil {
						return fmt.Errorf("could not port forward to kube service %s: %v", service.Spec, err)
					}
					localPort := forwardedPort.Local
					reqURL = fmt.Sprintf("%s://%s:%d/raft/leader/read", "http", "localhost", localPort)
				} else if service.IsHost() {
					reqURL = fmt.Sprintf("http://%s:%d/raft/leader/read", service.Host, service.Port)
				} else if service.IsLocal() {
					reqURL = fmt.Sprintf("http://localhost:%d/raft/leader/read", service.Port)
				}
			}

			// make the request
			if res, err := HTTPPost(reqURL, "application/json", reqBody); err != nil {
				return fmt.Errorf("could not request %s: %v", reqURL, err)
			} else {
				core.Log.Warnf("finished %s: %s", service.KubeName, res)
			}
			return nil
		})
	}

	return eg.Wait()
}
