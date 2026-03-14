package plugin

import (
	"logs-linter/pkg/analyzer"

	"golang.org/x/tools/go/analysis"
)

// New создает новый анализатор для использования в golangci-lint
func New() *analysis.Analyzer {
	return analyzer.Analyzer
}
