package schema

import (
	"fmt"
	"time"

	"github.com/jkassis/jerrie/core"
	"github.com/jkassis/jerriedr/cmd/kube"
	"golang.org/x/sync/errgroup"
)

func ArchiveSetNew() *ArchiveSet {
	archiveSet := &ArchiveSet{}
	archiveSet.Archives = make([]*Archive, 0)
	return archiveSet
}

type ArchiveSet struct {
	Archives []*Archive
	seekTime time.Time
	sss      *ArchiveFileSet
}

func (as *ArchiveSet) ArchiveAdd(archiveSpec string) (archive *Archive, err error) {
	archive = ArchiveNew()
	err = archive.Parse(archiveSpec)
	if err != nil {
		return nil, err
	}
	as.Archives = append(as.Archives, archive)
	return archive, nil
}

func (as *ArchiveSet) ArchiveAddAll(archiveSpecs []string, pathSuffix string) error {
	for _, archiveSpec := range archiveSpecs {
		a, err := as.ArchiveAdd(archiveSpec)
		if err != nil {
			return err
		}
		a.Path += pathSuffix
	}
	return nil
}

func (as *ArchiveSet) ArchiveGetByService(service string) (a *Archive, err error) {
	for _, archive := range as.Archives {
		if archive.ServiceName == service {
			return archive, nil
		}
	}

	archiveNames := make([]string, 0)
	for _, archive := range as.Archives {
		archiveNames = append(archiveNames, archive.ServiceName)
	}
	return nil, fmt.Errorf("could not find archive for service '%s' have only these... %v", service, archiveNames)
}

func (as *ArchiveSet) PickSnapshot() (err error) {
	// let the user pick a srcArchiveFileSet (snapshot)
	var srcArchiveFileSet *ArchiveFileSet
	{
		err = as.FilesFetch(nil)
		if err != nil {
			core.Log.Fatalf("failed to get files for cluster archive set: %v", err)
		}

		if !as.HasFiles() {
			core.Log.Fatalf("found no snapshots in %v", as)
		}

		picker := ArchiveFileSetPickerNew().ArchiveSetPut(as).Run()
		srcArchiveFileSet = picker.SelectedSnapshotArchiveFileSet

		if srcArchiveFileSet == nil {
			core.Log.Fatalf("snapshot not picked... cancelling operation")
		}
	}

	return nil
}

func (as *ArchiveSet) FilesFetch(kubeClient *kube.Client) error {
	eg := errgroup.Group{}

	for _, archive := range as.Archives {
		archive := archive
		eg.Go(func() error {
			return archive.FilesFetch(kubeClient)
		})
	}

	return eg.Wait()
}

func (as *ArchiveSet) HasFiles() bool {
	var hasFiles bool
	for _, srcArchive := range as.Archives {
		if len(srcArchive.Files) > 0 {
			hasFiles = true
			break
		}
	}
	return hasFiles
}

func (as *ArchiveSet) SeekTo(t time.Time) {
	as.seekTime = t
	as.sss = nil
}

func (as *ArchiveSet) ArchiveFileSetGetNext() *ArchiveFileSet {
	if as.sss != nil {
		as.seekTime = as.sss.NextSeekTime(as.seekTime)
	}

	// make the next ArchiveFileSet
	sss := ArchiveFileSetNew()
	for _, a := range as.Archives {
		archiveFile := a.FileGetFilteredBefore(as.seekTime)
		if archiveFile != nil {
			sss.ArchiveFileAdd(archiveFile)
		}
	}
	if len(sss.ArchiveFiles) == 0 {
		return nil
	}
	sss.SortByMostRecent()
	sss.EvaluateStatus()
	as.sss = sss
	return sss
}

func (as *ArchiveSet) FilterAdd(tf *TimeFilter) {
	for _, a := range as.Archives {
		a.Filters = append(a.Filters, tf)
	}
}

func (as *ArchiveSet) FiltersClear() {
	for _, a := range as.Archives {
		a.Filters = make([]*TimeFilter, 0)
	}
}
