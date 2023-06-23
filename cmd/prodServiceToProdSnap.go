package main

import (
	"time"

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
		Use:   "prodServiceToProdSnap",
		Short: ``,
		Long:  "",
		Run: func(cmd *cobra.Command, args []string) {

			start := time.Now()
			core.Log.Warnf("prodServiceToProdSnap: starting")

			kubeClient, err := KubeClientGet(v)
			if err != nil {
				core.Log.Warnf("could not init kubeClient: %v", err)
			}

			// for each service
			services := make([]*schema.Service, 0)
			for _, serviceSpec := range prodServiceSpecs {
				service := &schema.Service{}
				if err := service.Parse(serviceSpec); err != nil {
					core.Log.Fatalf("could not parse serviceSpec %s", serviceSpec)
				}

				services = append(services, service)
			}

			err = schema.EnvSnap(kubeClient, services)
			if err != nil {
				core.Log.Fatalf("could not complete production snapshot: %v", err)
			}

			duration := time.Since(start)
			core.Log.Warnf("prodSnapshotTake: took %s", duration.String())
		},
	}

	// kube
	FlagsAddKubeFlags(c, v)
	FlagsAddProtocolFlag(c, v)
	FlagsAddAPIVersionFlag(c, v)

	MAIN.AddCommand(c)
}
