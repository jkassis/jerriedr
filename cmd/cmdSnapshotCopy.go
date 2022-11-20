package main

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/jkassis/jerrie/core"
	"github.com/jkassis/jerriedr/cmd/kube"
	"github.com/jkassis/jerriedr/cmd/schema"
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
			CMDSnapshotCopy(v)
		},
	}

	// flag configuration
	FlagsAddKubeFlags(c, v)
	FlagsAddSrcArchiveFlag(c, v)
	FlagsAddDstArchiveFlag(c, v)

	MAIN.AddCommand(c)
}

func CMDSnapshotCopy(v *viper.Viper) {
	start := time.Now()
	core.Log.Warnf("CMDSnapshotCopy: starting")

	// for each service
	srcArchiveSpec := v.GetString(FLAG_SRC_ARCHIVE)
	dstArchiveSpec := v.GetString(FLAG_DST_ARCHIVE)

	SnapshotCopy(v, srcArchiveSpec, dstArchiveSpec)

	duration := time.Since(start)
	core.Log.Warnf("CMDSnapshotCopy: took %s", duration.String())
}

func SnapshotCopy(v *viper.Viper, srcArchiveSpec, dstArchiveSpec string) (err error) {
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
	srcArchive := &schema.Archive{}
	err = srcArchive.Parse(srcArchiveSpec)
	if err != nil {
		return fmt.Errorf("SnapshotCopy: %v", err)
	}

	if srcArchive.IsStatefulSet() {
		// yes. make sure we have a kube client
		if kubeErr != nil {
			return fmt.Errorf("kube client initialization failed: %v", kubeErr)
		}

		// get the statefulSet
		statefulSet, err := kubeClient.StatefulSetGetByName(srcArchive.KubeNamespace, srcArchive.KubeName)
		if err != nil {
			return fmt.Errorf("could not get pod: %v", err)
		}

		statefulSetFiles := make([]string, 0)

		// for each replica...
		replicas := statefulSet.Spec.Replicas
		for i := 0; i < int(*replicas); i++ {
			podArchive, err := srcArchive.PodArchiveGet(i)
			if err != nil {
				return fmt.Errorf("could not get podArchiveSpec from statefulSetArchiveSpec: %v", err)
			}

			// get the pod
			pod, err := kubeClient.PodGetByName(srcArchive.KubeNamespace, srcArchive.KubeName)
			if err != nil {
				return fmt.Errorf("could not get pod: %v", err)
			}

			podFiles, err := kubeClient.DirLs(&kube.FileSpec{
				PodNamespace: podArchive.KubeNamespace,
				PodName:      podArchive.KubeName,
				Path:         podArchive.Path,
			}, pod, podArchive.KubeContainer)
			if err != nil {
				return fmt.Errorf("could not list files for podSpec %s: %v", podArchive.Spec, err)
			}

			statefulSetFiles = append(statefulSetFiles, podFiles...)
		}

	} else if srcArchive.IsPod() {
		// yes. make sure we have a kube client
		if kubeErr != nil {
			return fmt.Errorf("kube client initialization failed: %v", kubeErr)
		}

		// get the pod
		pod, err := kubeClient.PodGetByName(srcArchive.KubeNamespace, srcArchive.KubeName)
		if err != nil {
			return fmt.Errorf("could not get pod: %v", err)
		}

		// read to the pipe writer
		errGroup.Go(func() error {
			return kubeClient.FileRead(
				&kube.FileSpec{
					PodNamespace: srcArchive.KubeNamespace,
					PodName:      srcArchive.KubeName,
					Path:         srcArchive.Path,
				},
				pipeW,
				pod,
				srcArchive.KubeContainer,
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
	dstArchive := &schema.Archive{}
	err = dstArchive.Parse(dstArchiveSpec)
	if err != nil {
		return fmt.Errorf("SnapshotCopy: %v", err)
	}

	if dstArchive.IsPod() {
		// yes. make sure we have a kube client
		if kubeErr != nil {
			return fmt.Errorf("kube client initialization failed: %v", kubeErr)
		}

		// get the pod
		pod, err := kubeClient.PodGetByName(dstArchive.KubeNamespace, dstArchive.KubeName)
		if err != nil {
			return fmt.Errorf("could not get pod: %v", err)
		}

		errGroup.Go(func() error {
			return kubeClient.FileWrite(
				pipeR,
				&kube.FileSpec{
					PodNamespace: dstArchive.KubeNamespace,
					PodName:      dstArchive.KubeName,
					Path:         dstArchive.Path,
				},
				pod,
				srcArchive.KubeContainer,
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
