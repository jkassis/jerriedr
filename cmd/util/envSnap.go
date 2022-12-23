package util

import (
	"github.com/jkassis/jerriedr/cmd/kube"
	"github.com/jkassis/jerriedr/cmd/schema"
	"golang.org/x/sync/errgroup"
)

func EnvSnap(kubeClient *kube.Client, services []*schema.Service) (err error) {
	// establish an errgroup
	eg := errgroup.Group{}

	// start backups on podServices
	for _, service := range services {
		service := service
		eg.Go(func() error {
			return service.Snap(kubeClient)
		})
	}

	return eg.Wait()
}
