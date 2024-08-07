package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/python"
)

type idType string

var (
	idTypeFunction idType = "function"
	idTypeClass    idType = "class"
	idTypeVariable idType = "variable"
	idTypeModule   idType = "module"

	idTypeLiteral idType = "literal"
	idTypeUnknown idType = "unknown"
)

type definition struct {
	idType idType
	name   string

	// The namespace where this definition was created
	ns *namespace

	// The scope created for this definition (if any)
	scope *scope
}

func newDefinition(ns *namespace, idType idType, name string) *definition {
	return &definition{
		idType: idType,
		name:   name,
		ns:     ns,
	}
}

func (def *definition) id() string {
	if def.ns != nil {
		return def.ns.id() + "/" + def.name + "[" + string(def.idType) + "]"
	} else {
		return def.name + "[" + string(def.idType) + "]"
	}
}

// List of definitions, forming a namespace. Represents a location
// in code where definitions are created
type namespace struct {
	// The definition that this namespace is associated with
	// There can be multiple definitions in a namespace
	definition *definition

	// The scope of this namespace
	scope *scope

	// The parent namespace
	parent *namespace
}

func (ns *namespace) newNamespace(def *definition, scope *scope) *namespace {
	return newNamespace(def, scope, ns)
}

func (ns *namespace) id() string {
	if ns.parent != nil {
		return ns.parent.id() + "/" + ns.definition.name
	} else {
		return ns.definition.name
	}
}

func newNamespace(def *definition, scope *scope, parent *namespace) *namespace {
	ns := &namespace{
		definition: def,
		parent:     parent,
		scope:      scope,
	}

	return ns
}

type scope struct {
	owner  *definition
	defs   map[string]*definition
	parent *scope
}

func newScope(parent *scope, owner *definition) *scope {
	s := &scope{
		defs:   make(map[string]*definition),
		parent: parent,
		owner:  owner,
	}

	return s
}

func (s *scope) id() string {
	if s.owner != nil {
		return s.owner.id()
	}

	return "global"
}

func (s *scope) newScope(owner *definition) *scope {
	return newScope(s, owner)
}

type AssignmentGraph struct {
	// Map of objects to an element to an element of the power set of objects
	edges map[string]map[string]bool
}

type AssignmentGraphBuilder struct {
	// Registry of definitions and objects for mapping
	// Id to structs
	definitionsRegistry map[string]*definition

	// Map of definitions to a set of definitions, forming a graph, where the
	// target node is defined within the parent node
	scope *scope

	// Class hierarchy to model child (key) to parent (value) relationships
	classHierarchy map[string][]string

	// Assignment graph holding object to object mapping, modelling the
	// assignment relationship between them
	assignmentGraph map[string][]string

	// The current namespace
	currentNamespace *namespace
}

func newAssignmentGraphBuilder(ns *namespace) *AssignmentGraphBuilder {
	return &AssignmentGraphBuilder{
		definitionsRegistry: make(map[string]*definition),
		classHierarchy:      make(map[string][]string),
		assignmentGraph:     make(map[string][]string),
		scope:               newScope(nil, nil),
		currentNamespace:    ns,
	}
}

func (b *AssignmentGraphBuilder) newDefinition(idType idType, name string) *definition {
	def := newDefinition(b.currentNamespace, idType, name)
	if _, ok := b.definitionsRegistry[def.id()]; !ok {
		b.definitionsRegistry[def.id()] = def
	}

	b.scope.defs[def.id()] = def
	return def
}

func (b *AssignmentGraphBuilder) switchNamespace(ns *namespace, fn func()) {
	old := b.currentNamespace
	b.currentNamespace = ns

	fn()
	b.currentNamespace = old
}

func (b *AssignmentGraphBuilder) switchScope(scope *scope, fn func()) {
	old := b.scope
	b.scope = scope

	fn()
	b.scope = old
}

func (b *AssignmentGraphBuilder) newScope(def *definition, fn func()) {
	scope := newScope(b.scope, def)

	old := b.scope
	b.scope = scope

	def.scope = scope

	// A scope switch will always switch namespace
	b.switchNamespace(b.currentNamespace.newNamespace(def, scope), func() {
		fn()
	})

	// Restore the scope
	b.scope = old
}

func (b *AssignmentGraphBuilder) assignmentEdge(from, to *definition) {
	if _, ok := b.assignmentGraph[from.id()]; !ok {
		b.assignmentGraph[from.id()] = make([]string, 0)
	}

	b.assignmentGraph[from.id()] = append(b.assignmentGraph[from.id()], to.id())
}

// Find in scope by name (binding)
func (b *AssignmentGraphBuilder) findInScope(name string) (*definition, bool) {
	for scope := b.scope; scope != nil; scope = scope.parent {
		fmt.Printf("Searching for %s in scope: %s\n", name, b.scope.id())

		for _, def := range scope.defs {
			if def.name == name {
				return def, true
			}
		}
	}

	return nil, false
}

// Find attributed name in scope
func (b *AssignmentGraphBuilder) findAttributedNameInScope(name string) (*definition, bool) {
	attributes := strings.Split(name, ".")
	if len(attributes) == 0 {
		return nil, false
	}

	var def *definition
	var ok bool

	// Check if the first attribute is in scope
	if def, ok = b.findInScope(attributes[0]); !ok {
		return nil, false
	}

	for _, attr := range attributes[1:] {
		fmt.Printf("Searching for %s in %s\n", attr, def.id())

		found := false

		scope := def.scope
		if scope == nil {
			scope = def.ns.scope
		}

		if scope == nil {
			return nil, false
		}

		for _, childDef := range scope.defs {
			if childDef.name == attr {
				def = childDef
				found = true
			}
		}

		if !found {
			return nil, false
		}
	}

	return def, true
}

func (b *AssignmentGraphBuilder) eval(v *Visitor, node *sitter.Node) (*definition, error) {
	return v.visit(node)
}

func (b *AssignmentGraphBuilder) visitClassDefinition(v *Visitor, node *sitter.Node) (*definition, error) {
	name := node.ChildByFieldName("name")
	body := node.ChildByFieldName("body")
	superclasses := node.ChildByFieldName("superclasses")

	if (name == nil) || (body == nil) {
		return nil, fmt.Errorf("Invalid class definition")
	}

	classDef := b.newDefinition(idTypeClass, v.val(name))
	if (superclasses != nil) && (superclasses.ChildCount() > 0) {
		if superclasses.Child(0).Type() == "(" {
			// Handle grouping, skip the ( and ) nodes
			for i := 1; i < int(superclasses.ChildCount()-1); i++ {
				superClassDef := b.newDefinition(idTypeClass, v.val(superclasses.Child(i)))
				b.classHierarchy[classDef.id()] = append(b.classHierarchy[classDef.id()], superClassDef.id())

				// Skip the "," node
				i++
			}
		} else {
			// Handle single superclass
			superClassDef := b.newDefinition(idTypeClass, v.val(superclasses.Child(0)))
			b.classHierarchy[classDef.id()] = append(b.classHierarchy[classDef.id()], superClassDef.id())
		}
	}

	var err error
	b.newScope(classDef, func() {
		_, err = v.visit(body)
	})

	fmt.Printf("Class: %s defined in scope: %s\n", classDef.name, b.scope.id())

	return classDef, err
}

func (b *AssignmentGraphBuilder) visitFunctionDefinition(v *Visitor, node *sitter.Node) (*definition, error) {
	name := node.ChildByFieldName("name")
	body := node.ChildByFieldName("body")

	if (name == nil) || (body == nil) {
		return nil, fmt.Errorf("Invalid function definition")
	}

	var err error
	funcDef := b.newDefinition(idTypeFunction, v.val(name))

	b.newScope(funcDef, func() {
		params := node.ChildByFieldName("parameters")
		if params != nil {
			// Params are a group, surrounded by ( and )
			for i := 1; i < int(params.ChildCount()-1); i++ {
				// Bind the param to the function scope
				b.newDefinition(idTypeVariable, v.val(params.Child(i)))

				// Skip the "," node
				i++
			}
		}

		_, err = v.visit(body)
	})

	return funcDef, err
}

func (b *AssignmentGraphBuilder) visitReturnStatement(v *Visitor, node *sitter.Node) (*definition, error) {
	fmt.Printf("Visiting return statement with child count: %d\n", node.ChildCount())

	// https://github.com/tree-sitter/tree-sitter-python/blob/master/grammar.js#L235
	if node.ChildCount() > 1 {
		expr := node.Child(1)
		exprDef, err := b.eval(v, expr)

		if err != nil {
			return nil, err
		}

		retDef := b.newDefinition(idTypeVariable, "__ret")
		b.assignmentEdge(retDef, exprDef)

		return retDef, nil
	}

	return b.newDefinition(idTypeUnknown, "nil_return"), nil
}

func (b *AssignmentGraphBuilder) visitCall(v *Visitor, node *sitter.Node) (*definition, error) {
	name := node.ChildByFieldName("function")
	if name == nil {
		return nil, fmt.Errorf("Invalid call")
	}

	calleeName := v.val(name)

	fmt.Printf("%s -> %s@%s\n", b.currentNamespace.id(),
		b.scope.id(),
		calleeName)

	// Lookup callee in scope
	if calleeDef, ok := b.findAttributedNameInScope(calleeName); ok {
		var retDef *definition = b.newDefinition(idTypeUnknown, fmt.Sprintf("__call_%s_ret", calleeName))

		fmt.Printf("Found callee: %s\n", calleeDef.id())

		// If the callee is a class constructor, we need to resolve the
		// __init__ method
		if calleeDef.idType == idTypeClass {
			fmt.Printf("Callee is a class constructor\n")

			// TODO: Resolve __init__ method in class hierarchy

			b.switchScope(calleeDef.scope, func() {
				if initDef, ok := b.findInScope("__init__"); ok {
					calleeDef = initDef
					retDef = initDef
				} else {
					calleeDef = b.newDefinition(idTypeUnknown, fmt.Sprintf("__class_init_%s", calleeName))
					retDef = calleeDef
				}
			})
		}

		args := node.ChildByFieldName("arguments")
		if (args != nil) && (args.ChildCount() > 0) {
			// Check if args is of type argument_list
			if args.Child(0).Type() == "argument_list" {
				// Skip the ( and ) nodes
				for i := 1; i < int(args.ChildCount()-1); i++ {
					arg := args.Child(i)
					argDef, err := b.eval(v, arg)
					if err != nil {
						return nil, err
					}

					b.assignmentEdge(calleeDef, argDef)

					// Skip the "," node
					i++
				}
			}
		}

		b.switchScope(calleeDef.scope, func() {
			if r, ok := b.findInScope("__ret"); ok {
				retDef = r
			}
		})

		return retDef, nil
	}

	return b.newDefinition(idTypeUnknown, fmt.Sprintf("__call_%s", calleeName)), nil
}

func (b *AssignmentGraphBuilder) visitAssignment(v *Visitor, node *sitter.Node) (*definition, error) {
	left := node.ChildByFieldName("left")
	right := node.ChildByFieldName("right")

	if (left == nil) || (right == nil) {
		return nil, fmt.Errorf("Invalid assignment")
	}

	leftDef, err := b.eval(v, left)
	if err != nil {
		return nil, err
	}

	rightDef, err := b.eval(v, right)
	if err != nil {
		return nil, err
	}

	fmt.Printf("left: %v right: %v\n", leftDef, rightDef)

	// Add assignment to graph
	b.assignmentEdge(leftDef, rightDef)

	return leftDef, nil
}

func (b *AssignmentGraphBuilder) visitLiteral(v *Visitor, node *sitter.Node) (*definition, error) {
	return b.newDefinition(idTypeLiteral, v.val(node)), nil
}

func (b *AssignmentGraphBuilder) visitExpressionStatement(v *Visitor, node *sitter.Node) (*definition, error) {
	var def *definition
	var err error

	for i := 0; i < int(node.ChildCount()); i++ {
		def, err = v.visit(node.Child(i))
		if err != nil {
			return nil, err
		}
	}

	if def == nil {
		return b.newDefinition(idTypeUnknown, "nil_expression"), nil
	}

	return def, err
}

func (b *AssignmentGraphBuilder) visitIdentifier(v *Visitor, node *sitter.Node) (*definition, error) {
	name := v.val(node)
	return b.newDefinition(idTypeVariable, name), nil
}

func (b *AssignmentGraphBuilder) visitAttributeExpression(_ *Visitor, _ *sitter.Node) (*definition, error) {
	fmt.Printf("Visiting attribute expression\n")
	return b.newDefinition(idTypeUnknown, "attribute"), nil
}

func (b *AssignmentGraphBuilder) visitList(v *Visitor, node *sitter.Node) (*definition, error) {
	if node.ChildCount() < 2 {
		return nil, fmt.Errorf("Invalid list")
	}

	for i := 1; i < int(node.ChildCount()-1); i++ {
		_, err := b.eval(v, node.Child(i))
		if err != nil {
			return nil, err
		}
	}

	return b.newDefinition(idTypeUnknown, "list"), nil
}

// Module definition, identifies the root node of the AST
func (b *AssignmentGraphBuilder) visitModule(v *Visitor, node *sitter.Node) (*definition, error) {
	for i := 0; i < int(node.ChildCount()); i++ {
		_, err := b.eval(v, node.Child(i))
		if err != nil {
			return nil, err
		}
	}

	return b.newDefinition(idTypeModule, "module"), nil
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
func (v *Visitor) visit(node *sitter.Node) (*definition, error) {
	switch node.Type() {
	case "class_definition":
		return v.builder.visitClassDefinition(v, node)
	case "call":
		return v.builder.visitCall(v, node)
	case "function_definition":
		return v.builder.visitFunctionDefinition(v, node)
	case "assignment":
		return v.builder.visitAssignment(v, node)
	case "identifier":
		return v.builder.visitIdentifier(v, node)
	case "expression_statement":
		return v.builder.visitExpressionStatement(v, node)
	case "number", "integer", "string", "boolean":
		return v.builder.visitLiteral(v, node)
	case "attribute":
		return v.builder.visitAttributeExpression(v, node)
	case "return_statement":
		return v.builder.visitReturnStatement(v, node)
	case "module":
		return v.builder.visitModule(v, node)
	case "list":
		return v.builder.visitList(v, node)
	default:
		fmt.Printf("Visiting node: %s\n", node.Type())

		var err error
		var def *definition = v.builder.newDefinition(idTypeUnknown, node.Type())

		// Recursively visit children without evaluation
		// We will return the last evaluated value
		for i := 0; i < int(node.ChildCount()); i++ {
			def, err = v.visit(node.Child(i))
		}

		return def, err
	}
}

// Convert file path to module name.
// Example samples/4.py to samples.4
func fileToModuleName(file string) string {
	name := filepath.Clean(file)
	name = strings.ReplaceAll(file, "/", ".")

	if strings.HasSuffix(name, ".py") {
		name = name[:len(name)-3]
	}

	return name
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <file.py>\n", os.Args[0])
		os.Exit(1)
	}

	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())

	programDef := newDefinition(nil, idTypeModule, fileToModuleName(os.Args[1]))
	programNs := newNamespace(programDef, nil, nil)

	builder := newAssignmentGraphBuilder(programNs)

	loadModule := func(path string, builder *AssignmentGraphBuilder) error {
		file, err := os.Open(path)
		if err != nil {
			return err
		}

		defer file.Close()

		fileContent, err := io.ReadAll(file)
		if err != nil {
			return err
		}

		cst, err := parser.ParseCtx(context.Background(), nil, fileContent)
		if err != nil {
			return err
		}

		if cst.RootNode() == nil {
			return fmt.Errorf("Error parsing file: root node is nil")
		}

		visitor := newVisitor(fileContent, builder)

		_, err = visitor.visit(cst.RootNode())
		if err != nil {
			return err
		}

		return nil
	}

	err := loadModule(os.Args[1], builder)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading module: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("Assignment Graph:\n")

	jsonGraph, err := json.MarshalIndent(builder.assignmentGraph, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshalling assignment graph: %s\n", err)
	} else {
		fmt.Println(string(jsonGraph))
	}
}
