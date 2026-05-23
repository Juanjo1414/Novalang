// =============================================================================
// ast/ast.go — Árbol Sintáctico Abstracto (AST) de NovaLang
//
// El Parser convierte la secuencia de tokens en este árbol.
// El Evaluador recorre el árbol para ejecutar el programa.
//
// Jerarquía:
//   Node
//   ├── Statement (sentencias: no producen valor directamente)
//   │   ├── Program              ← nodo raíz
//   │   ├── LetStatement         ← let x = expr;
//   │   ├── ReturnStatement      ← return expr;
//   │   ├── ExpressionStatement  ← expr; (expresión como sentencia)
//   │   ├── BlockStatement       ← { stmt; stmt; }
//   │   ├── PrintStatement       ← print(expr);
//   │   ├── WhileStatement       ← while (cond) { }
//   │   ├── ForStatement         ← for (init; cond; upd) { }
//   │   ├── BreakStatement       ← break;
//   │   └── ContinueStatement    ← continue;
//   │
//   └── Expression (expresiones: producen un valor)
//       ├── Identifier           ← x, miVar
//       ├── IntegerLiteral       ← 42
//       ├── FloatLiteral         ← 3.14
//       ├── StringLiteral        ← "hola"
//       ├── BooleanLiteral       ← true / false
//       ├── NilLiteral           ← nil
//       ├── PrefixExpression     ← !expr, -expr
//       ├── InfixExpression      ← expr OP expr
//       ├── IfExpression         ← if (c) { } elseif (c) { } else { }
//       ├── FunctionLiteral      ← function(params) { body }
//       └── CallExpression       ← func(args)
// =============================================================================

package ast

import (
	"fmt"
	"strings"

	"github.com/novalang/novalang/internal/token"
)

// ─────────────────────────────────────────────────────────────────────────────
// INTERFACES BASE
// ─────────────────────────────────────────────────────────────────────────────

// Node es la interfaz que implementan todos los nodos del AST.
type Node interface {
	TokenLiteral() string // Literal del token principal del nodo (para debug)
	String() string       // Representación legible del nodo
}

// Statement es un nodo que representa una SENTENCIA (no produce valor).
type Statement interface {
	Node
	statementNode() // marca de tipo (evita que Expression sea usada como Statement)
}

// Expression es un nodo que representa una EXPRESIÓN (produce un valor).
type Expression interface {
	Node
	expressionNode() // marca de tipo
}

// ─────────────────────────────────────────────────────────────────────────────
// NODO RAÍZ
// ─────────────────────────────────────────────────────────────────────────────

// Program es el nodo raíz del AST. Contiene todas las sentencias del programa.
type Program struct {
	Statements []Statement
}

func (p *Program) TokenLiteral() string {
	if len(p.Statements) > 0 {
		return p.Statements[0].TokenLiteral()
	}
	return ""
}

func (p *Program) String() string {
	var sb strings.Builder
	for _, s := range p.Statements {
		sb.WriteString(s.String())
		sb.WriteRune('\n')
	}
	return sb.String()
}

// ─────────────────────────────────────────────────────────────────────────────
// SENTENCIAS
// ─────────────────────────────────────────────────────────────────────────────

// LetStatement representa: let <nombre> = <expresión>;
type LetStatement struct {
	Token token.Token // El token LET
	Name  *Identifier // El nombre de la variable
	Value Expression  // La expresión del lado derecho
}

func (ls *LetStatement) statementNode()       {}
func (ls *LetStatement) TokenLiteral() string { return ls.Token.Literal }
func (ls *LetStatement) String() string {
	val := ""
	if ls.Value != nil {
		val = ls.Value.String()
	}
	return fmt.Sprintf("let %s = %s;", ls.Name, val)
}

// ReturnStatement representa: return <expresión>;
type ReturnStatement struct {
	Token       token.Token // El token RETURN
	ReturnValue Expression  // La expresión a retornar
}

func (rs *ReturnStatement) statementNode()       {}
func (rs *ReturnStatement) TokenLiteral() string { return rs.Token.Literal }
func (rs *ReturnStatement) String() string {
	val := ""
	if rs.ReturnValue != nil {
		val = rs.ReturnValue.String()
	}
	return fmt.Sprintf("return %s;", val)
}

// ExpressionStatement representa una expresión usada como sentencia: expr;
type ExpressionStatement struct {
	Token      token.Token // El primer token de la expresión
	Expression Expression  // La expresión
}

func (es *ExpressionStatement) statementNode()       {}
func (es *ExpressionStatement) TokenLiteral() string { return es.Token.Literal }
func (es *ExpressionStatement) String() string {
	if es.Expression != nil {
		return es.Expression.String()
	}
	return ""
}

// BlockStatement representa un bloque: { stmt; stmt; }
type BlockStatement struct {
	Token      token.Token // El token '{'
	Statements []Statement
}

func (bs *BlockStatement) statementNode()       {}
func (bs *BlockStatement) TokenLiteral() string { return bs.Token.Literal }
func (bs *BlockStatement) String() string {
	var sb strings.Builder
	sb.WriteString("{\n")
	for _, s := range bs.Statements {
		sb.WriteString("  ")
		sb.WriteString(s.String())
		sb.WriteRune('\n')
	}
	sb.WriteString("}")
	return sb.String()
}

// PrintStatement representa: print(<expresión>);
type PrintStatement struct {
	Token token.Token // El token PRINT
	Value Expression  // La expresión a imprimir
}

func (ps *PrintStatement) statementNode()       {}
func (ps *PrintStatement) TokenLiteral() string { return ps.Token.Literal }
func (ps *PrintStatement) String() string {
	return fmt.Sprintf("print(%s);", ps.Value)
}

// WhileStatement representa: while (<condición>) { <cuerpo> }
type WhileStatement struct {
	Token     token.Token    // El token WHILE
	Condition Expression     // La condición del bucle
	Body      *BlockStatement // El cuerpo
}

func (ws *WhileStatement) statementNode()       {}
func (ws *WhileStatement) TokenLiteral() string { return ws.Token.Literal }
func (ws *WhileStatement) String() string {
	return fmt.Sprintf("while (%s) %s", ws.Condition, ws.Body)
}

// ElseIfClause representa una rama elseif (condición, bloque).
type ElseIfClause struct {
	Condition Expression
	Body      *BlockStatement
}

// IfExpression representa: if (cond) { } [elseif (c) { }]* [else { }]
// Se modela como Expression porque en el lenguaje del profesor puede usarse
// en posición de expresión. En NovaLang lo tratamos igual para compatibilidad.
type IfExpression struct {
	Token       token.Token    // El token IF
	Condition   Expression     // Condición principal
	Consequence *BlockStatement // Bloque if
	Alternatives []ElseIfClause // Ramas elseif
	ElseBlock   *BlockStatement // Bloque else (opcional)
}

func (ie *IfExpression) expressionNode()      {}
func (ie *IfExpression) TokenLiteral() string { return ie.Token.Literal }
func (ie *IfExpression) String() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "if (%s) %s", ie.Condition, ie.Consequence)
	for _, alt := range ie.Alternatives {
		fmt.Fprintf(&sb, " elseif (%s) %s", alt.Condition, alt.Body)
	}
	if ie.ElseBlock != nil {
		fmt.Fprintf(&sb, " else %s", ie.ElseBlock)
	}
	return sb.String()
}

// ForStatement representa: for (<init>; <cond>; <upd>) { <body> }
type ForStatement struct {
	Token     token.Token    // El token FOR
	Init      Statement      // Inicialización (puede ser nil)
	Condition Expression     // Condición (puede ser nil → infinito)
	Update    Statement      // Actualización (puede ser nil)
	Body      *BlockStatement
}

func (fs *ForStatement) statementNode()       {}
func (fs *ForStatement) TokenLiteral() string { return fs.Token.Literal }
func (fs *ForStatement) String() string {
	initStr := ""
	if fs.Init != nil {
		initStr = fs.Init.String()
	}
	condStr := ""
	if fs.Condition != nil {
		condStr = fs.Condition.String()
	}
	updStr := ""
	if fs.Update != nil {
		updStr = fs.Update.String()
	}
	return fmt.Sprintf("for (%s; %s; %s) %s", initStr, condStr, updStr, fs.Body)
}

// BreakStatement representa: break;
type BreakStatement struct {
	Token token.Token
}

func (bs *BreakStatement) statementNode()       {}
func (bs *BreakStatement) TokenLiteral() string { return bs.Token.Literal }
func (bs *BreakStatement) String() string       { return "break;" }

// ContinueStatement representa: continue;
type ContinueStatement struct {
	Token token.Token
}

func (cs *ContinueStatement) statementNode()       {}
func (cs *ContinueStatement) TokenLiteral() string { return cs.Token.Literal }
func (cs *ContinueStatement) String() string       { return "continue;" }

// ─────────────────────────────────────────────────────────────────────────────
// EXPRESIONES
// ─────────────────────────────────────────────────────────────────────────────

// Identifier representa un nombre de variable o función.
type Identifier struct {
	Token token.Token // El token IDENTIFIER
	Value string      // El nombre
}

func (i *Identifier) expressionNode()      {}
func (i *Identifier) TokenLiteral() string { return i.Token.Literal }
func (i *Identifier) String() string       { return i.Value }

// IntegerLiteral representa un número entero: 42
type IntegerLiteral struct {
	Token token.Token
	Value int64
}

func (il *IntegerLiteral) expressionNode()      {}
func (il *IntegerLiteral) TokenLiteral() string { return il.Token.Literal }
func (il *IntegerLiteral) String() string       { return il.Token.Literal }

// FloatLiteral representa un número flotante: 3.14
type FloatLiteral struct {
	Token token.Token
	Value float64
}

func (fl *FloatLiteral) expressionNode()      {}
func (fl *FloatLiteral) TokenLiteral() string { return fl.Token.Literal }
func (fl *FloatLiteral) String() string       { return fl.Token.Literal }

// StringLiteral representa una cadena de texto: "hola"
type StringLiteral struct {
	Token token.Token
	Value string
}

func (sl *StringLiteral) expressionNode()      {}
func (sl *StringLiteral) TokenLiteral() string { return sl.Token.Literal }
func (sl *StringLiteral) String() string       { return fmt.Sprintf("%q", sl.Value) }

// BooleanLiteral representa true o false.
type BooleanLiteral struct {
	Token token.Token
	Value bool
}

func (bl *BooleanLiteral) expressionNode()      {}
func (bl *BooleanLiteral) TokenLiteral() string { return bl.Token.Literal }
func (bl *BooleanLiteral) String() string {
	if bl.Value {
		return "true"
	}
	return "false"
}

// NilLiteral representa nil.
type NilLiteral struct {
	Token token.Token
}

func (nl *NilLiteral) expressionNode()      {}
func (nl *NilLiteral) TokenLiteral() string { return nl.Token.Literal }
func (nl *NilLiteral) String() string       { return "nil" }

// PrefixExpression representa: <operador><expresión>  →  !x, -5
type PrefixExpression struct {
	Token    token.Token // El token del operador
	Operator string      // El símbolo del operador
	Right    Expression  // La expresión de la derecha
}

func (pe *PrefixExpression) expressionNode()      {}
func (pe *PrefixExpression) TokenLiteral() string { return pe.Token.Literal }
func (pe *PrefixExpression) String() string {
	return fmt.Sprintf("(%s%s)", pe.Operator, pe.Right)
}

// InfixExpression representa: <izq> <operador> <der>  →  5 + 3, x == y
type InfixExpression struct {
	Token    token.Token // El token del operador
	Left     Expression
	Operator string
	Right    Expression
}

func (ie *InfixExpression) expressionNode()      {}
func (ie *InfixExpression) TokenLiteral() string { return ie.Token.Literal }
func (ie *InfixExpression) String() string {
	return fmt.Sprintf("(%s %s %s)", ie.Left, ie.Operator, ie.Right)
}

// FunctionLiteral representa: function(<params>) { <body> }
type FunctionLiteral struct {
	Token      token.Token    // El token FUNCTION
	Parameters []*Identifier  // Parámetros formales
	Body       *BlockStatement
	Name       string // Nombre si fue asignada con let (para recursión)
}

func (fl *FunctionLiteral) expressionNode()      {}
func (fl *FunctionLiteral) TokenLiteral() string { return fl.Token.Literal }
func (fl *FunctionLiteral) String() string {
	params := make([]string, len(fl.Parameters))
	for i, p := range fl.Parameters {
		params[i] = p.String()
	}
	nameStr := ""
	if fl.Name != "" {
		nameStr = " " + fl.Name
	}
	return fmt.Sprintf("function%s(%s) %s", nameStr, strings.Join(params, ", "), fl.Body)
}

// CallExpression representa: <función>(<argumentos>)
type CallExpression struct {
	Token     token.Token  // El token '('
	Function  Expression   // La función (Identifier o FunctionLiteral)
	Arguments []Expression // Los argumentos reales
}

func (ce *CallExpression) expressionNode()      {}
func (ce *CallExpression) TokenLiteral() string { return ce.Token.Literal }
func (ce *CallExpression) String() string {
	args := make([]string, len(ce.Arguments))
	for i, a := range ce.Arguments {
		args[i] = a.String()
	}
	return fmt.Sprintf("%s(%s)", ce.Function, strings.Join(args, ", "))
}
