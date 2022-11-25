package schema

import (
	"fmt"
	"time"

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

func (as *ArchiveSet) ArchiveAdd(archiveSpec string) (a *Archive, err error) {
	archive := ArchiveNew()
	err = archive.Parse(archiveSpec)
	if err != nil {
		return nil, err
	}
	as.Archives = append(as.Archives, archive)
	return a, nil
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

	return nil, fmt.Errorf("could not find archive for service '%s' have only these... %v", service, as.Archives)
}

func (as *ArchiveSet) FilesFetch(kubeClient *kube.KubeClient) error {
	eg := errgroup.Group{}

	for _, archive := range as.Archives {
		archive := archive
		eg.Go(func() error {
			return archive.FilesFetch(kubeClient)
		})
	}

	return eg.Wait()
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
