package ast

import (
	"math/big"
	"testing"

	"github.com/marshallburns/ez/pkg/tokenizer"
)

// Helper to create a token
func tok(t tokenizer.TokenType, lit string) tokenizer.Token {
	return tokenizer.Token{Type: t, Literal: lit, Line: 1, Column: 1}
}

// ============================================================================
// Interface Implementation Tests
// ============================================================================

// TestNodeInterface verifies all types implement Node interface
func TestNodeInterface(t *testing.T) {
	nodes := []Node{
		&Program{},
		&ModuleDeclaration{},
		&Label{},
		&IntegerValue{},
		&FloatValue{},
		&StringValue{},
		&InterpolatedString{},
		&CharValue{},
		&BooleanValue{},
		&NilValue{},
		&ArrayValue{},
		&MapValue{},
		&StructValue{},
		&PrefixExpression{},
		&InfixExpression{},
		&PostfixExpression{},
		&CallExpression{},
		&IndexExpression{},
		&MemberExpression{},
		&NewExpression{},
		&RangeExpression{},
		&BlankIdentifier{},
		&Attribute{},
		&VariableDeclaration{},
		&AssignmentStatement{},
		&ReturnStatement{},
		&ExpressionStatement{},
		&BlockStatement{},
		&IfStatement{},
		&ForStatement{},
		&ForEachStatement{},
		&WhileStatement{},
		&LoopStatement{},
		&BreakStatement{},
		&ContinueStatement{},
		&FunctionDeclaration{},
		&ImportStatement{},
		&FromImportStatement{},
		&UsingStatement{},
		&StructDeclaration{},
		&EnumDeclaration{},
	}

	for _, node := range nodes {
		// Just calling TokenLiteral() verifies the interface is implemented
		_ = node.TokenLiteral()
	}
}

// TestExpressionInterface verifies expression types implement Expression
func TestExpressionInterface(t *testing.T) {
	expressions := []Expression{
		&Label{},
		&IntegerValue{},
		&FloatValue{},
		&StringValue{},
		&InterpolatedString{},
		&CharValue{},
		&BooleanValue{},
		&NilValue{},
		&ArrayValue{},
		&MapValue{},
		&StructValue{},
		&PrefixExpression{},
		&InfixExpression{},
		&PostfixExpression{},
		&CallExpression{},
		&IndexExpression{},
		&MemberExpression{},
		&NewExpression{},
		&RangeExpression{},
		&BlankIdentifier{},
		&Attribute{},
	}

	for _, expr := range expressions {
		expr.expressionNode()
		_ = expr.TokenLiteral()
	}
}

// TestStatementInterface verifies statement types implement Statement
func TestStatementInterface(t *testing.T) {
	statements := []Statement{
		&ModuleDeclaration{},
		&VariableDeclaration{},
		&AssignmentStatement{},
		&ReturnStatement{},
		&ExpressionStatement{},
		&BlockStatement{},
		&IfStatement{},
		&ForStatement{},
		&ForEachStatement{},
		&WhileStatement{},
		&LoopStatement{},
		&BreakStatement{},
		&ContinueStatement{},
		&FunctionDeclaration{},
		&ImportStatement{},
		&FromImportStatement{},
		&UsingStatement{},
		&StructDeclaration{},
		&EnumDeclaration{},
	}

	for _, stmt := range statements {
		stmt.statementNode()
		_ = stmt.TokenLiteral()
	}
}

// ============================================================================
// TokenLiteral Tests
// ============================================================================

func TestProgramTokenLiteral(t *testing.T) {
	t.Run("empty program", func(t *testing.T) {
		p := &Program{}
		if p.TokenLiteral() != "" {
			t.Errorf("TokenLiteral() = %q, want empty string", p.TokenLiteral())
		}
	})

	t.Run("program with statements", func(t *testing.T) {
		p := &Program{
			Statements: []Statement{
				&VariableDeclaration{Token: tok(tokenizer.TEMP, "temp")},
			},
		}
		if p.TokenLiteral() != "temp" {
			t.Errorf("TokenLiteral() = %q, want %q", p.TokenLiteral(), "temp")
		}
	})
}

func TestLabelTokenLiteral(t *testing.T) {
	l := &Label{Token: tok(tokenizer.IDENT, "myVar"), Value: "myVar"}
	if l.TokenLiteral() != "myVar" {
		t.Errorf("TokenLiteral() = %q, want %q", l.TokenLiteral(), "myVar")
	}
}

func TestIntegerValueTokenLiteral(t *testing.T) {
	v := &IntegerValue{Token: tok(tokenizer.INT, "42"), Value: big.NewInt(42)}
	if v.TokenLiteral() != "42" {
		t.Errorf("TokenLiteral() = %q, want %q", v.TokenLiteral(), "42")
	}
}

func TestFloatValueTokenLiteral(t *testing.T) {
	v := &FloatValue{Token: tok(tokenizer.FLOAT, "3.14"), Value: 3.14}
	if v.TokenLiteral() != "3.14" {
		t.Errorf("TokenLiteral() = %q, want %q", v.TokenLiteral(), "3.14")
	}
}

func TestStringValueTokenLiteral(t *testing.T) {
	v := &StringValue{Token: tok(tokenizer.STRING, "hello"), Value: "hello"}
	if v.TokenLiteral() != "hello" {
		t.Errorf("TokenLiteral() = %q, want %q", v.TokenLiteral(), "hello")
	}
}

func TestCharValueTokenLiteral(t *testing.T) {
	v := &CharValue{Token: tok(tokenizer.CHAR, "a"), Value: 'a'}
	if v.TokenLiteral() != "a" {
		t.Errorf("TokenLiteral() = %q, want %q", v.TokenLiteral(), "a")
	}
}

func TestBooleanValueTokenLiteral(t *testing.T) {
	v := &BooleanValue{Token: tok(tokenizer.TRUE, "true"), Value: true}
	if v.TokenLiteral() != "true" {
		t.Errorf("TokenLiteral() = %q, want %q", v.TokenLiteral(), "true")
	}
}

func TestNilValueTokenLiteral(t *testing.T) {
	v := &NilValue{Token: tok(tokenizer.NIL, "nil")}
	if v.TokenLiteral() != "nil" {
		t.Errorf("TokenLiteral() = %q, want %q", v.TokenLiteral(), "nil")
	}
}

func TestArrayValueTokenLiteral(t *testing.T) {
	v := &ArrayValue{Token: tok(tokenizer.LBRACE, "{")}
	if v.TokenLiteral() != "{" {
		t.Errorf("TokenLiteral() = %q, want %q", v.TokenLiteral(), "{")
	}
}

func TestMapValueTokenLiteral(t *testing.T) {
	v := &MapValue{Token: tok(tokenizer.LBRACE, "{")}
	if v.TokenLiteral() != "{" {
		t.Errorf("TokenLiteral() = %q, want %q", v.TokenLiteral(), "{")
	}
}

func TestPrefixExpressionTokenLiteral(t *testing.T) {
	p := &PrefixExpression{Token: tok(tokenizer.MINUS, "-"), Operator: "-"}
	if p.TokenLiteral() != "-" {
		t.Errorf("TokenLiteral() = %q, want %q", p.TokenLiteral(), "-")
	}
}

func TestInfixExpressionTokenLiteral(t *testing.T) {
	i := &InfixExpression{Token: tok(tokenizer.PLUS, "+"), Operator: "+"}
	if i.TokenLiteral() != "+" {
		t.Errorf("TokenLiteral() = %q, want %q", i.TokenLiteral(), "+")
	}
}

func TestPostfixExpressionTokenLiteral(t *testing.T) {
	p := &PostfixExpression{Token: tok(tokenizer.INCREMENT, "++"), Operator: "++"}
	if p.TokenLiteral() != "++" {
		t.Errorf("TokenLiteral() = %q, want %q", p.TokenLiteral(), "++")
	}
}

func TestCallExpressionTokenLiteral(t *testing.T) {
	c := &CallExpression{Token: tok(tokenizer.LPAREN, "(")}
	if c.TokenLiteral() != "(" {
		t.Errorf("TokenLiteral() = %q, want %q", c.TokenLiteral(), "(")
	}
}

func TestIndexExpressionTokenLiteral(t *testing.T) {
	i := &IndexExpression{Token: tok(tokenizer.LBRACKET, "[")}
	if i.TokenLiteral() != "[" {
		t.Errorf("TokenLiteral() = %q, want %q", i.TokenLiteral(), "[")
	}
}

func TestMemberExpressionTokenLiteral(t *testing.T) {
	m := &MemberExpression{Token: tok(tokenizer.DOT, ".")}
	if m.TokenLiteral() != "." {
		t.Errorf("TokenLiteral() = %q, want %q", m.TokenLiteral(), ".")
	}
}

func TestVariableDeclarationTokenLiteral(t *testing.T) {
	v := &VariableDeclaration{Token: tok(tokenizer.TEMP, "temp")}
	if v.TokenLiteral() != "temp" {
		t.Errorf("TokenLiteral() = %q, want %q", v.TokenLiteral(), "temp")
	}
}

func TestReturnStatementTokenLiteral(t *testing.T) {
	r := &ReturnStatement{Token: tok(tokenizer.RETURN, "return")}
	if r.TokenLiteral() != "return" {
		t.Errorf("TokenLiteral() = %q, want %q", r.TokenLiteral(), "return")
	}
}

func TestIfStatementTokenLiteral(t *testing.T) {
	i := &IfStatement{Token: tok(tokenizer.IF, "if")}
	if i.TokenLiteral() != "if" {
		t.Errorf("TokenLiteral() = %q, want %q", i.TokenLiteral(), "if")
	}
}

func TestForStatementTokenLiteral(t *testing.T) {
	f := &ForStatement{Token: tok(tokenizer.FOR, "for")}
	if f.TokenLiteral() != "for" {
		t.Errorf("TokenLiteral() = %q, want %q", f.TokenLiteral(), "for")
	}
}

func TestForEachStatementTokenLiteral(t *testing.T) {
	f := &ForEachStatement{Token: tok(tokenizer.FOR_EACH, "for_each")}
	if f.TokenLiteral() != "for_each" {
		t.Errorf("TokenLiteral() = %q, want %q", f.TokenLiteral(), "for_each")
	}
}

func TestWhileStatementTokenLiteral(t *testing.T) {
	w := &WhileStatement{Token: tok(tokenizer.AS_LONG_AS, "as_long_as")}
	if w.TokenLiteral() != "as_long_as" {
		t.Errorf("TokenLiteral() = %q, want %q", w.TokenLiteral(), "as_long_as")
	}
}

func TestLoopStatementTokenLiteral(t *testing.T) {
	l := &LoopStatement{Token: tok(tokenizer.LOOP, "loop")}
	if l.TokenLiteral() != "loop" {
		t.Errorf("TokenLiteral() = %q, want %q", l.TokenLiteral(), "loop")
	}
}

func TestBreakStatementTokenLiteral(t *testing.T) {
	b := &BreakStatement{Token: tok(tokenizer.BREAK, "break")}
	if b.TokenLiteral() != "break" {
		t.Errorf("TokenLiteral() = %q, want %q", b.TokenLiteral(), "break")
	}
}

func TestContinueStatementTokenLiteral(t *testing.T) {
	c := &ContinueStatement{Token: tok(tokenizer.CONTINUE, "continue")}
	if c.TokenLiteral() != "continue" {
		t.Errorf("TokenLiteral() = %q, want %q", c.TokenLiteral(), "continue")
	}
}

func TestFunctionDeclarationTokenLiteral(t *testing.T) {
	f := &FunctionDeclaration{Token: tok(tokenizer.DO, "do")}
	if f.TokenLiteral() != "do" {
		t.Errorf("TokenLiteral() = %q, want %q", f.TokenLiteral(), "do")
	}
}

func TestImportStatementTokenLiteral(t *testing.T) {
	i := &ImportStatement{Token: tok(tokenizer.IMPORT, "import")}
	if i.TokenLiteral() != "import" {
		t.Errorf("TokenLiteral() = %q, want %q", i.TokenLiteral(), "import")
	}
}

func TestUsingStatementTokenLiteral(t *testing.T) {
	u := &UsingStatement{Token: tok(tokenizer.USING, "using")}
	if u.TokenLiteral() != "using" {
		t.Errorf("TokenLiteral() = %q, want %q", u.TokenLiteral(), "using")
	}
}

func TestStructDeclarationTokenLiteral(t *testing.T) {
	s := &StructDeclaration{Token: tok(tokenizer.STRUCT, "struct")}
	if s.TokenLiteral() != "struct" {
		t.Errorf("TokenLiteral() = %q, want %q", s.TokenLiteral(), "struct")
	}
}

func TestEnumDeclarationTokenLiteral(t *testing.T) {
	e := &EnumDeclaration{Token: tok(tokenizer.ENUM, "enum")}
	if e.TokenLiteral() != "enum" {
		t.Errorf("TokenLiteral() = %q, want %q", e.TokenLiteral(), "enum")
	}
}

// ============================================================================
// Visibility Tests
// ============================================================================

func TestVisibilityConstants(t *testing.T) {
	if VisibilityPublic != 0 {
		t.Errorf("VisibilityPublic = %d, want 0", VisibilityPublic)
	}
	if VisibilityPrivate != 1 {
		t.Errorf("VisibilityPrivate = %d, want 1", VisibilityPrivate)
	}
	if VisibilityPrivateModule != 2 {
		t.Errorf("VisibilityPrivateModule = %d, want 2", VisibilityPrivateModule)
	}
}

// ============================================================================
// Structure Tests
// ============================================================================

func TestProgramStructure(t *testing.T) {
	p := &Program{
		Module: &ModuleDeclaration{
			Token: tok(tokenizer.MODULE, "module"),
			Name:  &Label{Value: "mymodule"},
		},
		FileUsing: []*UsingStatement{
			{Token: tok(tokenizer.USING, "using")},
		},
		Statements: []Statement{
			&VariableDeclaration{Token: tok(tokenizer.TEMP, "temp")},
		},
	}

	if p.Module == nil {
		t.Error("Module should not be nil")
	}
	if len(p.FileUsing) != 1 {
		t.Errorf("FileUsing length = %d, want 1", len(p.FileUsing))
	}
	if len(p.Statements) != 1 {
		t.Errorf("Statements length = %d, want 1", len(p.Statements))
	}
}

func TestVariableDeclarationStructure(t *testing.T) {
	v := &VariableDeclaration{
		Token:      tok(tokenizer.TEMP, "temp"),
		Name:       &Label{Value: "x"},
		TypeName:   "int",
		Value:      &IntegerValue{Value: big.NewInt(42)},
		Mutable:    true,
		Visibility: VisibilityPublic,
	}

	if v.Name.Value != "x" {
		t.Errorf("Name = %q, want %q", v.Name.Value, "x")
	}
	if v.TypeName != "int" {
		t.Errorf("TypeName = %q, want %q", v.TypeName, "int")
	}
	if !v.Mutable {
		t.Error("Mutable should be true")
	}
}

func TestFunctionDeclarationStructure(t *testing.T) {
	f := &FunctionDeclaration{
		Token: tok(tokenizer.DO, "do"),
		Name:  &Label{Value: "add"},
		Parameters: []*Parameter{
			{Name: &Label{Value: "a"}, TypeName: "int"},
			{Name: &Label{Value: "b"}, TypeName: "int"},
		},
		ReturnTypes: []string{"int"},
		Body:        &BlockStatement{},
	}

	if f.Name.Value != "add" {
		t.Errorf("Name = %q, want %q", f.Name.Value, "add")
	}
	if len(f.Parameters) != 2 {
		t.Errorf("Parameters length = %d, want 2", len(f.Parameters))
	}
	if len(f.ReturnTypes) != 1 {
		t.Errorf("ReturnTypes length = %d, want 1", len(f.ReturnTypes))
	}
}

func TestStructDeclarationStructure(t *testing.T) {
	s := &StructDeclaration{
		Token: tok(tokenizer.STRUCT, "struct"),
		Name:  &Label{Value: "Person"},
		Fields: []*StructField{
			{Name: &Label{Value: "name"}, TypeName: "string"},
			{Name: &Label{Value: "age"}, TypeName: "int"},
		},
	}

	if s.Name.Value != "Person" {
		t.Errorf("Name = %q, want %q", s.Name.Value, "Person")
	}
	if len(s.Fields) != 2 {
		t.Errorf("Fields length = %d, want 2", len(s.Fields))
	}
}

func TestEnumDeclarationStructure(t *testing.T) {
	e := &EnumDeclaration{
		Token: tok(tokenizer.ENUM, "enum"),
		Name:  &Label{Value: "Status"},
		Values: []*EnumValue{
			{Name: &Label{Value: "TODO"}},
			{Name: &Label{Value: "DONE"}},
		},
		Attributes: &EnumAttributes{
			TypeName: "int",
			Skip:     true,
		},
	}

	if e.Name.Value != "Status" {
		t.Errorf("Name = %q, want %q", e.Name.Value, "Status")
	}
	if len(e.Values) != 2 {
		t.Errorf("Values length = %d, want 2", len(e.Values))
	}
	if e.Attributes.TypeName != "int" {
		t.Errorf("Attributes.TypeName = %q, want %q", e.Attributes.TypeName, "int")
	}
}

func TestIfStatementStructure(t *testing.T) {
	i := &IfStatement{
		Token:     tok(tokenizer.IF, "if"),
		Condition: &BooleanValue{Value: true},
		Consequence: &BlockStatement{
			Statements: []Statement{},
		},
		Alternative: &BlockStatement{
			Statements: []Statement{},
		},
	}

	if i.Condition == nil {
		t.Error("Condition should not be nil")
	}
	if i.Consequence == nil {
		t.Error("Consequence should not be nil")
	}
	if i.Alternative == nil {
		t.Error("Alternative should not be nil")
	}
}

func TestImportStatementStructure(t *testing.T) {
	i := &ImportStatement{
		Token: tok(tokenizer.IMPORT, "import"),
		Imports: []ImportItem{
			{Module: "std", IsStdlib: true},
			{Module: "arrays", Alias: "arr", IsStdlib: true},
		},
		AutoUse: true,
	}

	if len(i.Imports) != 2 {
		t.Errorf("Imports length = %d, want 2", len(i.Imports))
	}
	if i.Imports[0].Module != "std" {
		t.Errorf("Imports[0].Module = %q, want %q", i.Imports[0].Module, "std")
	}
	if i.Imports[1].Alias != "arr" {
		t.Errorf("Imports[1].Alias = %q, want %q", i.Imports[1].Alias, "arr")
	}
}

func TestInfixExpressionStructure(t *testing.T) {
	i := &InfixExpression{
		Token:    tok(tokenizer.PLUS, "+"),
		Left:     &IntegerValue{Value: big.NewInt(1)},
		Operator: "+",
		Right:    &IntegerValue{Value: big.NewInt(2)},
	}

	if i.Operator != "+" {
		t.Errorf("Operator = %q, want %q", i.Operator, "+")
	}
	if i.Left == nil {
		t.Error("Left should not be nil")
	}
	if i.Right == nil {
		t.Error("Right should not be nil")
	}
}

func TestCallExpressionStructure(t *testing.T) {
	c := &CallExpression{
		Token:    tok(tokenizer.LPAREN, "("),
		Function: &Label{Value: "println"},
		Arguments: []Expression{
			&StringValue{Value: "hello"},
		},
	}

	if len(c.Arguments) != 1 {
		t.Errorf("Arguments length = %d, want 1", len(c.Arguments))
	}
}

func TestMapPairStructure(t *testing.T) {
	m := &MapValue{
		Token: tok(tokenizer.LBRACE, "{"),
		Pairs: []*MapPair{
			{
				Key:   &StringValue{Value: "name"},
				Value: &StringValue{Value: "Alice"},
			},
		},
	}

	if len(m.Pairs) != 1 {
		t.Errorf("Pairs length = %d, want 1", len(m.Pairs))
	}
}

func TestRangeExpressionStructure(t *testing.T) {
	r := &RangeExpression{
		Token: tok(tokenizer.RANGE, "range"),
		Start: &IntegerValue{Value: big.NewInt(0)},
		End:   &IntegerValue{Value: big.NewInt(10)},
		Step:  &IntegerValue{Value: big.NewInt(2)},
	}

	if r.Start == nil {
		t.Error("Start should not be nil")
	}
	if r.End == nil {
		t.Error("End should not be nil")
	}
	if r.Step == nil {
		t.Error("Step should not be nil")
	}
}
