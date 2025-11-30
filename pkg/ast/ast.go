package ast

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.
import . "github.com/marshallburns/ez/pkg/tokenizer"

// Node is the base interface for all AST nodes
type Node interface {
	TokenLiteral() string
}

// Statement represents a statement node
type Statement interface {
	Node
	statementNode()
}

// Expression represents an expression node
type Expression interface {
	Node
	expressionNode()
}

// Program is the root node of every AST
type Program struct {
	Module     *ModuleDeclaration // Optional module declaration (first statement if present)
	FileUsing  []*UsingStatement  // File-scoped using declarations
	Statements []Statement
}

// Visibility represents the access level of a declaration
type Visibility int

const (
	VisibilityPublic        Visibility = iota // Public (default)
	VisibilityPrivate                         // Private to this file
	VisibilityPrivateModule                   // Private to this module (all files in directory)
)

// ModuleDeclaration represents "module mymodule" at the top of a file
type ModuleDeclaration struct {
	Token Token
	Name  *Label // Module name (e.g., "server", "utils")
}

func (m *ModuleDeclaration) statementNode()       {}
func (m *ModuleDeclaration) TokenLiteral() string { return m.Token.Literal }

func (p *Program) TokenLiteral() string {
	if len(p.Statements) > 0 {
		return p.Statements[0].TokenLiteral()
	}
	return ""
}

// ============================================================================
// Expressions
// ============================================================================

// A "Label" is just my way of describing
// an "Identifier" like most languages do.
type Label struct {
	Token Token
	Value string
}

// Takes in a pointer to a Lable(l)
func (l *Label) expressionNode()      {}
func (l *Label) TokenLiteral() string { return l.Token.Literal }

// Represents an integer value
type IntegerValue struct {
	Token Token
	Value int64
}

// Takes in a pointer to an IntegerValue(v)
func (v *IntegerValue) expressionNode()      {}
func (v *IntegerValue) TokenLiteral() string { return v.Token.Literal }

// Represents a float value
type FloatValue struct {
	Token Token
	Value float64
}

// Takes in a pointer to an FloatValue(v)
func (v *FloatValue) expressionNode()      {}
func (v *FloatValue) TokenLiteral() string { return v.Token.Literal }

// Represents a string value
type StringValue struct {
	Token Token
	Value string
}

// Takes in a pointer to an StringValue(v)
func (v *StringValue) expressionNode()      {}
func (v *StringValue) TokenLiteral() string { return v.Token.Literal }

// InterpolatedString represents a string with embedded expressions like "Hello, ${name}!"
type InterpolatedString struct {
	Token Token
	Parts []Expression // alternating StringValue and Expression nodes
}

func (i *InterpolatedString) expressionNode()      {}
func (i *InterpolatedString) TokenLiteral() string { return i.Token.Literal }

// CharValue represents a character value
type CharValue struct {
	Token Token
	Value rune
}

// Takes in a pointer to an CharValue(v)
func (v *CharValue) expressionNode()      {}
func (v *CharValue) TokenLiteral() string { return v.Token.Literal }

// BooleanValue represents true/false
type BooleanValue struct {
	Token Token
	Value bool
}

// Takes in a pointer to an BooleanValue(v)
func (v *BooleanValue) expressionNode()      {}
func (v *BooleanValue) TokenLiteral() string { return v.Token.Literal }

// NilValue represents nil
type NilValue struct {
	Token Token
	// nil has no value...
}

// Takes in a pointer to an NilValue(v)
func (v *NilValue) expressionNode()      {}
func (v *NilValue) TokenLiteral() string { return v.Token.Literal }

// Represents an array literal {1, 2, 3, etc....}
type ArrayValue struct {
	Token    Token
	Elements []Expression
}

func (v *ArrayValue) expressionNode()      {}
func (v *ArrayValue) TokenLiteral() string { return v.Token.Literal }

// MapPair represents a key-value pair in a map literal
type MapPair struct {
	Key   Expression
	Value Expression
}

// MapValue represents a map literal {"key": value, ...}
type MapValue struct {
	Token Token
	Pairs []*MapPair
}

func (m *MapValue) expressionNode()      {}
func (m *MapValue) TokenLiteral() string { return m.Token.Literal }

// StructValue represents Person{name: "Bob", age: 25}
type StructValue struct {
	Token  Token
	Name   *Label
	Fields map[string]Expression
}

func (s *StructValue) expressionNode()      {}
func (s *StructValue) TokenLiteral() string { return s.Token.Literal }

// PrefixExpression represents prefix operators like -5 or !true
type PrefixExpression struct {
	Token    Token
	Operator string
	Right    Expression
}

func (p *PrefixExpression) expressionNode()      {}
func (p *PrefixExpression) TokenLiteral() string { return p.Token.Literal }

// represents binary operations like 5 + 5
type InfixExpression struct {
	Token    Token
	Left     Expression
	Operator string
	Right    Expression
}

func (i *InfixExpression) expressionNode()      {}
func (i *InfixExpression) TokenLiteral() string { return i.Token.Literal }

// PostfixExpression represents i++ and i--
type PostfixExpression struct {
	Token    Token
	Operator string
	Left     Expression
}

func (s *PostfixExpression) expressionNode()      {}
func (s *PostfixExpression) TokenLiteral() string { return s.Token.Literal }

// CallExpression represents function calls like add(1, 2)
type CallExpression struct {
	Token     Token
	Function  Expression
	Arguments []Expression
}

func (c *CallExpression) expressionNode()      {}
func (c *CallExpression) TokenLiteral() string { return c.Token.Literal }

// IndexExpression represents array indexing like arr[0]
type IndexExpression struct {
	Token Token
	Left  Expression
	Index Expression
}

func (i *IndexExpression) expressionNode()      {}
func (i *IndexExpression) TokenLiteral() string { return i.Token.Literal }

// MemberExpression represents member access like std.println
type MemberExpression struct {
	Token  Token
	Object Expression
	Member *Label
}

func (m *MemberExpression) expressionNode()      {}
func (m *MemberExpression) TokenLiteral() string { return m.Token.Literal }

// NewExpression represents new(Type)
type NewExpression struct {
	Token    Token
	TypeName *Label
}

func (n *NewExpression) expressionNode()      {}
func (n *NewExpression) TokenLiteral() string { return n.Token.Literal }

// RangeExpression represents range(end), range(start, end), or range(start, end, step)
type RangeExpression struct {
	Token Token
	Start Expression // nil for range(end) form, defaults to 0
	End   Expression
	Step  Expression // nil for default step of 1
}

func (r *RangeExpression) expressionNode()      {}
func (r *RangeExpression) TokenLiteral() string { return r.Token.Literal }

// ============================================================================
// Statements
// ============================================================================

// VariableDeclaration represents temp x int = 5 or const y int = 10
// Also supports multiple assignment: temp result, err = divide(10, 0)
// And typed tuple unpacking: temp a int, b string = getValues()
type VariableDeclaration struct {
	Token      Token // temp or const
	Name       *Label
	Names      []*Label // for multiple assignment (result, err)
	TypeName   string
	TypeNames  []string // for typed tuple unpacking (int, string)
	Value      Expression
	Mutable    bool
	Attributes []*Attribute // @suppress(...) attributes
	Visibility Visibility   // Public (default), Private, or PrivateModule
}

// IgnoreValue represents @ignore in multiple assignment
type IgnoreValue struct {
	Token Token
}

func (i *IgnoreValue) expressionNode()      {}
func (i *IgnoreValue) TokenLiteral() string { return i.Token.Literal }

// Attribute represents @suppress(warning_name, ...)
type Attribute struct {
	Token Token
	Name  string   // "suppress"
	Args  []string // warning codes to suppress
}

func (a *Attribute) expressionNode()      {}
func (a *Attribute) TokenLiteral() string { return a.Token.Literal }

func (v *VariableDeclaration) statementNode()       {}
func (v *VariableDeclaration) TokenLiteral() string { return v.Token.Literal }

// AssignmentStatement represents x = 5 or x += 1
type AssignmentStatement struct {
	Token    Token
	Name     Expression // can be Identifier, IndexExpression, or MemberExpression
	Operator string
	Value    Expression
}

func (a *AssignmentStatement) statementNode()       {}
func (a *AssignmentStatement) TokenLiteral() string { return a.Token.Literal }

// ReturnStatement represents return value
type ReturnStatement struct {
	Token  Token
	Values []Expression // multiple return values
}

func (r *ReturnStatement) statementNode()       {}
func (r *ReturnStatement) TokenLiteral() string { return r.Token.Literal }

// ExpressionStatement wraps an expression as a statement
type ExpressionStatement struct {
	Token      Token
	Expression Expression
}

func (es *ExpressionStatement) statementNode()       {}
func (es *ExpressionStatement) TokenLiteral() string { return es.Token.Literal }

// BlockStatement represents a block of statements { ... }
type BlockStatement struct {
	Token      Token
	Statements []Statement
}

func (b *BlockStatement) statementNode()       {}
func (b *BlockStatement) TokenLiteral() string { return b.Token.Literal }

// IfStatement represents if/or/otherwise
type IfStatement struct {
	Token       Token
	Condition   Expression
	Consequence *BlockStatement
	Alternative Statement // can be another IfStatement (for 'or') or BlockStatement (for 'otherwise')
}

func (is *IfStatement) statementNode()       {}
func (is *IfStatement) TokenLiteral() string { return is.Token.Literal }

// ForStatement represents for i in range(0, 10) { }
type ForStatement struct {
	Token    Token
	Variable *Label
	VarType  string // optional explicit type
	Iterable Expression
	Body     *BlockStatement
}

func (fs *ForStatement) statementNode()       {}
func (fs *ForStatement) TokenLiteral() string { return fs.Token.Literal }

// ForEachStatement represents for_each item in collection { }
type ForEachStatement struct {
	Token      Token
	Variable   *Label
	Collection Expression
	Body       *BlockStatement
}

func (f *ForEachStatement) statementNode()       {}
func (f *ForEachStatement) TokenLiteral() string { return f.Token.Literal }

// WhileStatement represents as_long_as condition { }
type WhileStatement struct {
	Token     Token
	Condition Expression
	Body      *BlockStatement
}

func (w *WhileStatement) statementNode()       {}
func (w *WhileStatement) TokenLiteral() string { return w.Token.Literal }

// LoopStatement represents loop { }
type LoopStatement struct {
	Token Token
	Body  *BlockStatement
}

func (l *LoopStatement) statementNode()       {}
func (l *LoopStatement) TokenLiteral() string { return l.Token.Literal }

// BreakStatement represents break
type BreakStatement struct {
	Token Token
}

func (b *BreakStatement) statementNode()       {}
func (b *BreakStatement) TokenLiteral() string { return b.Token.Literal }

// ContinueStatement represents continue
type ContinueStatement struct {
	Token Token
}

func (c *ContinueStatement) statementNode()       {}
func (c *ContinueStatement) TokenLiteral() string { return c.Token.Literal }

// FunctionDeclaration represents do func_name(params) -> return_type { }
type FunctionDeclaration struct {
	Token       Token
	Name        *Label
	Parameters  []*Parameter
	ReturnTypes []string // can be multiple for multi-return
	Body        *BlockStatement
	Attributes  []*Attribute // @suppress(...) attributes
	Visibility  Visibility   // Public (default), Private, or PrivateModule
}

func (f *FunctionDeclaration) statementNode()       {}
func (f *FunctionDeclaration) TokenLiteral() string { return f.Token.Literal }

// Parameter represents a function parameter
type Parameter struct {
	Name     *Label
	TypeName string
}

// ImportItem represents a single module import with optional alias
type ImportItem struct {
	Alias    string // Optional alias (e.g., "arr" in "import arr@arrays")
	Module   string // Module name for stdlib (e.g., "std", "arrays")
	Path     string // File path for user modules (e.g., "./utils", "../server")
	IsStdlib bool   // True if this is a stdlib import (prefixed with @)
}

// ImportStatement represents import "module" or import alias "module"
// Supports both single (import @std) and comma-separated (import @std, @arrays)
// Also supports file path imports (import "./utils") and import & use syntax
type ImportStatement struct {
	Token   Token
	Imports []ImportItem // For multiple imports
	AutoUse bool         // True for "import & use" syntax - automatically brings into scope
	// Deprecated: For backward compatibility with single imports
	Alias  string
	Module string
}

// FromImportStatement represents "from @arrays import append, pop"
// or "from ./utils import helper, Config"
type FromImportStatement struct {
	Token    Token
	Path     string        // Module path (e.g., "@arrays" or "./utils")
	IsStdlib bool          // True if importing from stdlib
	Items    []*ImportSpec // Specific items to import
}

func (f *FromImportStatement) statementNode()       {}
func (f *FromImportStatement) TokenLiteral() string { return f.Token.Literal }

// ImportSpec represents a single item in a from-import statement
// e.g., "append" or "sqrt as square_root"
type ImportSpec struct {
	Name  string // Original name
	Alias string // Optional alias (for "as" syntax)
}

func (i *ImportStatement) statementNode()       {}
func (i *ImportStatement) TokenLiteral() string { return i.Token.Literal }

// UsingStatement represents using module(s)
// Supports both single (using std) and comma-separated (using std, arrays)
type UsingStatement struct {
	Token   Token
	Modules []*Label // List of modules to bring into scope
}

func (u *UsingStatement) statementNode()       {}
func (u *UsingStatement) TokenLiteral() string { return u.Token.Literal }

// StructDeclaration represents Person struct { name string; age int }
type StructDeclaration struct {
	Token      Token
	Name       *Label
	Fields     []*StructField
	Visibility Visibility // Public (default), Private, or PrivateModule
}

func (sd *StructDeclaration) statementNode()       {}
func (sd *StructDeclaration) TokenLiteral() string { return sd.Token.Literal }

// StructField represents a field in a struct
type StructField struct {
	Name       *Label
	TypeName   string
	Visibility Visibility // Public (default), Private, or PrivateModule
}

// EnumDeclaration represents const STATUS enum { ... }
type EnumDeclaration struct {
	Token      Token
	Name       *Label
	Values     []*EnumValue
	Attributes *EnumAttributes
	Visibility Visibility // Public (default), Private, or PrivateModule
}

func (ed *EnumDeclaration) statementNode()       {}
func (ed *EnumDeclaration) TokenLiteral() string { return ed.Token.Literal }

// EnumValue represents a single enum value
type EnumValue struct {
	Name  *Label
	Value Expression // optional explicit value
}

// EnumAttributes represents @(type, skip, increment)
type EnumAttributes struct {
	TypeName  string
	Skip      bool
	Increment Expression
}
