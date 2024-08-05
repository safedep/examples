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

type definition struct {
	idType idType
	name   string
}

func newDefinition(idType idType, name string) *definition {
	return &definition{
		idType: idType,
		name:   name,
	}
}

func (def *definition) id() string {
	return "[" + string(def.idType) + "]" + def.name
}

type namespace struct {
	definition *definition
	parent     *namespace
}

func (ns *namespace) newNamespace(def *definition) *namespace {
	return newNamespace(def, ns)
}

func (ns *namespace) id() string {
	if ns.parent != nil {
		return ns.parent.id() + "/" + ns.definition.id()
	}

	return ns.definition.id()
}

func newNamespace(def *definition, parent *namespace) *namespace {
	return &namespace{
		definition: def,
		parent:     parent,
	}
}

type object struct {
	definition *definition
	ns         *namespace
}

func newObject(def *definition, ns *namespace) *object {
	return &object{
		definition: def,
		ns:         ns,
	}
}

func (obj *object) id() string {
	return obj.ns.id() + "/" + obj.definition.id()
}

type AssignmentGraphBuilder struct {
	// Registry of definitions and objects for mapping
	// Id to structs
	definitionsRegistry map[string]*definition
	objectRegistry      map[string]*object

	// Map of definitions to a set of definitions, forming a graph, where the
	// target node is defined within the parent node
	scope map[string][]string

	// Class hierarchy to model child (key) to parent (value) relationships
	classHierarchy map[string]string

	// Assignment graph holding object to object mapping, modelling the
	// assignment relationship between them
	assignmentGraph map[string]string

	// The current namespace
	currentNamespace *namespace
}

func newAssignmentGraphBuilder(ns *namespace) *AssignmentGraphBuilder {
	return &AssignmentGraphBuilder{
		definitionsRegistry: make(map[string]*definition),
		objectRegistry:      make(map[string]*object),
		scope:               make(map[string][]string),
		classHierarchy:      make(map[string]string),
		assignmentGraph:     make(map[string]string),
		currentNamespace:    ns,
	}
}

func (b *AssignmentGraphBuilder) newDefinition(idType idType, name string) *definition {
	def := newDefinition(idType, name)
	if _, ok := b.definitionsRegistry[def.id()]; !ok {
		b.definitionsRegistry[def.id()] = def
	}

	return def
}

func (b *AssignmentGraphBuilder) newObject(def *definition) *object {
	obj := newObject(def, b.currentNamespace)
	if _, ok := b.objectRegistry[obj.id()]; !ok {
		b.objectRegistry[obj.id()] = obj
	}

	return obj
}

func (b *AssignmentGraphBuilder) switchNamespace(ns *namespace, fn func()) {
	old := b.currentNamespace
	b.currentNamespace = ns

	fn()
	b.currentNamespace = old
}

func (b *AssignmentGraphBuilder) visitClassDefinition(v *Visitor, node *sitter.Node) {
	name := node.ChildByFieldName("name")
	body := node.ChildByFieldName("body")

	// Python3 support multiple inheritance, so we need to handle 'superclasses'

	classDef := b.newDefinition(idTypeClass, v.val(name))

	if (name != nil) && (body != nil) {
		b.switchNamespace(b.currentNamespace.newNamespace(classDef), func() {
			v.visit(body)
		})
	}
}

func (b *AssignmentGraphBuilder) visitFunctionDefinition(v *Visitor, node *sitter.Node) {
	name := node.ChildByFieldName("name")
	body := node.ChildByFieldName("body")

	funcDef := b.newDefinition(idTypeFunction, v.val(name))

	if (name != nil) && (body != nil) {
		b.switchNamespace(b.currentNamespace.newNamespace(funcDef), func() {
			v.visit(body)
		})
	}
}

func (b *AssignmentGraphBuilder) visitCall(v *Visitor, node *sitter.Node) {
	name := node.ChildByFieldName("function")

	if name != nil {
		fmt.Printf("%s -> %s\n", b.currentNamespace.id(), v.val(name))
	}
}

func (b *AssignmentGraphBuilder) visitAssignment(v *Visitor, node *sitter.Node) {
}

type Visitor struct {
	data    []byte
	builder *AssignmentGraphBuilder
}

func newVisitor(data []byte, builder *AssignmentGraphBuilder) *Visitor {
	return &Visitor{
		data:    data,
		builder: builder,
	}
}

func (v *Visitor) val(node *sitter.Node) string {
	start := node.StartByte()
	end := node.EndByte()

	return string(v.data[start:end])
}

// Tree Sitter python grammar
// https://github.com/tree-sitter/tree-sitter-python/blob/master/grammar.js
func (v *Visitor) visit(node *sitter.Node) {
	switch node.Type() {
	case "class_definition":
		v.builder.visitClassDefinition(v, node)
	case "call":
		v.builder.visitCall(v, node)
	case "function_definition":
		v.builder.visitFunctionDefinition(v, node)
	default:
		for i := 0; i < int(node.ChildCount()); i++ {
			v.visit(node.Child(i))
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

	builder := newAssignmentGraphBuilder(newNamespace(newDefinition(idTypeModule, "program"), nil))

	visitor := newVisitor(fileContent, builder)
	visitor.visit(cst.RootNode())
}
