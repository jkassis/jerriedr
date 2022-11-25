package main

import (
	"fmt"
	"time"

	"github.com/jkassis/jerrie/core"
	"github.com/jkassis/jerriedr/cmd/schema"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"
)

func init() {
	// A general configuration object (feed with flags, conf files, etc.)
	v := viper.New()

	// CLI Command with flag parsing
	c := &cobra.Command{
		Use:   "devsnapshotget",
		Short: "Retrieve a snapshot of cluster services and save to a local archive.",
		Long:  `This command is a shortcut for servicesnapshotcopy with several presets.`,
		Run: func(cmd *cobra.Command, args []string) {
			CMDDevSnapshotGet(v)
		},
	}

	FlagsAddKubeFlags(c, v)
	MAIN.AddCommand(c)
}

func CMDDevSnapshotGet(v *viper.Viper) {
	// pick a snapshot set
	var srcArchiveFileSet *schema.ArchiveFileSet
	{
		srcArchiveSet := schema.ArchiveSetNew()
		srcArchiveSet.ArchiveAdd(localDevArchiveSpec)
		err := srcArchiveSet.FilesFetch(nil)
		if err != nil {
			core.Log.Fatalf("failed to get files for cluster archive set: %v", err)
		}

		picker := ArchiveFileSetPickerNew()
		picker.ArchiveSetPut(srcArchiveSet)
		picker.Run()
		srcArchiveFileSet = picker.SelectedSnapshotArchiveFileSet
	}
	if srcArchiveFileSet == nil {
		core.Log.Fatalf("snapshot not picked... cancelling operation")
	}

	// get the dstArchvie
	var dstArchive *schema.Archive
	{
		dstArchive = &schema.Archive{}
		err := dstArchive.Parse(localProdArchiveSpec)
		if err != nil {
			core.Log.Fatalf("could not parse the dstArchiveSpec: %v", err)
		}
	}

	// present a progressWatcher
	progressWatcher := ProgressWatcherNew()
	go progressWatcher.Run()

	// copy files
	{
		core.Log.Warnf("CMDProdSnapshotGet: starting")
		start := time.Now()
		errGroup := errgroup.Group{}
		for _, srcArchiveFile := range srcArchiveFileSet.ArchiveFiles {
			srcArchiveFile := srcArchiveFile
			dstArchiveFile := &schema.ArchiveFile{
				Archive: dstArchive,
				Name:    srcArchiveFile.Archive.Parent.KubeName + "/" + srcArchiveFile.Name,
			}

			errGroup.Go(func() error {
				err := ArchiveFileCopy(v, srcArchiveFile, dstArchiveFile, progressWatcher)
				if err != nil {
					return fmt.Errorf("could not copy archive file: %v", err)
				}
				return nil
			})
		}

		err := errGroup.Wait()
		if err != nil {
			core.Log.Error(err)
		}

		duration := time.Since(start)
		core.Log.Warnf("CMDProdSnapshotGet: took %s", duration.String())
	}

	progressWatcher.App.Stop()

	// report at the end
	for _, watch := range progressWatcher.watches {
		message := fmt.Sprintf("[ %12d of %12d %s ] %s", watch.progress, watch.total, watch.unit, watch.item)
		core.Log.Warnf(message)
	}

}
