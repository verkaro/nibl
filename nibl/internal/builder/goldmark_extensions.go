// internal/builder/goldmark_extensions.go
package builder

import (
	"bytes"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// mdLinkTransformer is a struct that implements the goldmark ASTTransformer interface.
// Its purpose is to walk the document's Abstract Syntax Tree (AST) and modify link nodes.
type mdLinkTransformer struct {
}

// newMDLinkTransformer creates a new instance of our custom transformer.
func newMDLinkTransformer() parser.ASTTransformer {
	return &mdLinkTransformer{}
}

// Transform is the method called by Goldmark to apply our custom logic.
// It walks the AST and calls a function for each node.
func (t *mdLinkTransformer) Transform(node *ast.Document, reader text.Reader, pc parser.Context) {
	ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		// We only need to process nodes when "entering" them during the walk.
		if !entering {
			return ast.WalkContinue, nil
		}

		// Check if the current node is a link.
		link, ok := n.(*ast.Link)
		if !ok {
			// If it's not a link, continue to the next node.
			return ast.WalkContinue, nil
		}

		// Get the link destination (the URL part).
		dest := link.Destination
		// Check if the destination ends with ".md".
		if bytes.HasSuffix(dest, []byte(".md")) {
			// If it does, replace the .md extension with .html.
			newDest := bytes.TrimSuffix(dest, []byte(".md"))
			newDest = append(newDest, []byte(".html")...)
			link.Destination = newDest
		}
		return ast.WalkContinue, nil
	})
}

