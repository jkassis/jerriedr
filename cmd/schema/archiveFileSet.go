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
	ArchiveFiles  []*ArchiveFile
	status        ArchvieFileSetStatus
	statusMessage string
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
	nextSeekTime := sss.ArchiveFiles[0].Time.Add(-time.Millisecond)
	for i := 1; i < len(sss.ArchiveFiles); i++ {
		if sss.ArchiveFiles[i].Time.Before(nextSeekTime) {
			nextSeekTime = sss.ArchiveFiles[i].Time
			break
		}
	}

	return nextSeekTime
}

func (sss *ArchiveFileSet) FirstAndLastArchiveFileTime() (first time.Time, last time.Time) {
	first = time.Now()
	last = time.Unix(0, 0)
	for i := 1; i < len(sss.ArchiveFiles); i++ {
		if sss.ArchiveFiles[i].Time.Before(first) {
			first = sss.ArchiveFiles[i].Time
		}
		if sss.ArchiveFiles[i].Time.After(last) {
			last = sss.ArchiveFiles[i].Time
		}
	}

	return first, last
}
