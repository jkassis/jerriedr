package util

import (
	"fmt"
	"time"

	"github.com/jkassis/jerrie/core"
	"github.com/jkassis/jerriedr/cmd/kube"
	"github.com/jkassis/jerriedr/cmd/schema"
	"github.com/jkassis/jerriedr/cmd/ui"
	"golang.org/x/sync/errgroup"
)

// EnvCopy gets a list of source snapshots, prompts the user
// to select one and copies the snapshot to the destination env.
func EnvCopy(kubeClient *kube.Client, srcArchiveSpecs, dstArchiveSpecs []string) {
	var err error

	// get src and dst archiveSets
	var srcArchiveSet, dstArchiveSet *schema.ArchiveSet
	{
		srcArchiveSet = schema.ArchiveSetNew()
		err := srcArchiveSet.ArchiveAddAll(srcArchiveSpecs, "/backup")
		if err != nil {
			core.Log.Fatalf("could not add srcArchive %v", err)
		}

		dstArchiveSet = schema.ArchiveSetNew()
		err = dstArchiveSet.ArchiveAddAll(dstArchiveSpecs, "")
		if err != nil {
			core.Log.Fatalf("could not add dstArchive %v", err)
		}
	}

	// pick a snapshot set
	var srcArchiveFileSet *schema.ArchiveFileSet
	{
		err = srcArchiveSet.FilesFetch(kubeClient)
		if err != nil {
			core.Log.Fatalf("failed to get files for cluster archive set: %v", err)
		}

		var hasFiles bool
		for _, srcArchive := range srcArchiveSet.Archives {
			if len(srcArchive.Files) > 0 {
				hasFiles = true
				break
			}
		}
		if !hasFiles {
			core.Log.Fatalf("found no snapshots in %v", srcArchiveSpecs)
		}

		picker := ArchiveFileSetPickerNew().
			ArchiveSetPut(srcArchiveSet).
			Run()
		srcArchiveFileSet = picker.SelectedSnapshotArchiveFileSet
	}
	if srcArchiveFileSet == nil {
		core.Log.Fatalf("snapshot not picked... cancelling operation")
	}

	// present a progressWatcher
	progressWatcher := ui.ProgressWatcherNew()
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
				err := ArchiveFileCopy(kubeClient, srcArchiveFile, dstArchiveFile, progressWatcher)
				if err != nil {
					return fmt.Errorf("could not copy archive file: %v", err)
				}
				return nil
			})
		}

		err := errGroup.Wait()
		if err != nil {
			core.Log.Fatalf("problem with copy: %v", err)
		}

		duration := time.Since(start)
		core.Log.Warnf("snapshotGet: took %s", duration.String())
	}

	progressWatcher.App.Stop()

	// report at the end
	for _, watch := range progressWatcher.Watches {
		message := fmt.Sprintf(
			"[ %12d of %12d %s ] %s",
			watch.Progress,
			watch.Total,
			watch.Unit,
			watch.Item)
		core.Log.Warnf(message)
	}
}
