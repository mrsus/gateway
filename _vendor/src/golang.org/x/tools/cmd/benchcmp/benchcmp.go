// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"
	"text/tabwriter"

	"golang.org/x/tools/benchmark/parse"
)

var (
	changedOnly = flag.Bool("changed", false, "show only benchmarks that have changed")
	magSort     = flag.Bool("mag", false, "sort benchmarks by magnitude of change")
	best        = flag.Bool("best", false, "compare best times from old and new")
)

const usageFooter = `
Each input file should be from:
	go test -run=NONE -bench=. > [old,new].txt

Benchcmp compares old and new for each benchmark.

If -test.benchmem=true is added to the "go test" command
benchcmp will also compare memory allocations.
`

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s old.txt new.txt\n\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Fprint(os.Stderr, usageFooter)
		os.Exit(2)
	}
	flag.Parse()
	if flag.NArg() != 2 {
		flag.Usage()
	}

	before := parseFile(flag.Arg(0))
	after := parseFile(flag.Arg(1))

	cmps, warnings := Correlate(before, after)

	for _, warn := range warnings {
		fmt.Fprintln(os.Stderr, warn)
	}

	if len(cmps) == 0 {
		fatal("benchcmp: no repeated benchmarks")
	}

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 0, 5, ' ', 0)
	defer w.Flush()

	var header bool // Has the header has been displayed yet for a given block?

	if *magSort {
		sort.Sort(ByDeltaNsOp(cmps))
	} else {
		sort.Sort(ByParseOrder(cmps))
	}
	for _, cmp := range cmps {
		if !cmp.Measured(parse.NsOp) {
			continue
		}
		if delta := cmp.DeltaNsOp(); !*changedOnly || delta.Changed() {
			if !header {
				fmt.Fprint(w, "benchmark\told ns/op\tnew ns/op\tdelta\n")
				header = true
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", cmp.Name(), formatNs(cmp.Before.NsOp), formatNs(cmp.After.NsOp), delta.Percent())
		}
	}

	header = false
	if *magSort {
		sort.Sort(ByDeltaMbS(cmps))
	}
	for _, cmp := range cmps {
		if !cmp.Measured(parse.MbS) {
			continue
		}
		if delta := cmp.DeltaMbS(); !*changedOnly || delta.Changed() {
			if !header {
				fmt.Fprint(w, "\nbenchmark\told MB/s\tnew MB/s\tspeedup\n")
				header = true
			}
			fmt.Fprintf(w, "%s\t%.2f\t%.2f\t%s\n", cmp.Name(), cmp.Before.MbS, cmp.After.MbS, delta.Multiple())
		}
	}

	header = false
	if *magSort {
		sort.Sort(ByDeltaAllocsOp(cmps))
	}
	for _, cmp := range cmps {
		if !cmp.Measured(parse.AllocsOp) {
			continue
		}
		if delta := cmp.DeltaAllocsOp(); !*changedOnly || delta.Changed() {
			if !header {
				fmt.Fprint(w, "\nbenchmark\told allocs\tnew allocs\tdelta\n")
				header = true
			}
			fmt.Fprintf(w, "%s\t%d\t%d\t%s\n", cmp.Name(), cmp.Before.AllocsOp, cmp.After.AllocsOp, delta.Percent())
		}
	}

	header = false
	if *magSort {
		sort.Sort(ByDeltaBOp(cmps))
	}
	for _, cmp := range cmps {
		if !cmp.Measured(parse.BOp) {
			continue
		}
		if delta := cmp.DeltaBOp(); !*changedOnly || delta.Changed() {
			if !header {
				fmt.Fprint(w, "\nbenchmark\told bytes\tnew bytes\tdelta\n")
				header = true
			}
			fmt.Fprintf(w, "%s\t%d\t%d\t%s\n", cmp.Name(), cmp.Before.BOp, cmp.After.BOp, cmp.DeltaBOp().Percent())
		}
	}
}

func fatal(msg interface{}) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}

func parseFile(path string) parse.BenchSet {
	f, err := os.Open(path)
	if err != nil {
		fatal(err)
	}
	defer f.Close()
	bb, err := parse.ParseBenchSet(f)
	if err != nil {
		fatal(err)
	}
	if *best {
		selectBest(bb)
	}
	return bb
}

func selectBest(bs parse.BenchSet) {
	for name, bb := range bs {
		if len(bb) < 2 {
			continue
		}
		ord := bb[0].Ord
		best := bb[0]
		for _, b := range bb {
			if b.NsOp < best.NsOp {
				b.Ord = ord
				best = b
			}
		}
		bs[name] = []*parse.Bench{best}
	}
}

// formatNs formats ns measurements to expose a useful amount of
// precision. It mirrors the ns precision logic of testing.B.
func formatNs(ns float64) string {
	prec := 0
	switch {
	case ns < 10:
		prec = 2
	case ns < 100:
		prec = 1
	}
	return strconv.FormatFloat(ns, 'f', prec, 64)
}
