package chunk

// byReverseIndex implements sort.Interface (DESCENDING).
type byReverseIndex []*indexedC

func (bri byReverseIndex) Len() int {
	return len(bri)
}

func (bri byReverseIndex) Less(i, j int) bool {
	return bri[i].idx > bri[j].idx
}

func (bri byReverseIndex) Swap(i, j int) {
	bri[i], bri[j] = bri[j], bri[i]
}

type indexedC struct {
	C
	idx int
}
