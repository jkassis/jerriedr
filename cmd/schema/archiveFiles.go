package schema

type ByMostRecent []*ArchiveFile

func (afbt ByMostRecent) Len() int {
	return len(afbt)
}

func (afbt ByMostRecent) Less(i, j int) bool {
	return afbt[i].Time.After(afbt[j].Time)
}

// Swap swaps the elements with indexes i and j.
func (afbt ByMostRecent) Swap(i, j int) {
	t := afbt[i]
	afbt[i] = afbt[j]
	afbt[j] = t
}
