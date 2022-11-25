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
		Use:   "devsnapshottake",
		Short: `Snaps a snapshot of all services in the dev cluster.`,
		Long:  "Ask all dev services to snap snapshots of their data and save in their local archives. No data is transferred. Can be done without downtime.",
		Run: func(cmd *cobra.Command, args []string) {
			CMDDevSnapshotTake(v)
		},
	}

	// kube
	FlagsAddProtocolFlag(c, v)
	FlagsAddAPIVersionFlag(c, v)

	MAIN.AddCommand(c)
}

func CMDDevSnapshotTake(v *viper.Viper) {
	start := time.Now()
	core.Log.Warnf("devSnapshotTake: starting")

	// for each service
	services := make([]*schema.Service, 0)
	for _, serviceSpec := range devServiceSpecs {
		service := &schema.Service{}
		if err := service.Parse(serviceSpec); err != nil {
			core.Log.Fatalf("could not parse serviceSpec %s", serviceSpec)
		}
		services = append(services, service)
	}

	err := EnvSnapshotTake(v, services)
	if err != nil {
		core.Log.Fatal("could not complete dev snapshot: %v", err)
	}

	duration := time.Since(start)
	core.Log.Warnf("devSnapshotTake: took %s", duration.String())
}
