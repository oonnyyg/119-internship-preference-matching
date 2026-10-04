package main

import (
	"encoding/json"
	"fmt"
	"os"
)

func usage() {
	fmt.Fprintln(os.Stderr, "usage:")
	fmt.Fprintln(os.Stderr, "  matcher match <input.json>")
	fmt.Fprintln(os.Stderr, "  matcher check <input.json> <assignment.json>")
	os.Exit(2)
}

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	switch os.Args[1] {
	case "match":
		if len(os.Args) != 3 {
			usage()
		}
		in := mustLoadInput(os.Args[2])
		writeJSON(Match(in))
	case "check":
		if len(os.Args) != 4 {
			usage()
		}
		in := mustLoadInput(os.Args[2])
		f, err := os.Open(os.Args[3])
		if err != nil {
			fatal(err)
		}
		defer f.Close()
		af, err := LoadAssignments(f)
		if err != nil {
			fatal(err)
		}
		rep := Check(in, af)
		writeJSON(rep)
		if !rep.Valid || !rep.Stable {
			os.Exit(1)
		}
	default:
		usage()
	}
}

func mustLoadInput(path string) *Input {
	f, err := os.Open(path)
	if err != nil {
		fatal(err)
	}
	defer f.Close()
	in, err := LoadInput(f)
	if err != nil {
		fatal(err)
	}
	return in
}

func writeJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(2)
}
