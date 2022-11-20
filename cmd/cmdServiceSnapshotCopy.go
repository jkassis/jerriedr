package main

import (
	"io"
	"os"
	"time"

	"github.com/jkassis/jerrie/core"
	"github.com/jkassis/jerriedr/cmd/kube"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"
)

func init() {
	v := viper.New()

	c := &cobra.Command{
		Use:   "servicesnapshotcopy",
		Short: "Copies snapshots from one archive to another. Archives can be in kube, on hosts, or local.",
		Long:  ``,
		Run: func(cmd *cobra.Command, args []string) {
			CMDServiceSnapshotCopy(v)
		},
	}

	// flag configuration
	FlagsAddKubeFlags(c, v)
	FlagsAddSrcArchiveFlag(c, v)
	FlagsAddDstArchiveFlag(c, v)

	MAIN.AddCommand(c)
}

func CMDServiceSnapshotCopy(v *viper.Viper) {
	start := time.Now()
	core.Log.Warnf("CMDServiceSnapshotCopy: starting")

	// for each service
	srcArchiveSpec := v.GetString(FLAG_SRC_ARCHIVE)
	dstArchiveSpec := v.GetString(FLAG_DST_ARCHIVE)

	ServiceSnapshotCopy(v, srcArchiveSpec, dstArchiveSpec)

	duration := time.Since(start)
	core.Log.Warnf("CMDServiceSnapshotCopy: took %s", duration.String())
}

func ServiceSnapshotCopy(v *viper.Viper, srcArchiveSpec, dstArchiveSpec string) (err error) {
	// get http protocol and backup protocol version
	protocol := v.GetString(FLAG_PROTOCOL)
	core.Log.Infof("got protocol: %s", protocol)
	version := v.GetString(FLAG_VERSION)
	core.Log.Infof("got version: %s", version)

	// get kube client
	kubeClient, kubeErr := KubeClientGet(v)

	// make an errGroup
	errGroup := &errgroup.Group{}

	// make a pipe
	pipeR, pipeW := io.Pipe()

	// read from the src to the pipe
	srcArchive := &Archive{}
	err = srcArchive.Parse(srcArchiveSpec)
	if err != nil {
		core.Log.Fatalf("ServiceSnapshotCopy: %v", err)
	}

	if srcArchive.IsPod() {
		// yes. make sure we have a kube client
		if kubeErr != nil {
			core.Log.Fatalf("kube client initialization failed: %v", kubeErr)
		}

		// get the pod
		pod, err := kubeClient.GetPod(srcArchive.PodName, srcArchive.PodNamespace)
		if err != nil {
			core.Log.Fatalf("could not get pod: %v", err)
		}

		// read to the pipe writer
		errGroup.Go(func() error {
			return kubeClient.FileRead(
				&kube.FileSpec{
					PodNamespace: srcArchive.PodNamespace,
					PodName:      srcArchive.PodName,
					File:         srcArchive.Path,
				},
				pipeW,
				pod,
				srcArchive.PodContainer,
			)
		})
	} else if srcArchive.IsLocal() {
		f, err := os.Open(srcArchive.Path)
		if err != nil {
			return err
		}

		defer f.Close()
		errGroup.Go(func() error {
			_, err := io.Copy(pipeW, f)
			return err
		})
	}

	// write from the pipe to the dst
	dstArchive := &Archive{}
	err = dstArchive.Parse(dstArchiveSpec)
	if err != nil {
		core.Log.Fatalf("ServiceSnapshotCopy: %v", err)
	}

	if dstArchive.IsPod() {
		// yes. make sure we have a kube client
		if kubeErr != nil {
			core.Log.Fatalf("kube client initialization failed: %v", kubeErr)
		}

		// get the pod
		pod, err := kubeClient.GetPod(dstArchive.PodName, dstArchive.PodNamespace)
		if err != nil {
			core.Log.Fatalf("could not get pod: %v", err)
		}

		errGroup.Go(func() error {
			return kubeClient.FileWrite(
				pipeR,
				&kube.FileSpec{
					PodNamespace: dstArchive.PodNamespace,
					PodName:      dstArchive.PodName,
					File:         dstArchive.Path,
				},
				pod,
				srcArchive.PodContainer,
			)
		})
	} else if dstArchive.IsLocal() {
		// open the dstFile
		f, err := os.OpenFile(dstArchive.Path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}
		defer func() {
			f.Sync()
			f.Close()
		}()

		errGroup.Go(func() error {
			_, err := io.Copy(f, pipeR)
			return err
		})
	}

	return errGroup.Wait()
}
