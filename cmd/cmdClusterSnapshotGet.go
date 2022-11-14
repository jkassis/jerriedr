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
		Use:   "clustersnapshotget",
		Short: "Retrieve a snapshot of cluster services and save to a local file.",
		Long:  `This command is a shortcut for backupremote.`,
		Run: func(cmd *cobra.Command, args []string) {
			CMDClusterSnapshotGet(v)
		},
	}

	// kube
	CMDKubeConfig(c, v)

	// localDir
	MAIN.AddCommand(c)
}

func CMDClusterSnapshotGet(v *viper.Viper) {
	start := time.Now()
	core.Log.Warnf("ClusterSnapshotGet: starting")

	serviceSpecs := []string{"pod|fg/dockie-0:10000|/var/data/single/dockie-0-server-0"}
	ServiceSnapshotGet(v, serviceSpecs)
	duration := time.Since(start)
	core.Log.Warnf("ClusterSnapshotGet: took %s", duration.String())
}
