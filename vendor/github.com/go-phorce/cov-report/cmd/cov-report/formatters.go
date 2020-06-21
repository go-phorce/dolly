package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
)

type formatter func(w io.Writer, r *coverageResult)

var formatters = map[string]formatter{
	"txt":  txtFormatter,
	"json": jsonFormatter,
	"xml":  xmlFormatter,
	"ds":   dsFormatter,
}

func keys(m map[string]formatter) []string {
	r := make([]string, 0, len(m))
	for k := range m {
		r = append(r, k)
	}
	sort.Strings(r)
	return r
}

// dsFormatter generates an output suitable for using with the Jenkins display summary plugin
func dsFormatter(w io.Writer, cr *coverageResult) {
	x := xml.NewEncoder(w)
	n := func(local string) xml.Name {
		return xml.Name{Local: local}
	}
	td := func(value, color string) {
		attr := []xml.Attr{
			{Name: n("value"), Value: value},
		}
		if color != "" {
			attr = append(attr, xml.Attr{Name: n("fontcolor"), Value: color})
		}
		x.EncodeToken(xml.StartElement{Name: n("td"), Attr: attr})
		x.EncodeToken(xml.EndElement{Name: n("td")})
	}
	tr := func() {
		x.EncodeToken(xml.StartElement{Name: n("tr")})
	}
	endtr := func() {
		x.EncodeToken(xml.EndElement{Name: n("tr")})
	}
	rowColor := func(label, val, color string) {
		tr()
		td(label, "")
		td(val, color)
		endtr()
	}
	rowN := func(values ...string) {
		tr()
		for _, v := range values {
			td(v, "")
		}
		endtr()
	}
	x.EncodeToken(xml.StartElement{Name: n("section"), Attr: []xml.Attr{
		{Name: n("title"), Value: "Code Coverage"},
	}})
	x.EncodeToken(xml.StartElement{Name: n("table")})
	rowColor("Total Coverage", cr.ResultFormatted, colorOf(cr.Result))
	rowColor("Total Statements", fmt.Sprintf("%d", cr.Total), "black")
	rowColor("Covered Statements", fmt.Sprintf("%d", cr.Covered), "black")
	x.EncodeToken(xml.EndElement{Name: n("table")})

	if len(cr.TopUncovered) > 0 {
		x.EncodeToken(xml.StartElement{Name: n("table")})
		rowN("Filename", "Uncovered Stmts", `% Uncovered [of Total]`)
		for _, fr := range cr.TopUncovered {
			rowN(fr.Filename, strconv.Itoa(fr.Uncovered), fmt.Sprintf("%.1f%%", 100*float32(fr.Uncovered)/float32(cr.Total)))
		}
		x.EncodeToken(xml.EndElement{Name: n("table")})
	}
	x.EncodeToken(xml.EndElement{Name: n("section")})
	x.Flush()
}

func colorOf(percent float32) string {
	if percent >= 90.0 {
		return "green"
	}
	if percent >= 80.0 {
		return "peru"
	}
	return "red"
}

func txtFormatter(w io.Writer, r *coverageResult) {
	fmt.Fprintf(w, "Statements\ntotal:     %d\ncovered:   %d\nuncovered: %d\nexcluded:  %v\nresult:    %v\n",
		r.Total, r.Covered, r.Uncovered, strings.Join(r.ExcludedSources, ","), r.ResultFormatted)
	if len(r.TopUncovered) > 0 {
		fmt.Fprintf(w, "\nTop uncovered source files [name, uncovered count, total uncovered %%]\n")
		longestFilename := longestFilename(r.TopUncovered)
		for _, fr := range r.TopUncovered {
			fmt.Fprintf(w, " %s%s  %5d   %4.1f%%\n", fr.Filename, strings.Repeat(" ", longestFilename-len(fr.Filename)), fr.Uncovered, 100*float32(fr.Uncovered)/float32(r.Total))
		}
	}
}

func jsonFormatter(w io.Writer, r *coverageResult) {
	d, _ := json.MarshalIndent(r, "", "  ")
	w.Write(d)
}

func xmlFormatter(w io.Writer, r *coverageResult) {
	d, _ := xml.MarshalIndent(r, "", "  ")
	w.Write(d)
}
