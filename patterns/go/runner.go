package main

// Entry point of the Go port.
//
// Two harnesses live in this directory and both stay runnable, because a release
// must carry its own evidence and the previous release's:
//
//	go run . v07   # the release's conformance suite (harness_v07.go), default
//	go run . v06   # the v0.6 harness (main.go), historical evidence
//
// `v07` is the default so that a bare `go run .` — and any tool that builds and
// runs the port without arguments — exercises the current release.

import (
	"fmt"
	"os"
)

func main() {
	which := "v07"
	if len(os.Args) > 1 {
		which = os.Args[1]
	}
	switch which {
	case "v07":
		runHarnessV07()
	case "v06":
		runHarnessV06()
	case "dump":
		// Print the canonical text of the release fixture's declaration and both
		// digests — the tool that makes a cross-port mismatch a diff instead of a
		// mystery. Mirrors patterns/python/reference_digest.py.
		orch := NewDOFOrchestrator(0.05)
		orch.mapper.PollEnvironment(FixtureScene(FixtureOptions{}))
		decl := orch.mapper.LastDeclaration
		fmt.Println(decl.CanonicalText())
		fmt.Printf("RULER %s\n", decl.Digest())
		if ctx := orch.mapper.LastObservation; ctx != nil {
			fmt.Printf("OBSERVATION %s\n", ctx.ObservationDigest)
		}
	default:
		fmt.Printf("unknown harness %q: expected v07 (default) or v06\n", which)
		os.Exit(2)
	}
	if len(failures) > 0 {
		os.Exit(1)
	}
}
