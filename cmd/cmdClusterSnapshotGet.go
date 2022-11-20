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
	// dstArchiveSpec := "local|/var/cluster"

	// setup the srcArchiveSet
	srcArchiveSet := schema.ArchiveSetNew()
	for _, srcArchiveSpec := range []string{
		"statefulset|fg/dockie|/var/data/single/<pod>-server-0/backup",
		"statefulset|fg/ledgie|/var/data/single/<pod>-server-0/backup",
		"statefulset|fg/tickie|/var/data/single/<pod>-server-0/backup",
		"statefulset|fg/dubbie|/var/data/single/<pod>-server-0/backup",
		"statefulset|fg/keevie|/var/data/single/<pod>-server-0/backup",
		"statefulset|fg/permie|/var/data/single/<pod>-server-0/backup",
	} {
		srcArchiveSet.ArchiveAdd(srcArchiveSpec)
	}

	// get a kube client
	kubeClient, kubeErr := KubeClientGet(v)
	if kubeErr != nil {
		core.Log.Fatalf("kube client initialization failed: %v", kubeErr)
	}

	// fetch all the files
	err := srcArchiveSet.FilesFetch(kubeClient)
	if err != nil {
		core.Log.Fatalf("failed to get files for cluster archive set: %v", err)
	}

	// seek to now
	srcArchiveSet.SeekTo(time.Now())

	sss := srcArchiveSet.SnapshotSetGetNext()
	core.Log.Warn(sss.String())

	// {
	// 	core.Log.Warnf("CMDClusterSnapshotGet: starting")
	// 	start := time.Now()
	// 	errGroup := errgroup.Group{}
	// 	for _, srcArchiveSpec := range srcArchiveSpecs {
	// 		srcArchiveSpec := srcArchiveSpec
	// 		errGroup.Go(func() error {
	// 			return SnapshotCopy(v, srcArchiveSpec, dstArchiveSpec)
	// 		})
	// 	}

	// 	err := errGroup.Wait()
	// 	if err != nil {
	// 		core.Log.Error(err)
	// 	}

	// 	duration := time.Since(start)
	// 	core.Log.Warnf("CMDClusterSnapshotGet: took %s", duration.String())
	// }
}
