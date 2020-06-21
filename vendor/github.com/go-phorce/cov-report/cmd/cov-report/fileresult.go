package main

type fileResult struct {
	Filename   string  `json:"filename"`
	Total      int     `json:"total"`
	Covered    int     `json:"covered"`
	Uncovered  int     `json:"uncovered"`
	CoveredPct float32 `json:"coveredPercent"`
}

func (fr *fileResult) finish() {
	if fr.Total > 0 {
		fr.CoveredPct = 100 * float32(fr.Covered) / float32(fr.Total)
	}
}

func longestFilename(frs []fileResult) int {
	r := 0
	for _, f := range frs {
		r = max(r, len(f.Filename))
	}
	return r
}
