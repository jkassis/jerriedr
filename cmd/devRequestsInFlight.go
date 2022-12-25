package main

import (
	"github.com/jkassis/jerrie/core"
	"github.com/jkassis/jerriedr/cmd/schema"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	// A general configuration object (feed with flags, conf files, etc.)
	v := viper.New()

	// CLI Command with flag parsing
	c := &cobra.Command{
		Use:   "devRequestsInFlight",
		Short: "",
		Long:  ``,
		Run: func(cmd *cobra.Command, args []string) {

			kubeClient, kubeErr := KubeClientGet(v)
			if kubeErr != nil {
				core.Log.Errorf("could not get KubeClient: %v", kubeErr)
			}

			// for each service
			for _, serviceSpec := range devServiceSpecs {
				service := &schema.Service{}
				if err := service.Parse(serviceSpec); err != nil {
					core.Log.Fatalf("could not parse serviceSpec %s", serviceSpec)
				}

				n, err := service.RequestsInFlight(kubeClient)
				if err != nil {
					core.Log.Fatalf("could not get requests in flight", err)
				}
				core.Log.Warnf("dev has %d requests in flight", n)
			}
		},
	}

	FlagsAddKubeFlags(c, v)
	MAIN.AddCommand(c)
}
