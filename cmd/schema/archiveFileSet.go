package schema

import (
	"sort"
	"time"
)

type ArchvieFileSetStatus int

const (
	SSSStatusError ArchvieFileSetStatus = iota
	SSSStatusWarn
	SSSStatusOK
)

func ArchiveFileSetNew() *ArchiveFileSet {
	sss := &ArchiveFileSet{}
	sss.ArchiveFiles = make([]*ArchiveFile, 0)
	return sss
}

type ArchiveFileSet struct {
	ArchiveFiles []*ArchiveFile
}

func (sss *ArchiveFileSet) ArchiveFileAdd(af *ArchiveFile) {
	sss.ArchiveFiles = append(sss.ArchiveFiles, af)
}

func (sss *ArchiveFileSet) SortByMostRecent() {
	sort.Sort(ByMostRecent(sss.ArchiveFiles))
}

func (ss *ArchiveFileSet) EvaluateStatus() {

}

func (sss *ArchiveFileSet) NextSeekTime(t time.Time) time.Time {
	nextSeekTime := t.Add(-time.Millisecond)
	for _, archiveFile := range sss.ArchiveFiles {
		if archiveFile.Time.Before(nextSeekTime) {
			nextSeekTime = archiveFile.Time
			break
		}
	}

	return nextSeekTime
}

func (sss *ArchiveFileSet) FirstAndLastArchiveFileTime() (first time.Time, last time.Time) {
	first = time.Now()
	last = time.Unix(0, 0)
	for _, archiveFile := range sss.ArchiveFiles {
		if archiveFile.Time.Before(first) {
			first = archiveFile.Time
		}
		if archiveFile.Time.After(last) {
			last = archiveFile.Time
		}
	}

	return first, last
}
