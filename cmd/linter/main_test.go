package main

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestPanicDetector(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), panicDetector, "./...")
}
