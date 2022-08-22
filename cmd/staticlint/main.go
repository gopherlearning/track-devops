package main

import (
	"strings"

	_ "github.com/golangci/golangci-lint/pkg/golinters/goanalysis"
	"github.com/gopherlearning/track-devops/internal/staticlint"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"honnef.co/go/tools/staticcheck"
)

func main() {
	// gocrA := goanalysis.DummyRun()

	mychecks := []*analysis.Analyzer{
		// gocrA,
		staticlint.OsExitAnalyzer,
	}

	// добавляем анализаторы из staticcheck, которые указаны в файле конфигурации
	for _, v := range staticcheck.Analyzers {

		if strings.HasPrefix(v.Analyzer.Name, "SA") {
			mychecks = append(mychecks, v.Analyzer)
		}
	}

	// if v, ok := staticcheck.Analyzers; ok {
	// 	mychecks = append(mychecks, v)
	// }
	for _, v := range staticcheck.Analyzers {
		if v.Analyzer.Name == "ST1003" {
			mychecks = append(mychecks, v.Analyzer)
			break
		}
	}

	// Adding all analyzers from Golang-ci

	mychecks = append(mychecks, staticlint.AllAnalyzers...)

	multichecker.Main(
		mychecks...,
	)
}
