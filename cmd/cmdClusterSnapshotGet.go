package main

import (
	"fmt"
	"time"

	"github.com/jkassis/jerrie/core"
	"github.com/jkassis/jerriedr/cmd/kube"
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

	serviceSpecs := []string{
		"kube|fg/dockie-0:10000",
	}

	// get the list of all files
	func() error {
		if len(serviceSpecs) == 0 {
			core.Log.Fatalf("No services specified.")
		}

		// get kube client
		kubeClient, kubeErr := KubeClientGet(v)
		if err != nil {
			return fmt.Errorf("pod %s/%s : not found", "fg", "dockie-0")
		}

		pod, err := kubeClient.GetPod("dockie-0", "fg")
		if err != nil {
			return fmt.Errorf("pod %s/%s : not found", "fg", "dockie-0")
		}

		stdout, stderr, err := kubeClient.LsOnPod(&kube.FileSpec{
			PodNamespace: "fg",
			PodName:      "dockie-0",
			File:         "/var",
		}, pod, "")
		if err != nil {
			return fmt.Errorf("pod %s/%s : not found", "fg", "dockie-0")
		}

		go kube.StreamAllToLog("ClusterSnapshotGet: ", stdout, stderr)

		// pick the most recent

		// rsync to the target dir

		return nil
	}()

	duration := time.Since(start)
	core.Log.Warnf("ClusterSnapshotGet: took %s", duration.String())
}
