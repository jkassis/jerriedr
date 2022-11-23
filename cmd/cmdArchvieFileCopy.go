package main

import (
	"fmt"
	"io"
	"os"
	"path"
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
	core.Log.Warnf("CMDArchiveFileCopy: starting")

	// get the archive files
	var srcArchiveFile, dstArchiveFile *schema.ArchiveFile
	{
		srcArchiveFileSpec := v.GetString(FLAG_SRC)
		srcArchiveFile = &schema.ArchiveFile{}
		err := srcArchiveFile.Parse(srcArchiveFileSpec)
		if err != nil {
			core.Log.Fatalf("CMDArchiveFileCopy: %v", err)
		}
	}
	{
		dstArchiveFileSpec := v.GetString(FLAG_DST)
		dstArchiveFile = &schema.ArchiveFile{}
		err := dstArchiveFile.Parse(dstArchiveFileSpec)
		if err != nil {
			core.Log.Fatalf("CMDArchiveFileCopy: %v", err)
		}
	}

	progressWatcher := ProgressWatcherNew()
	go progressWatcher.Run()
	ArchiveFileCopy(v, srcArchiveFile, dstArchiveFile, progressWatcher)

	duration := time.Since(start)
	core.Log.Warnf("CMDArchiveFileCopy: took %s", duration.String())
}

func ArchiveFileCopy(v *viper.Viper, srcArchiveFile, dstArchiveFile *schema.ArchiveFile, progressWatcher *ProgressWatcher) (err error) {
	core.Log.Warnf("starting copy of '%s' to '%s'", srcArchiveFile.Archive.Spec+"/"+srcArchiveFile.Name, dstArchiveFile.Archive.Spec+"/"+dstArchiveFile.Name)

	// get kube client
	kubeClient, kubeErr := KubeClientGet(v)

	// make an eg
	eg := &errgroup.Group{}

	// make a pipe
	srcReader, srcWriter := io.Pipe()
	var progressUpdater func(progress int64)

	// read from the src to the pipe
	var dstFileSize int64
	var dstFileFullPath string
	if srcArchiveFile.Archive.IsStatefulSet() {
		return fmt.Errorf("cannot copy to/from statefulset archiveFile")
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

		dstFileFullPath = srcArchiveFile.Archive.Path + "/" + srcArchiveFile.Name

		// get the file size
		fileStats, err := kubeClient.FileStatGet(pod, srcArchiveFile.Archive.KubeContainer, dstFileFullPath)
		if err != nil {
			return fmt.Errorf("could not get stats for %s: %v", dstFileFullPath, err)
		}
		dstFileSize = fileStats.Size

		// read to the pipe writer
		eg.Go(func() error {
			return kubeClient.FileRead(
				&kube.FileSpec{
					PodNamespace: srcArchiveFile.Archive.KubeNamespace,
					PodName:      srcArchiveFile.Archive.KubeName,
					Path:         dstFileFullPath,
				},
				srcWriter,
				pod,
				srcArchiveFile.Archive.KubeContainer,
			)
		})
	} else if srcArchiveFile.Archive.IsLocal() {
		dstFileFullPath = srcArchiveFile.Archive.Path + "/" + srcArchiveFile.Name

		// get the file size
		fileInfo, err := os.Stat(dstFileFullPath)
		if err != nil {
			return fmt.Errorf("could not get stats for %s: %v", dstFileFullPath, err)
		}
		dstFileSize = fileInfo.Size()

		f, err := os.Open(dstFileFullPath)
		if err != nil {
			return err
		}

		defer f.Close()
		eg.Go(func() error {
			_, err := io.Copy(srcWriter, f)
			return err
		})
	}

	// setup the progressUpdater with some fancy pipes
	progressUpdater = progressWatcher.AddWatch(&Watch{item: dstFileFullPath, unit: "bytes", total: dstFileSize})
	dstWriterReader, dstWriterWriter := io.Pipe()
	progressUpdaterReader, progressUpdaterWriter := io.Pipe()

	// read the src into the splitter
	splitter := io.MultiWriter(dstWriterWriter, progressUpdaterWriter)
	eg.Go(func() error {
		_, err := io.Copy(splitter, srcReader)
		progressUpdaterWriter.Close()
		dstWriterWriter.Close()
		return err
	})

	// read the progress into discard and count bytes
	eg.Go(func() error {
		discardBuffer := make([]byte, 8192) // seems to be standard size for blackhole
		for {
			n, err := progressUpdaterReader.Read(discardBuffer)
			if err != nil {
				return err
			}
			progressUpdater(int64(n))
		}
	})

	// setup the dstArchive first
	if dstArchiveFile.Archive.IsStatefulSet() {
		return fmt.Errorf("cannot copy to/from statefulset archiveFile")
	} else if dstArchiveFile.Archive.IsPod() {
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
				dstWriterReader,
				&kube.FileSpec{
					PodNamespace: dstArchiveFile.Archive.KubeNamespace,
					PodName:      dstArchiveFile.Archive.KubeName,
					Path:         dstArchiveFile.Archive.Path + "/" + dstArchiveFile.Name,
				},
				pod,
				srcArchiveFile.Archive.KubeContainer,
			)
		})
	} else if dstArchiveFile.Archive.IsLocal() {
		dstPathFull := dstArchiveFile.Archive.Path + "/" + dstArchiveFile.Name
		dstDir := path.Dir(dstPathFull)
		err := os.MkdirAll(dstDir, os.ModePerm)
		if err != nil {
			return fmt.Errorf("could not make directory '%s': %v", dstDir, err)
		}

		// open the dstFile
		dstFile, err := os.OpenFile(dstPathFull, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return fmt.Errorf("could not open file  '%s': %v", dstPathFull, err)
		}
		defer func() {
			dstFile.Sync()
			dstFile.Close()
		}()

		// read into the local file
		eg.Go(func() error {
			_, err := io.Copy(dstFile, dstWriterReader)
			return err
		})
	}

	return eg.Wait()
}
