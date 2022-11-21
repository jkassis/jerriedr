package schema

import (
	"fmt"
	"sort"
	"time"
)

type SnapshotSetStatus int

const (
	SSSStatusError SnapshotSetStatus = iota
	SSSStatusWarn
	SSSStatusOK
)

func SnapshotSetNew() *SnapshotSet {
	sss := &SnapshotSet{}
	sss.ArchiveFiles = make([]*ArchiveFile, 0)
	return sss
}

type SnapshotSet struct {
	ArchiveFiles  []*ArchiveFile
	status        SnapshotSetStatus
	statusMessage string
}

func (sss *SnapshotSet) ArchiveFileAdd(af *ArchiveFile) {
	sss.ArchiveFiles = append(sss.ArchiveFiles, af)
}

func (sss *SnapshotSet) SortByMostRecent() {
	sort.Sort(ByMostRecent(sss.ArchiveFiles))
}

func (ss *SnapshotSet) EvaluateStatus() {

}

func (sss *SnapshotSet) NextSeekTime(t time.Time) time.Time {
	nextSeekTime := sss.ArchiveFiles[0].Time.Add(-time.Millisecond)
	for i := 1; i < len(sss.ArchiveFiles); i++ {
		if sss.ArchiveFiles[i].Time.Before(nextSeekTime) {
			nextSeekTime = sss.ArchiveFiles[i].Time
			break
		}
	}

	return nextSeekTime
}

func (sss *SnapshotSet) String() string {
	var out string
	for _, archiveFile := range sss.ArchiveFiles {
		time := archiveFile.Time.Format(time.RFC3339)
		spec := archiveFile.Archive.Spec
		out = out + fmt.Sprintf("   %20s (%s)\n", time, spec)
	}
	return out
}
