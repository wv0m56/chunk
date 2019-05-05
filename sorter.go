package chunk

type byIndex []*indexedC

func (bi byIndex) Len() int {
	return len(bi)
}

func (bi byIndex) Less(i, j int) bool {
	return bi[i].idx < bi[j].idx
}

func (bi byIndex) Swap(i, j int) {
	bi[i], bi[j] = bi[j], bi[i]
}

type indexedC struct {
	C
	idx int
}
