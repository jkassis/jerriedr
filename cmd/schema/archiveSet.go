package schema

import (
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

func (as *ArchiveSet) ArchiveAdd(archiveSpec string) error {
	archive := ArchiveNew()
	err := archive.Parse(archiveSpec)
	if err != nil {
		return err
	}
	as.Archives = append(as.Archives, archive)
	return nil
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
