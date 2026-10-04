package main

import (
	"encoding/json"
	"fmt"
	"os"
)

const usage = `usage:
  matching match <instance.json>   run deferred-acceptance matching
  matching check <candidate.json>  verify a candidate assignment and find blocking pairs
Input is read from the file argument, or stdin when the argument is "-".`

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}
	cmd, path := os.Args[1], os.Args[2]
	in := os.Stdin
	if path != "-" {
		f, err := os.Open(path)
		if err != nil {
			fatal(err)
		}
		defer f.Close()
		in = f
	}
	switch cmd {
	case "match":
		inst, err := decodeInstance(in)
		if err != nil {
			fatal(err)
		}
		v, err := validate(inst)
		if err != nil {
			fatal(err)
		}
		writeJSON(match(v))
	case "check":
		ci, err := decodeCheckInput(in)
		if err != nil {
			fatal(err)
		}
		v, err := validate(&ci.Instance)
		if err != nil {
			fatal(err)
		}
		rep := check(v, ci)
		writeJSON(rep)
		if !rep.Stable {
			os.Exit(1)
		}
	default:
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}
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
