package main

import (
	"fmt"
	"io"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/jkassis/jerrie/core"
	"github.com/jkassis/jerriedr/cmd/schema"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"
)

func init() {
	v := viper.New()

	c := &cobra.Command{
		Use:   "archviefilecopy",
		Short: "Copies snapshots from one archive to another. Archives can be in kube, on hosts, or local.",
		Long:  ``,
		Run: func(cmd *cobra.Command, args []string) {
			CMDArchiveFileCopy(v)
		},
	}

	// flag configuration
	FlagsAddKubeFlags(c, v)
	FlagsAddSrcFlag(c, v)
	FlagsAddDstFlag(c, v)

	MAIN.AddCommand(c)
}

func CMDArchiveFileCopy(v *viper.Viper) {
	start := time.Now()
	core.Log.Warnf("archiveFileCopy: starting")

	// get the archive files
	var srcArchiveFile, dstArchiveFile *schema.ArchiveFile
	{
		srcArchiveFileSpec := v.GetString(FLAG_SRC)
		srcArchiveFile = &schema.ArchiveFile{}
		err := srcArchiveFile.Parse(srcArchiveFileSpec)
		if err != nil {
			core.Log.Fatalf("archiveFileCopy: %v", err)
		}
	}
	{
		dstArchiveFileSpec := v.GetString(FLAG_DST)
		dstArchiveFile = &schema.ArchiveFile{}
		err := dstArchiveFile.Parse(dstArchiveFileSpec)
		if err != nil {
			core.Log.Fatalf("archiveFileCopy: %v", err)
		}
	}

	progressWatcher := ProgressWatcherNew()
	go progressWatcher.Run()
	ArchiveFileCopy(v, srcArchiveFile, dstArchiveFile, progressWatcher)

	duration := time.Since(start)
	core.Log.Warnf("archiveFileCopy: took %s", duration.String())
}

func ArchiveFileCopy(v *viper.Viper, srcArchiveFile, dstArchiveFile *schema.ArchiveFile, progressWatcher *ProgressWatcher) (err error) {
	core.Log.Warnf("starting copy of '%s' to '%s'", srcArchiveFile.Archive.Spec+"/"+srcArchiveFile.Name, dstArchiveFile.Archive.Spec+"/"+dstArchiveFile.Name)

	// get kube client
	kubeClient, kubeErr := KubeClientGet(v)

	// make an eg
	eg := &errgroup.Group{}

	// make a splitter
	progressPipeReader, progressPipeWriter := io.Pipe()
	dstPipeReader, dstPipeWriter := io.Pipe()
	splitWriter := io.MultiWriter(dstPipeWriter, progressPipeWriter)

	// read from the src to the splitter
	var srcFileSize int64
	if srcArchiveFile.Archive.IsStatefulSet() {
		return fmt.Errorf("cannot copy from statefulset archiveFile")
	} else if srcArchiveFile.Archive.IsPod() {
		// yes. make sure we have a kube client
		if kubeErr != nil {
			return fmt.Errorf("kube client initialization failed: %v", kubeErr)
		}

		// get the pod
		pod, err := kubeClient.PodGetByName(srcArchiveFile.Archive.KubeNamespace, srcArchiveFile.Archive.KubeName)
		if err != nil {
			return fmt.Errorf("could not get pod: %v", err)
		}

		// get the file size
		srcFileFullPath := srcArchiveFile.Archive.Path + "/" + srcArchiveFile.Name
		fileStats, err := kubeClient.Stat(pod, srcArchiveFile.Archive.KubeContainer, srcFileFullPath)
		if err != nil {
			return fmt.Errorf("could not get stats for %s: %v", srcFileFullPath, err)
		}
		srcFileSize = fileStats.Size

		// read to the splitter
		eg.Go(func() error {
			err := kubeClient.FileRead(srcFileFullPath, splitWriter, pod,
				srcArchiveFile.Archive.KubeContainer)
			if err != nil {
				return fmt.Errorf("trouble with file read while copying file from kube: %v", err)
			}
			if err := progressPipeWriter.Close(); err != nil {
				return fmt.Errorf("could not close progressPipeWriter: %v", err)
			}
			if err := dstPipeWriter.Close(); err != nil {
				return fmt.Errorf("could not close dstPipeWriter: %v", err)
			}
			return nil
		})
	} else if srcArchiveFile.Archive.IsLocal() {
		srcFileFullPath := srcArchiveFile.Archive.Path + "/" + srcArchiveFile.Name

		// get the file size
		fileInfo, err := os.Stat(srcFileFullPath)
		if err != nil {
			return fmt.Errorf("could not get stats for %s: %v", srcFileFullPath, err)
		}
		srcFileSize = fileInfo.Size()

		srcFile, err := os.Open(srcFileFullPath)
		if err != nil {
			return err
		}

		eg.Go(func() (err error) {
			_, err = io.Copy(splitWriter, srcFile)
			if err != nil {
				return fmt.Errorf("trouble reading local file: %v", err)
			}
			if err := progressPipeWriter.Close(); err != nil {
				return fmt.Errorf("could not close progressPipeWriter: %v", err)
			}
			if err := dstPipeWriter.Close(); err != nil {
				return fmt.Errorf("could not close dstPipeWriter: %v", err)
			}
			return nil
		})
	}

	// read the progress into a discard buffer
	// TODO wish that one could read without coping bytes
	var progressUpdater func(progress int64)
	dstFileFullPath := dstArchiveFile.Archive.Path + "/" + srcArchiveFile.Name
	progressUpdater = progressWatcher.AddWatch(&Watch{item: dstFileFullPath, unit: "bytes", total: srcFileSize})
	eg.Go(func() error {
		discardBuffer := make([]byte, 8192) // seems to be standard size for blackhole
		for {
			n, err := progressPipeReader.Read(discardBuffer)
			progressUpdater(int64(n))
			if err != nil {
				if err == io.EOF {
					return nil
				}
				return err
			}
		}
	})

	writeToPod := func(podName string) error {
		// yes. make sure we have a kube client
		if kubeErr != nil {
			return fmt.Errorf("kube client initialization failed: %v", kubeErr)
		}

		// get the pod
		pod, err := kubeClient.PodGetByName(dstArchiveFile.Archive.KubeNamespace, dstArchiveFile.Archive.KubeName)
		if err != nil {
			return fmt.Errorf("could not get pod: %v", err)
		}

		// read into the kube file writer
		eg.Go(func() error {
			return kubeClient.FileWrite(
				dstPipeReader,
				dstArchiveFile.Archive.Path+"/"+dstArchiveFile.Name,
				pod,
				srcArchiveFile.Archive.KubeContainer,
			)
		})
		return nil
	}

	// setup the dstArchive first
	if dstArchiveFile.Archive.IsStatefulSet() {
		n, err := dstArchiveFile.Archive.Replicas(kubeClient)
		if err != nil {
			return err
		}
		for i := 0; i < n; i++ {
			if err = writeToPod(dstArchiveFile.Archive.KubeName +
				"-" +
				strconv.Itoa(i)); err != nil {
				return err
			}
		}
	} else if dstArchiveFile.Archive.IsPod() {
		if err = writeToPod(dstArchiveFile.Archive.KubeName); err != nil {
			return err
		}
	} else if dstArchiveFile.Archive.IsLocal() {
		dstFilePath := dstArchiveFile.Archive.Path + "/" + dstArchiveFile.Name
		dstDir := path.Dir(dstFilePath)
		err := os.MkdirAll(dstDir, os.ModePerm)
		if err != nil {
			return fmt.Errorf("could not make directory '%s': %v", dstDir, err)
		}

		// open the dstFile
		dstFile, err := os.OpenFile(dstFilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return fmt.Errorf("could not open file  '%s': %v", dstFilePath, err)
		}

		// read into the local file
		eg.Go(func() error {
			_, err := io.Copy(dstFile, dstPipeReader)
			if err != nil {
				return fmt.Errorf("copy error from dstPipeReader to dstFile: %v", err)
			}

			err = dstFile.Sync()
			if err != nil {
				return fmt.Errorf("sync error for dstFile: %v", err)
			}
			err = dstFile.Close()
			if err != nil {
				return fmt.Errorf("close error for dstFile: %v", err)
			}
			return err
		})
	}

	return eg.Wait()
}
