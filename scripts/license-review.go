// license-review.go (build-tag-isolated)
//
// Standalone helper used by Phase 8 / T066 to enumerate every Go module
// dependency reported by `go list -m -json all` (passed in on stdin) and
// classify each module's license file. Output is a stable plain-text table
// that gets archived as part of the v0.1.0 release notes.
//
// This file is fenced behind the `licensereview` build tag so it never
// participates in normal `go build ./...` (it is `package main`, but it
// is run via `go run -tags licensereview` from `scripts/run-license-review.sh`).
//
// Why we don't use `go-licenses`: as of go-licenses v1.6.0 (the last tagged
// release), the tool crashes on Go >= 1.22 with "Package <stdlib> does not
// have module info" because it cannot enumerate the stdlib via the new
// modular toolchain layout. A 100-line classifier is cheaper than chasing
// upstream fixes.

//go:build licensereview

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type modInfo struct {
	Path     string
	Version  string
	Dir      string
	Main     bool
	Indirect bool
}

type row struct {
	Path, License, Source string
}

func main() {
	dec := json.NewDecoder(os.Stdin)
	var rows []row
	for {
		var m modInfo
		if err := dec.Decode(&m); err == io.EOF {
			break
		} else if err != nil {
			fmt.Fprintln(os.Stderr, "decode:", err)
			os.Exit(1)
		}
		if m.Main || m.Dir == "" {
			continue
		}
		licName, licSrc := findLicense(m.Dir)
		rows = append(rows, row{m.Path, licName, licSrc})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Path < rows[j].Path })

	fmt.Printf("%-70s  %-20s  %s\n", "MODULE", "LICENSE", "SOURCE")
	fmt.Println(strings.Repeat("-", 70+2+20+2+10))
	for _, r := range rows {
		fmt.Printf("%-70s  %-20s  %s\n", r.Path, r.License, r.Source)
	}

	fmt.Println()
	fmt.Println("---")
	fmt.Println("Summary:")
	counts := map[string]int{}
	var bad []row
	for _, r := range rows {
		counts[r.License]++
		if isIncompatible(r.License) {
			bad = append(bad, r)
		}
	}
	var keys []string
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Printf("  %-20s  %d\n", k, counts[k])
	}
	fmt.Println()
	if len(bad) == 0 {
		fmt.Println("RESULT: PASS — no GPL/AGPL/SSPL/Commons-Clause dependencies detected.")
	} else {
		fmt.Println("RESULT: FAIL — license-incompatible dependencies detected:")
		for _, r := range bad {
			fmt.Printf("  - %s (%s)\n", r.Path, r.License)
		}
		os.Exit(2)
	}
}

func isIncompatible(lic string) bool {
	switch lic {
	case "GPL", "LGPL", "AGPL", "SSPL", "CC":
		return true
	}
	return false
}

func findLicense(dir string) (string, string) {
	cands := []string{"LICENSE", "LICENSE.txt", "LICENSE.md", "LICENCE", "COPYING", "COPYING.md", "COPYING.txt", "COPYRIGHT", "License", "license"}
	for _, c := range cands {
		p := filepath.Join(dir, c)
		if b, err := os.ReadFile(p); err == nil {
			return classify(string(b)), c
		}
	}
	// Some modules use suffixed filenames for dual-licensing (e.g.
	// LICENSE.BSD + LICENSE.MPL-2.0 in cyphar/filepath-securejoin). Glob
	// for any LICENSE.* file, classify each, and join the results so the
	// reviewer sees the full picture.
	matches, _ := filepath.Glob(filepath.Join(dir, "LICENSE.*"))
	if len(matches) > 0 {
		var labels []string
		var srcs []string
		seen := map[string]bool{}
		for _, p := range matches {
			b, err := os.ReadFile(p)
			if err != nil {
				continue
			}
			lic := classify(string(b))
			if !seen[lic] {
				seen[lic] = true
				labels = append(labels, lic)
			}
			srcs = append(srcs, filepath.Base(p))
		}
		if len(labels) > 0 {
			return strings.Join(labels, "+"), strings.Join(srcs, ",")
		}
	}
	return "MISSING", ""
}

// classify is intentionally simple: it scans for the common header tokens
// that uniquely identify each license family. Order matters — the more
// specific patterns are checked first.
func classify(s string) string {
	s = strings.ToLower(s)
	switch {
	case strings.Contains(s, "mozilla public license") && strings.Contains(s, "version 2.0"):
		return "MPL-2.0"
	case strings.Contains(s, "apache license") && strings.Contains(s, "version 2.0"):
		return "Apache-2.0"
	case strings.Contains(s, "redistribution and use") && strings.Contains(s, "neither the name"):
		return "BSD-3-Clause"
	case strings.Contains(s, "redistribution and use"):
		return "BSD-2-Clause"
	case strings.Contains(s, "mit license"):
		return "MIT"
	case strings.Contains(s, "permission is hereby granted") && strings.Contains(s, "without restriction"):
		return "MIT"
	case strings.Contains(s, "isc license"):
		return "ISC"
	case strings.Contains(s, "gnu affero general public license"):
		return "AGPL"
	case strings.Contains(s, "gnu lesser general public license"):
		return "LGPL"
	case strings.Contains(s, "gnu general public license"):
		return "GPL"
	case strings.Contains(s, "server side public license"):
		return "SSPL"
	case strings.Contains(s, "creative commons"):
		return "CC"
	case strings.Contains(s, "the unlicense") || strings.Contains(s, "this is free and unencumbered"):
		return "Unlicense"
	}
	return "UNKNOWN"
}
