package main

import (
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"os"
	r "regexp"
	"sort"
	"strings"

	"github.com/go-phorce/cov-report/version"
	"golang.org/x/tools/cover"
)

func main() {
	os.Exit(realMain(os.Stdout, os.Args))
}

func realMain(outw io.WriteCloser, args []string) int {
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	flags.SetOutput(outw)
	ver := flags.Bool("v", false, "Print version")
	exc := flags.String("ex", "", "A regex to exclude files from the calculation [file names that match the regex are excluded]")
	format := flags.String("fmt", "txt", fmt.Sprintf("What format do you want the results in (%v)", strings.Join(keys(formatters), ", ")))
	out := flags.String("o", "", "Filename to write the results to (or stdout if not set)")
	cc := flags.String("cc", "", "Filename to write the combined coverage details to")
	uncovered := flags.Int("u", 10, "Number of top uncovered files to list")

	if err := flags.Parse(args[1:]); err != nil {
		fmt.Fprintf(outw, "%s: %s", args[0], err)
		return 2
	}

	if *ver {
		fmt.Printf("cov-report %v\n", version.Current())
		os.Exit(0)
	}

	files := flags.Args()
	if len(files) == 0 {
		fmt.Fprint(outw, "Please specify one or more coverprofile files to parse\n")
		return 2
	}
	var err error
	var exclude *r.Regexp
	if *exc != "" {
		exclude, err = r.Compile(*exc)
		if err != nil {
			fmt.Fprintf(outw, "Error compiling exclusion regex (%v) : %v\n", *exc, err)
			return 2
		}
	}
	fmtr, exists := formatters[*format]
	if !exists {
		fmt.Fprintf(outw, "Specified formatter %v doesn't exist, valid options are %v\n", *format, strings.Join(keys(formatters), ", "))
		return 2
	}
	writer := outw
	if *out != "" {
		writer, err = os.Create(*out)
		if err != nil {
			fmt.Fprintf(outw, "Unable to create output file %v: %v\n", *out, err)
			return 2
		}
	}
	defer writer.Close()
	a := newCoverageAccumulator()
	for _, f := range files {
		if err := a.parse(f, exclude); err != nil {
			fmt.Fprintf(outw, "%v\n", err)
			return 1
		}
	}
	a.calculate()
	if len(*cc) > 0 {
		a.dumpCombined(*cc)
	}
	fmtr(writer, a.result(*uncovered))
	return 0
}

// blockKey is a compound key to a block of code
type blockKey struct {
	filename            string
	startLine, startCol int
	endLine, endCol     int
}

// blockKeys allows for sorting by name,startLine,startCol,endLine
type blockKeys []blockKey

func (b blockKeys) Len() int {
	return len(b)
}

func (b blockKeys) Less(i, j int) bool {
	x := b[i]
	y := b[j]
	if x.filename == y.filename {
		if x.startLine == y.startLine {
			if x.startCol == y.startCol {
				return x.endLine < y.endLine
			}
			return x.startCol < y.startCol
		}
		return x.startLine < y.startLine
	}
	return x.filename < y.filename
}

func (b blockKeys) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

func makeKey(p *cover.Profile, b *cover.ProfileBlock) blockKey {
	return blockKey{
		p.FileName,
		b.StartLine, b.StartCol,
		b.EndLine, b.EndCol,
	}
}

// blockState is the coverage state of a particular block of code
type blockState struct {
	numStmt  int
	count    int
	excluded bool
}

type coverageAccumulator struct {
	blocks          map[blockKey]*blockState
	mode            string
	Total           int      `json:"total"`
	Covered         int      `json:"covered"`
	Uncovered       int      `json:"uncovered"`
	Excluded        int      `json:"excluded"`
	ExcludedSources []string `json:"excludedSources,omitempty"`

	excluded map[string]bool
	files    []fileResult
}

func newCoverageAccumulator() *coverageAccumulator {
	return &coverageAccumulator{
		blocks:   make(map[blockKey]*blockState),
		excluded: make(map[string]bool),
	}
}

type coverageResult struct {
	coverageAccumulator
	XMLName         xml.Name     `xml:"CoverageResult" json:"-"`
	Result          float32      `json:"result"`
	ResultFormatted string       `json:"resultFormatted"`
	TopUncovered    []fileResult `json:"topUncoveredFiles"`
}

func (a *coverageAccumulator) result(numTopUncovered int) *coverageResult {
	var pc float32
	if a.Total > 0 {
		pc = 100 * float32(a.Covered) / float32(a.Total)
	}
	r := coverageResult{
		coverageAccumulator: *a,
		Result:              pc,
		ResultFormatted:     fmt.Sprintf("%.1f%%", pc),
		TopUncovered:        a.files[:min(len(a.files), numTopUncovered)],
	}
	return &r
}

// dumpCombined will take the combined aggreagate coverage details and write them out to a new file in the coverage profile format
func (a *coverageAccumulator) dumpCombined(fn string) {
	f, err := os.Create(fn)
	if err != nil {
		fmt.Printf("Unable to create combined coverage output file %v: %v", fn, err)
		os.Exit(1)
	}
	defer f.Close()
	fmt.Fprintf(f, "mode: %s\n", a.mode)
	keys := make([]blockKey, 0, len(a.blocks))
	for bk, bs := range a.blocks {
		if !bs.excluded {
			keys = append(keys, bk)
		}
	}
	sort.Sort(blockKeys(keys))
	for _, bk := range keys {
		bs := a.blocks[bk]
		fmt.Fprintf(f, "%s:%d.%d,%d.%d %d %d\n", bk.filename, bk.startLine, bk.startCol, bk.endLine, bk.endCol, bs.numStmt, bs.count)
	}
}

func (a *coverageAccumulator) calculate() {
	byFile := make(map[string]*fileResult)
	for bk, bs := range a.blocks {
		if bs.excluded {
			a.Excluded += bs.numStmt
		} else {
			a.Total += bs.numStmt
			if bs.count > 0 {
				a.Covered += bs.numStmt
			} else {
				a.Uncovered += bs.numStmt
			}

			fr, exists := byFile[bk.filename]
			if !exists {
				fr = new(fileResult)
				fr.Filename = bk.filename
				byFile[bk.filename] = fr
			}
			fr.Total += bs.numStmt
			if bs.count > 0 {
				fr.Covered += bs.numStmt
			} else {
				fr.Uncovered += bs.numStmt
			}
		}
	}
	a.ExcludedSources = make([]string, 0, len(a.excluded))
	for k := range a.excluded {
		a.ExcludedSources = append(a.ExcludedSources, k)
	}

	a.files = make([]fileResult, 0, len(byFile))
	for _, fr := range byFile {
		if fr.Uncovered > 0 {
			fr.finish()
			a.files = append(a.files, *fr)
		}
	}
	sort.Slice(a.files, func(i, j int) bool {
		return a.files[i].Uncovered > a.files[j].Uncovered
	})
}

func (a *coverageAccumulator) parse(file string, exclude *r.Regexp) error {
	cp, err := cover.ParseProfiles(file)
	if err != nil {
		return fmt.Errorf("unable to parse coverage file %v: %v", file, err)
	}
	for _, p := range cp {
		if a.mode != "" && p.Mode != a.mode {
			return fmt.Errorf("file %v has cover Mode %v, but previously set to %v, all files must use same mode", p.FileName, p.Mode, a.mode)
		}
		a.mode = p.Mode
		ex := false
		if exclude != nil && exclude.MatchString(p.FileName) {
			a.excluded[p.FileName] = true
			ex = true
		}
		// multiple source profiles might refer to the same block of source code
		// so we need to aggreagate them into a single value for that block
		for _, b := range p.Blocks {
			k := makeKey(p, &b)
			bs, exists := a.blocks[k]
			if !exists {
				bs = new(blockState)
				a.blocks[k] = bs
			}
			bs.excluded = ex
			// todo, how we update this should depend on mode
			bs.count = max(bs.count, b.Count)
			bs.numStmt = b.NumStmt
		}
	}
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
