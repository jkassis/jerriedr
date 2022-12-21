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
			CMDProdServiceToProdSnap(v)
		},
	}

	// kube
	FlagsAddKubeFlags(c, v)
	FlagsAddProtocolFlag(c, v)
	FlagsAddAPIVersionFlag(c, v)

	MAIN.AddCommand(c)
}

func CMDProdServiceToProdSnap(v *viper.Viper) {
	start := time.Now()
	core.Log.Warnf("prodServiceToProdSnap: starting")

	// for each service
	services := make([]*schema.Service, 0)
	for _, serviceSpec := range prodServiceSpecs {
		service := &schema.Service{}
		if err := service.Parse(serviceSpec); err != nil {
			core.Log.Fatalf("could not parse serviceSpec %s", serviceSpec)
		}

		services = append(services, service)
	}

	err := EnvSnap(v, services)
	if err != nil {
		core.Log.Fatal("could not complete production snapshot: %v", err)
	}

	duration := time.Since(start)
	core.Log.Warnf("prodSnapshotTake: took %s", duration.String())
}
