package main

import (
	"sync"
	"time"

	"github.com/jkassis/jerrie/core"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const FLAG_TIME_TARGET = "time"

func init() {
	v := viper.New()

	c := &cobra.Command{
		Use:   "servicerestoreprepare",
		Short: "Copy one or more snapshots from a snapshot archive to a restore archive to prepare for restoration.",
		Long:  ``,
		Run: func(cmd *cobra.Command, args []string) {
			CMDserviceRestorePrepare(v)
		},
	}

	// flag configuration
	CMDKubeConfig(c, v)
	CMDSnapshotArchiveDir(c, v)
	CMDRestoreArchiveDir(c, v)

	// this flag represents the time target for selecting snapshots
	c.PersistentFlags().StringP(FLAG_TIME_TARGET, "t", "now", "a time target for selecting snapshots to restore")
	v.BindPFlag(FLAG_TIME_TARGET, c.PersistentFlags().Lookup(FLAG_TIME_TARGET))

	MAIN.AddCommand(c)
}

func CMDserviceRestorePrepare(v *viper.Viper) {
	start := time.Now()
	core.Log.Warnf("serviceRestorePrepare: starting")

	// for each archive dir
	snapshotArchiveSpecs := v.GetStringSlice(FLAG_SNAPSHOT_ARCHIVE)
	restoreArchiveSpecs := v.GetString(FLAG_RESTORE_ARCHIVE)
	serviceRestorePrepare(v, snapshotArchiveSpecs, restoreArchiveSpecs)

	duration := time.Since(start)
	core.Log.Warnf("serviceRestorePrepare: took %s", duration.String())
}

func serviceRestorePrepare(v *viper.Viper, snapshotArchivesSpecs []string, restoreArchiveSpecs string) {
	if len(snapshotArchivesSpecs) == 0 {
		core.Log.Fatalf("No services specified.")
	}

	// parse service specs
	hostServices, podServices, err := parseServiceSpecs(serviceSpecs)
	if err != nil {
		core.Log.Fatalf("serviceRestorePrepare: %v", err)
	}

	// establish a WaitGroup
	wg := sync.WaitGroup{}

	// get kube client
	kubeClient, kubeErr := KubeClientGet(v)

	// for podServices...
	for _, service := range podServices {
		wg.Add(1)
		go func(service *PodService) {
			defer wg.Done()
			core.Log.Warnf("Preparing restore for %s", service.Spec)

			// yes. make sure we have a kube client
			if kubeErr != nil {
				core.Log.Fatalf("kube client initialization failed: %v", kubeErr)
			}

			// get the list of snapshots on the target pod

			// find the snapshot taken most recently before the time target

			// move the snapshot to the restore directory

		}(service)
	}

	// for hostServices...
	for _, service := range hostServices {
		wg.Add(1)
		go func(service *HostService) {
			defer wg.Done()
			core.Log.Warnf("Preparing restore for %s", service.Spec)

			// get the list of snapshots locally

			// find the snapshot taken most recently before the time target

			// move the snapshot to the restore directory
		}(service)
	}

	wg.Wait()
}