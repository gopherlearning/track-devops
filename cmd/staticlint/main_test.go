//go:build ignore
// +build ignore

package main

import (
	"testing"

	"github.com/gopherlearning/track-devops/internal/staticlint"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestMyAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), staticlint.OsExitAnalyzer, "./...")
}
