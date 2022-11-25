package main

import (
	"fmt"
	"time"

	"github.com/jkassis/jerrie/core"
	"github.com/jkassis/jerriedr/cmd/schema"
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"
)

func CMDEnvSnapshotGet(v *viper.Viper, srcArchiveSpecs, dstArchiveSpecs []string) {
	var err error

	// get src and dst archiveSets
	var srcArchiveSet, dstArchiveSet *schema.ArchiveSet
	{
		srcArchiveSet := schema.ArchiveSetNew()
		err := srcArchiveSet.ArchiveAddAll(srcArchiveSpecs, "/backup")
		if err != nil {
			core.Log.Fatalf("could not add srcArchive %v", err)
		}

		dstArchiveSet := schema.ArchiveSetNew()
		err = dstArchiveSet.ArchiveAddAll(dstArchiveSpecs, "")
		if err != nil {
			core.Log.Fatalf("could not add dstArchive %v", err)
		}
	}

	// pick a snapshot set
	var srcArchiveFileSet *schema.ArchiveFileSet
	{
		err = srcArchiveSet.FilesFetch(nil)
		if err != nil {
			core.Log.Fatalf("failed to get files for cluster archive set: %v", err)
		}

		picker := ArchiveFileSetPickerNew().ArchiveSetPut(srcArchiveSet).Run()
		srcArchiveFileSet = picker.SelectedSnapshotArchiveFileSet
	}
	if srcArchiveFileSet == nil {
		core.Log.Fatalf("snapshot not picked... cancelling operation")
	}

	// present a progressWatcher
	progressWatcher := ProgressWatcherNew()
	go progressWatcher.Run()

	// copy files
	{
		core.Log.Warnf("snapshotGet: starting")
		start := time.Now()
		errGroup := errgroup.Group{}
		for _, srcArchiveFile := range srcArchiveFileSet.ArchiveFiles {
			srcArchiveFile := srcArchiveFile
			dstArchive, err := dstArchiveSet.ArchiveGetByService(srcArchiveFile.Archive.ServiceName)
			if err != nil {
				core.Log.Fatalf("couldn't find dstArchive: %v", dstArchive)
			}

			dstArchiveFile := &schema.ArchiveFile{
				Archive: dstArchive,
				Name:    srcArchiveFile.Name,
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
		core.Log.Warnf("snapshotGet: took %s", duration.String())
	}

	progressWatcher.App.Stop()

	// report at the end
	for _, watch := range progressWatcher.watches {
		message := fmt.Sprintf("[ %12d of %12d %s ] %s", watch.progress, watch.total, watch.unit, watch.item)
		core.Log.Warnf(message)
	}
}
