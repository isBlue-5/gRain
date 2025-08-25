package processor

import (
	"go/ast"

	"github.com/grain-framework/grain/pkg/annotation/registry"
)

type EntityProcessor struct {
	registry   registry.Registry
	parser     AnnotationParser
	outputPath string
	verbose    bool
}

func NewEntityProcessor(reg registry.Registry, parser AnnotationParser, outputPath string) *EntityProcessor {
	return &EntityProcessor{
		registry:   reg,
		parser:     parser,
		outputPath: outputPath,
		verbose:    false,
	}
}

func (p *EntityProcessor) WithVerbose(verbose bool) *EntityProcessor {
	p.verbose = verbose
	return p
}

func (p *EntityProcessor) ProcessEntity(paths []string) error {
	return nil
}

func extractDocComment(node ast.Node) string {
	return ""
}
