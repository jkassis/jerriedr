package main

import (
	"time"

	"github.com/jkassis/jerrie/core"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"
)

func init() {
	// A general configuration object (feed with flags, conf files, etc.)
	v := viper.New()

	// CLI Command with flag parsing
	c := &cobra.Command{
		Use:   "clustersnapshotget",
		Short: "Retrieve a snapshot of cluster services and save to a local archive.",
		Long:  `This command is a shortcut for servicesnapshotcopy with several presets.`,
		Run: func(cmd *cobra.Command, args []string) {
			CMDClusterSnapshotGet(v)
		},
	}

	// kube
	FlagsAddKubeFlags(c, v)

	// localDir
	MAIN.AddCommand(c)
}

func CMDClusterSnapshotGet(v *viper.Viper) {
	start := time.Now()
	core.Log.Warnf("ClusterSnapshotGet: starting")

	srcArchiveSpecs := []string{
		"pod|fg/dockie-0|/var/data/single/dockie-0-server-0",
		"pod|fg/ledgie-0|/var/data/single/ledgie-0-server-0",
	}

	dstArchiveSpec := "host|localhost|/var/cluster"

	errGroup := errgroup.Group{}
	for _, srcArchiveSpec := range srcArchiveSpecs {
		srcArchiveSpec := srcArchiveSpec
		errGroup.Go(func() error {
			return ServiceSnapshotCopy(v, srcArchiveSpec, dstArchiveSpec)
		})
	}

	err := errGroup.Wait()
	if err != nil {
		core.Log.Error(err)
	}

	duration := time.Since(start)
	core.Log.Warnf("ClusterSnapshotGet: took %s", duration.String())
}
