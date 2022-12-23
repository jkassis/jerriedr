package main

import (
	"github.com/jkassis/jerrie/core"
	"github.com/jkassis/jerriedr/cmd/schema"
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"
)

func EnvSnap(v *viper.Viper, services []*schema.Service) (err error) {
	// establish an errgroup
	eg := errgroup.Group{}

	// get kube client
	kubeClient, kubeErr := KubeClientGet(v)
	if kubeErr != nil {
		core.Log.Errorf("kube client initialization failed: %v", kubeErr)
	}

	// start backups on podServices
	for _, service := range services {
		service := service
		eg.Go(func() error {
			return service.Snap(kubeClient)
		})
	}

	return eg.Wait()
}
