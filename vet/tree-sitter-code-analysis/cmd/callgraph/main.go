package main

import (
	"context"
	"fmt"
	"io"
	"os"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/python"
)

type idType string

var (
	idTypeFunction idType = "function"
	idTypeClass    idType = "class"
	idTypeVariable idType = "variable"
	idTypeModule   idType = "module"
)

type namespace struct {
	name   string
	parent *namespace
}

func (ns *namespace) String() string {
	if ns.parent != nil {
		return ns.parent.String() + "/" + ns.name
	}

	return ns.name
}

type identifier struct {
	idType idType
	name   string
	ns     *namespace
}

func (id *identifier) String() string {
	return id.ns.String() + "/" + id.name
}

type Visitor struct {
	data []byte
}

func (v *Visitor) val(node *sitter.Node) string {
	start := node.StartByte()
	end := node.EndByte()

	return string(v.data[start:end])
}

// Tree Sitter python grammar
// https://github.com/tree-sitter/tree-sitter-python/blob/master/grammar.js
func (v *Visitor) visit(ns *namespace, node *sitter.Node, depth int) {
	switch node.Type() {
	case "class_definition":
		name := node.ChildByFieldName("name")
		body := node.ChildByFieldName("body")

		v.visit(&namespace{parent: ns, name: v.val(name)}, body, depth+1)

	case "assignment":
		left := node.ChildByFieldName("left")
		right := node.ChildByFieldName("right")

		v.visit(ns, right, depth+1)

		if (left != nil) && (right != nil) {
			fmt.Printf("%s = %s\n", v.val(left), v.val(right))
		}

	case "call":
		site := node.ChildByFieldName("function")
		if site != nil {
			fmt.Printf("%s -> %s\n", ns.String(), v.val(site))
		}
	case "function_definition":
		name := node.ChildByFieldName("name")
		body := node.ChildByFieldName("body")

		if (name != nil) && (body != nil) {
			v.visit(&namespace{parent: ns, name: v.val(name)}, body, depth+1)
		}
	default:
		for i := 0; i < int(node.ChildCount()); i++ {
			v.visit(ns, node.Child(i), depth)
		}
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <file.py>\n", os.Args[0])
		os.Exit(1)
	}

	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())

	file, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening file: %s\n", err)
		os.Exit(1)
	}

	defer file.Close()

	fileContent, err := io.ReadAll(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %s\n", err)
		os.Exit(1)
	}

	cst, err := parser.ParseCtx(context.Background(), nil, fileContent)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing file: %s\n", err)
		os.Exit(1)
	}

	if cst.RootNode() == nil {
		fmt.Fprintf(os.Stderr, "Error parsing file: root node is nil\n")
		os.Exit(1)
	}

	visitor := &Visitor{
		data: fileContent,
	}

	visitor.visit(&namespace{name: "program"}, cst.RootNode(), 0)
}
