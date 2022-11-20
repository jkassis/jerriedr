package main

import (
	"time"

	"github.com/jkassis/jerrie/core"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	// A general configuration object (feed with flags, conf files, etc.)
	v := viper.New()

	// CLI Command with flag parsing
	c := &cobra.Command{
		Use:   "clustersnapshottake",
		Short: "Backup cluster services using http backup reqeusts",
		Long:  `This command is a shortcut for servicesnapshottake with preconfigured services.`,
		Run: func(cmd *cobra.Command, args []string) {
			CMDClusterSnapshotTake(v)
		},
	}

	// kube
	FlagsAddKubeFlags(c, v)
	FlagsAddProtocolFlag(c, v)
	FlagsAddAPIVersionFlag(c, v)

	MAIN.AddCommand(c)
}

func CMDClusterSnapshotTake(v *viper.Viper) {
	start := time.Now()
	core.Log.Warnf("CMDClusterSnapshotTake: starting")

	// for each service
	serviceSpecs := []string{
		"kube|fg/dockie-0:10000",
		"kube|fg/tickie-0:10000",
	}
	SnapshotTake(v, serviceSpecs)

	duration := time.Since(start)
	core.Log.Warnf("CMDClusterSnapshotTake: took %s", duration.String())
}
