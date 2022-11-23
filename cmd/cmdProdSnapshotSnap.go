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
		Use:   "prodsnapshotsnap",
		Short: `Snaps a snapshot of all services in the prod cluster.`,
		Long:  "Ask all prod services to snap snapshots of their data and save in their local archives. No data is transferred. Can be done without downtime.",
		Run: func(cmd *cobra.Command, args []string) {
			CMDProdSnapshotSnap(v)
		},
	}

	// kube
	FlagsAddKubeFlags(c, v)
	FlagsAddProtocolFlag(c, v)
	FlagsAddAPIVersionFlag(c, v)

	MAIN.AddCommand(c)
}

func CMDProdSnapshotSnap(v *viper.Viper) {
	start := time.Now()
	core.Log.Warnf("CMDProdSnapshotSnap: starting")

	// for each service
	services := make([]*schema.Service, 0)
	for _, serviceSpec := range prodServiceSpecs {
		service := &schema.Service{}
		if err := service.Parse(serviceSpec); err != nil {
			core.Log.Fatalf("could not parse serviceSpec %s", serviceSpec)
		}

		services = append(services, service)
	}

	err := ServiceSnapshotSnap(v, services)
	if err != nil {
		core.Log.Fatal("could not complete production snapshot: %v", err)
	}

	duration := time.Since(start)
	core.Log.Warnf("CMDProdSnapshotSnap: took %s", duration.String())
}
