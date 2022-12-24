package schema

import (
	"github.com/jkassis/jerriedr/cmd/kube"
	"golang.org/x/sync/errgroup"
)

func EnvSnap(kubeClient *kube.Client, services []*Service) (err error) {
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
