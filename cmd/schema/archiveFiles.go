package schema

type ByMostRecent []*ArchiveFile

func (afs ByMostRecent) Len() int {
	return len(afs)
}

func (afs ByMostRecent) Less(i, j int) bool {
	return afs[i].Time.After(afs[j].Time)
}

func (afs ByMostRecent) Swap(i, j int) {
	t := afs[i]
	afs[i] = afs[j]
	afs[j] = t
}

type ByArchiveSpec []*ArchiveFile

func (afs ByArchiveSpec) Len() int {
	return len(afs)
}

func (afs ByArchiveSpec) Less(i, j int) bool {
	return afs[i].Archive.Spec < afs[j].Archive.Spec
}

func (afs ByArchiveSpec) Swap(i, j int) {
	t := afs[i]
	afs[i] = afs[j]
	afs[j] = t
}
