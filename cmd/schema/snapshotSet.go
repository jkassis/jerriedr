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
	sss.archiveFiles = make([]*ArchiveFile, 0)
	return sss
}

type SnapshotSet struct {
	archiveFiles  []*ArchiveFile
	status        SnapshotSetStatus
	statusMessage string
}

func (sss *SnapshotSet) ArchiveFileAdd(af *ArchiveFile) {
	sss.archiveFiles = append(sss.archiveFiles, af)
}

func (sss *SnapshotSet) SortByMostRecent() {
	sort.Sort(ByMostRecent(sss.archiveFiles))
}

func (ss *SnapshotSet) EvaluateStatus() {

}

func (sss *SnapshotSet) NextSeekTime(t time.Time) time.Time {
	if len(sss.archiveFiles) == 0 {
		return time.Now()
	}

	nextSeekTime := sss.archiveFiles[0].Time.Add(time.Millisecond)
	for i := 1; i < len(sss.archiveFiles); i++ {
		if sss.archiveFiles[i].Time.After(nextSeekTime) {
			nextSeekTime = sss.archiveFiles[i].Time
			break
		}
	}

	return nextSeekTime
}

func (sss *SnapshotSet) String() string {
	var out string
	for _, archiveFile := range sss.archiveFiles {
		time := archiveFile.Time.Format(time.RFC3339)
		spec := archiveFile.Archive.Spec
		out = out + fmt.Sprintf("   %20s (%s)\n", time, spec)
	}
	return out
}
