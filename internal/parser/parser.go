// =============================================================================
// parser/parser.go — Analizador Sintáctico (Pratt Parser) de NovaLang
//
// Toma la secuencia de tokens del Lexer y construye el AST.
//
// Se implementa un PRATT PARSER (Top-Down Operator Precedence):
//   - Cada token tiene asociada una función "nud" (prefija) que sabe
//     cómo parsearlo cuando aparece al INICIO de una expresión.
//   - Cada token infijo tiene una función "led" que recibe la expresión
//     izquierda ya parseada.
//   - La precedencia controla qué tan fuerte un operador "atrae" sus operandos.
//
// Tabla de precedencias (menor → mayor):
//   LOWEST    = 1  → base
//   OR        = 2  → or
//   AND       = 3  → and
//   EQUALS    = 4  → ==, !=
//   LESSGREAT = 5  → <, >, <=, >=
//   SUM       = 6  → +, -
//   PRODUCT   = 7  → *, /, %
//   PREFIX    = 8  → -x, !x
//   POWER     = 9  → ^
//   CALL      = 10 → f(args)
// =============================================================================

package parser

import (
	"fmt"
	"strconv"

	"github.com/novalang/novalang/internal/ast"
	"github.com/novalang/novalang/internal/lexer"
	"github.com/novalang/novalang/internal/token"
)

// ─────────────────────────────────────────────────────────────────────────────
// PRECEDENCIAS
// ─────────────────────────────────────────────────────────────────────────────

type Precedence int

const (
	LOWEST    Precedence = iota + 1 // 1 — base
	OR                              // 2 — or
	AND                             // 3 — and
	EQUALS                          // 4 — == !=
	LESSGREAT                       // 5 — < > <= >=
	SUM                             // 6 — + -
	PRODUCT                         // 7 — * / %
	PREFIX                          // 8 — !x -x
	POWER                           // 9 — ^
	CALL                            // 10 — f(args)
)

// precedences mapea cada TokenType a su precedencia como operador infijo.
var precedences = map[token.TokenType]Precedence{
	token.OR:       OR,
	token.AND:      AND,
	token.EQ:       EQUALS,
	token.NEQ:      EQUALS,
	token.LT:       LESSGREAT,
	token.LTE:      LESSGREAT,
	token.GT:       LESSGREAT,
	token.GTE:      LESSGREAT,
	token.PLUS:     SUM,
	token.MINUS:    SUM,
	token.MULTIPLY: PRODUCT,
	token.DIVISION: PRODUCT,
	token.MOD:      PRODUCT,
	token.POW:      POWER,
	token.LPAREN:   CALL,
}

// Tipos de funciones de parseo
type (
	prefixParseFn func() ast.Expression
	infixParseFn  func(ast.Expression) ast.Expression
)

// ─────────────────────────────────────────────────────────────────────────────
// PARSER
// ─────────────────────────────────────────────────────────────────────────────

// Parser construye el AST a partir de los tokens del Lexer.
type Parser struct {
	l *lexer.Lexer

	currentToken token.Token // token bajo análisis
	peekToken    token.Token // siguiente token (look-ahead de 1)

	errors []string // errores de parseo encontrados

	prefixParseFns map[token.TokenType]prefixParseFn
	infixParseFns  map[token.TokenType]infixParseFn
}

// New crea un Parser inicializado para el lexer dado.
func New(l *lexer.Lexer) *Parser {
	p := &Parser{l: l}

	// ── Funciones de parseo PREFIJAS ─────────────────────────────────────────
	// Se llaman cuando el token aparece al INICIO de una expresión.
	p.prefixParseFns = map[token.TokenType]prefixParseFn{
		token.IDENTIFIER: p.parseIdentifier,
		token.INTEGER:    p.parseIntegerLiteral,
		token.FLOAT:      p.parseFloatLiteral,
		token.STRING:     p.parseStringLiteral,
		token.TRUE:       p.parseBooleanLiteral,
		token.FALSE:      p.parseBooleanLiteral,
		token.NIL:        p.parseNilLiteral,
		token.NEGATION:   p.parsePrefixExpression,
		token.MINUS:      p.parsePrefixExpression,
		token.LPAREN:     p.parseGroupedExpression,
		token.IF:         p.parseIfExpression,
		token.FUNCTION:   p.parseFunctionLiteral,
	}

	// ── Funciones de parseo INFIJAS ───────────────────────────────────────────
	// Se llaman cuando el token aparece como operador entre dos expresiones.
	p.infixParseFns = map[token.TokenType]infixParseFn{
		token.PLUS:     p.parseInfixExpression,
		token.MINUS:    p.parseInfixExpression,
		token.MULTIPLY: p.parseInfixExpression,
		token.DIVISION: p.parseInfixExpression,
		token.MOD:      p.parseInfixExpression,
		token.POW:      p.parseInfixExpression,
		token.EQ:       p.parseInfixExpression,
		token.NEQ:      p.parseInfixExpression,
		token.LT:       p.parseInfixExpression,
		token.LTE:      p.parseInfixExpression,
		token.GT:       p.parseInfixExpression,
		token.GTE:      p.parseInfixExpression,
		token.AND:      p.parseInfixExpression,
		token.OR:       p.parseInfixExpression,
		token.LPAREN:   p.parseCallExpression,
	}

	// Carga los dos primeros tokens para que current y peek estén listos
	p.advance()
	p.advance()

	return p
}

// Errors retorna la lista de errores de parseo.
func (p *Parser) Errors() []string { return p.errors }

// HasErrors reporta si hubo algún error de parseo.
func (p *Parser) HasErrors() bool { return len(p.errors) > 0 }

// ─────────────────────────────────────────────────────────────────────────────
// PUNTO DE ENTRADA PRINCIPAL
// ─────────────────────────────────────────────────────────────────────────────

// ParseProgram parsea el programa completo y retorna el nodo raíz.
func (p *Parser) ParseProgram() *ast.Program {
	program := &ast.Program{}

	for p.currentToken.Type != token.EOF {
		stmt := p.parseStatement()
		if stmt != nil {
			program.Statements = append(program.Statements, stmt)
		}
		p.advance()
	}

	return program
}

// ─────────────────────────────────────────────────────────────────────────────
// PARSEO DE SENTENCIAS
// ─────────────────────────────────────────────────────────────────────────────

func (p *Parser) parseStatement() ast.Statement {
	switch p.currentToken.Type {
	case token.LET:
		return p.parseLetStatement()
	case token.RETURN:
		return p.parseReturnStatement()
	case token.WHILE:
		return p.parseWhileStatement()
	case token.FOR:
		return p.parseForStatement()
	case token.PRINT:
		return p.parsePrintStatement()
	case token.BREAK:
		return p.parseBreakStatement()
	case token.CONTINUE:
		return p.parseContinueStatement()
	default:
		return p.parseExpressionStatement()
	}
}

// parseLetStatement parsea: let <id> = <expr>;
func (p *Parser) parseLetStatement() *ast.LetStatement {
	stmt := &ast.LetStatement{Token: p.currentToken}

	if !p.expectPeek(token.IDENTIFIER) {
		return nil
	}

	stmt.Name = &ast.Identifier{
		Token: p.currentToken,
		Value: p.currentToken.Literal,
	}

	if !p.expectPeek(token.ASSIGN) {
		return nil
	}

	p.advance() // avanza al inicio de la expresión

	stmt.Value = p.parseExpression(LOWEST)

	// Consume el punto y coma opcional
	if p.peekToken.Type == token.SEMICOLON {
		p.advance()
	}

	// Si el valor es una función, guardamos su nombre para la recursión
	if fn, ok := stmt.Value.(*ast.FunctionLiteral); ok {
		fn.Name = stmt.Name.Value
	}

	return stmt
}

// parseReturnStatement parsea: return <expr>;
func (p *Parser) parseReturnStatement() *ast.ReturnStatement {
	stmt := &ast.ReturnStatement{Token: p.currentToken}

	p.advance() // avanza al inicio de la expresión

	stmt.ReturnValue = p.parseExpression(LOWEST)

	if p.peekToken.Type == token.SEMICOLON {
		p.advance()
	}

	return stmt
}

// parseExpressionStatement parsea una expresión como sentencia: expr;
func (p *Parser) parseExpressionStatement() *ast.ExpressionStatement {
	stmt := &ast.ExpressionStatement{Token: p.currentToken}
	stmt.Expression = p.parseExpression(LOWEST)

	if p.peekToken.Type == token.SEMICOLON {
		p.advance()
	}

	return stmt
}

// parsePrintStatement parsea: print(<expr>);
func (p *Parser) parsePrintStatement() *ast.PrintStatement {
	stmt := &ast.PrintStatement{Token: p.currentToken}

	if !p.expectPeek(token.LPAREN) {
		return nil
	}

	p.advance() // al inicio del argumento

	stmt.Value = p.parseExpression(LOWEST)

	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	if p.peekToken.Type == token.SEMICOLON {
		p.advance()
	}

	return stmt
}

// parseWhileStatement parsea: while (<cond>) { <body> }
func (p *Parser) parseWhileStatement() *ast.WhileStatement {
	stmt := &ast.WhileStatement{Token: p.currentToken}

	if !p.expectPeek(token.LPAREN) {
		return nil
	}

	p.advance() // al inicio de la condición

	stmt.Condition = p.parseExpression(LOWEST)

	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	stmt.Body = p.parseBlockStatement()
	return stmt
}

// parseForStatement parsea: for (<init>; <cond>; <upd>) { <body> }
func (p *Parser) parseForStatement() *ast.ForStatement {
	stmt := &ast.ForStatement{Token: p.currentToken}

	if !p.expectPeek(token.LPAREN) {
		return nil
	}

	p.advance() // al inicio del init

	// ── Init ──────────────────────────────────────────────────────────────────
	if p.currentToken.Type == token.SEMICOLON {
		stmt.Init = nil // init vacío
	} else if p.currentToken.Type == token.LET {
		stmt.Init = p.parseForLetClause()
	} else {
		exprToken := p.currentToken
		expr := p.parseExpression(LOWEST)
		stmt.Init = &ast.ExpressionStatement{Token: exprToken, Expression: expr}
	}

	// Consume ';' entre init y condition
	if !p.expectPeek(token.SEMICOLON) {
		return nil
	}
	p.advance()

	// ── Condition ─────────────────────────────────────────────────────────────
	if p.currentToken.Type != token.SEMICOLON {
		stmt.Condition = p.parseExpression(LOWEST)
	}

	if !p.expectPeek(token.SEMICOLON) {
		return nil
	}
	p.advance()

	// ── Update ────────────────────────────────────────────────────────────────
	if p.currentToken.Type != token.RPAREN {
		if p.currentToken.Type == token.LET {
			stmt.Update = p.parseForLetClause()
		} else {
			exprToken := p.currentToken
			expr := p.parseExpression(LOWEST)
			stmt.Update = &ast.ExpressionStatement{Token: exprToken, Expression: expr}
		}
	}

	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	stmt.Body = p.parseBlockStatement()
	return stmt
}

// parseForLetClause parsea un 'let id = expr' dentro de la cabecera del for,
// SIN consumir el ';' (el for lo maneja).
func (p *Parser) parseForLetClause() *ast.LetStatement {
	letToken := p.currentToken
	if !p.expectPeek(token.IDENTIFIER) {
		return nil
	}
	name := &ast.Identifier{Token: p.currentToken, Value: p.currentToken.Literal}
	if !p.expectPeek(token.ASSIGN) {
		return nil
	}
	p.advance()
	value := p.parseExpression(LOWEST)
	return &ast.LetStatement{Token: letToken, Name: name, Value: value}
}

// parseBlockStatement parsea: { stmt; stmt; }
// Asume que currentToken es '{' al entrar.
func (p *Parser) parseBlockStatement() *ast.BlockStatement {
	block := &ast.BlockStatement{Token: p.currentToken}

	p.advance() // avanza al primer token del cuerpo

	for p.currentToken.Type != token.RBRACE && p.currentToken.Type != token.EOF {
		stmt := p.parseStatement()
		if stmt != nil {
			block.Statements = append(block.Statements, stmt)
		}
		p.advance()
	}

	return block
}

// parseBreakStatement parsea: break;
func (p *Parser) parseBreakStatement() *ast.BreakStatement {
	stmt := &ast.BreakStatement{Token: p.currentToken}
	if p.peekToken.Type == token.SEMICOLON {
		p.advance()
	}
	return stmt
}

// parseContinueStatement parsea: continue;
func (p *Parser) parseContinueStatement() *ast.ContinueStatement {
	stmt := &ast.ContinueStatement{Token: p.currentToken}
	if p.peekToken.Type == token.SEMICOLON {
		p.advance()
	}
	return stmt
}

// ─────────────────────────────────────────────────────────────────────────────
// CORAZÓN DEL PRATT PARSER: parseExpression
// ─────────────────────────────────────────────────────────────────────────────

// parseExpression es el núcleo del Pratt Parser.
//
// Algoritmo:
//  1. Busca la función prefija para el token actual → parsea la parte izquierda.
//  2. Mientras el siguiente token tenga mayor precedencia que la actual,
//     busca la función infija del siguiente token y la llama pasando la izquierda.
//  3. Retorna la expresión resultante (que puede ser un árbol complejo).
//
// Ejemplo para "1 + 2 * 3" con precedencia LOWEST:
//   left = IntegerLiteral(1)
//   peek '+' tiene prec SUM > LOWEST → left = InfixExpr(1, +, parseExpr(SUM))
//     dentro: left = IntegerLiteral(2)
//             peek '*' tiene prec PRODUCT > SUM → left = InfixExpr(2, *, 3)
//   resultado: InfixExpr(1, +, InfixExpr(2, *, 3))
func (p *Parser) parseExpression(precedence Precedence) ast.Expression {
	prefix := p.prefixParseFns[p.currentToken.Type]
	if prefix == nil {
		p.noPrefixParseFnError(p.currentToken.Type)
		return nil
	}

	leftExp := prefix()

	for p.peekToken.Type != token.SEMICOLON && precedence < p.peekPrecedence() {
		infix := p.infixParseFns[p.peekToken.Type]
		if infix == nil {
			return leftExp
		}
		p.advance()
		leftExp = infix(leftExp)
	}

	return leftExp
}

// ─────────────────────────────────────────────────────────────────────────────
// FUNCIONES DE PARSEO PREFIJAS (nud)
// ─────────────────────────────────────────────────────────────────────────────

func (p *Parser) parseIdentifier() ast.Expression {
	return &ast.Identifier{Token: p.currentToken, Value: p.currentToken.Literal}
}

func (p *Parser) parseIntegerLiteral() ast.Expression {
	tok := p.currentToken
	value, err := strconv.ParseInt(tok.Literal, 0, 64) // base 0 → detecta hex
	if err != nil {
		p.errors = append(p.errors, fmt.Sprintf(
			"no se pudo convertir %q a entero", tok.Literal))
		return nil
	}
	return &ast.IntegerLiteral{Token: tok, Value: value}
}

func (p *Parser) parseFloatLiteral() ast.Expression {
	tok := p.currentToken
	value, err := strconv.ParseFloat(tok.Literal, 64)
	if err != nil {
		p.errors = append(p.errors, fmt.Sprintf(
			"no se pudo convertir %q a flotante", tok.Literal))
		return nil
	}
	return &ast.FloatLiteral{Token: tok, Value: value}
}

func (p *Parser) parseStringLiteral() ast.Expression {
	return &ast.StringLiteral{Token: p.currentToken, Value: p.currentToken.Literal}
}

func (p *Parser) parseBooleanLiteral() ast.Expression {
	return &ast.BooleanLiteral{
		Token: p.currentToken,
		Value: p.currentToken.Type == token.TRUE,
	}
}

func (p *Parser) parseNilLiteral() ast.Expression {
	return &ast.NilLiteral{Token: p.currentToken}
}

// parsePrefixExpression parsea: <op><expr>  →  !x, -5
func (p *Parser) parsePrefixExpression() ast.Expression {
	expr := &ast.PrefixExpression{
		Token:    p.currentToken,
		Operator: p.currentToken.Literal,
	}
	p.advance()
	expr.Right = p.parseExpression(PREFIX)
	return expr
}

// parseGroupedExpression parsea: (<expr>)
func (p *Parser) parseGroupedExpression() ast.Expression {
	p.advance() // salta el '('

	exp := p.parseExpression(LOWEST)

	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	return exp
}

// parseIfExpression parsea: if (cond) { } [elseif (c) { }]* [else { }]
func (p *Parser) parseIfExpression() ast.Expression {
	expr := &ast.IfExpression{Token: p.currentToken}

	if !p.expectPeek(token.LPAREN) {
		return nil
	}
	p.advance()
	expr.Condition = p.parseExpression(LOWEST)

	if !p.expectPeek(token.RPAREN) {
		return nil
	}
	if !p.expectPeek(token.LBRACE) {
		return nil
	}
	expr.Consequence = p.parseBlockStatement()

	// Parsea todas las ramas elseif
	for p.peekToken.Type == token.ELSEIF {
		p.advance() // al token 'elseif'

		if !p.expectPeek(token.LPAREN) {
			return nil
		}
		p.advance()
		altCond := p.parseExpression(LOWEST)

		if !p.expectPeek(token.RPAREN) {
			return nil
		}
		if !p.expectPeek(token.LBRACE) {
			return nil
		}
		altBlock := p.parseBlockStatement()

		expr.Alternatives = append(expr.Alternatives, ast.ElseIfClause{
			Condition: altCond,
			Body:      altBlock,
		})
	}

	// Parsea la rama else opcional
	if p.peekToken.Type == token.ELSE {
		p.advance() // al token 'else'
		if !p.expectPeek(token.LBRACE) {
			return nil
		}
		expr.ElseBlock = p.parseBlockStatement()
	}

	return expr
}

// parseFunctionLiteral parsea: function(<params>) { <body> }
func (p *Parser) parseFunctionLiteral() ast.Expression {
	lit := &ast.FunctionLiteral{Token: p.currentToken}

	if !p.expectPeek(token.LPAREN) {
		return nil
	}

	lit.Parameters = p.parseFunctionParameters()
	if lit.Parameters == nil && p.HasErrors() {
		return nil
	}

	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	lit.Body = p.parseBlockStatement()
	return lit
}

// parseFunctionParameters parsea la lista de parámetros: (a, b, c)
// Asume que currentToken es '(' al entrar.
func (p *Parser) parseFunctionParameters() []*ast.Identifier {
	var identifiers []*ast.Identifier

	// Lista vacía: function() { }
	if p.peekToken.Type == token.RPAREN {
		p.advance()
		return identifiers
	}

	p.advance() // al primer parámetro

	identifiers = append(identifiers, &ast.Identifier{
		Token: p.currentToken,
		Value: p.currentToken.Literal,
	})

	for p.peekToken.Type == token.COMMA {
		p.advance() // al ','
		p.advance() // al siguiente parámetro
		identifiers = append(identifiers, &ast.Identifier{
			Token: p.currentToken,
			Value: p.currentToken.Literal,
		})
	}

	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	return identifiers
}

// ─────────────────────────────────────────────────────────────────────────────
// FUNCIONES DE PARSEO INFIJAS (led)
// ─────────────────────────────────────────────────────────────────────────────

// parseInfixExpression parsea: <left> <op> <right>
func (p *Parser) parseInfixExpression(left ast.Expression) ast.Expression {
	expr := &ast.InfixExpression{
		Token:    p.currentToken,
		Operator: p.currentToken.Literal,
		Left:     left,
	}

	precedence := p.currentPrecedence()
	p.advance()
	expr.Right = p.parseExpression(precedence)

	return expr
}

// parseCallExpression parsea: <función>(<args>)
func (p *Parser) parseCallExpression(function ast.Expression) ast.Expression {
	expr := &ast.CallExpression{Token: p.currentToken, Function: function}
	expr.Arguments = p.parseCallArguments()
	return expr
}

// parseCallArguments parsea la lista de argumentos: (arg1, arg2, ...)
func (p *Parser) parseCallArguments() []ast.Expression {
	var args []ast.Expression

	// Llamada sin argumentos: f()
	if p.peekToken.Type == token.RPAREN {
		p.advance()
		return args
	}

	p.advance() // al primer argumento

	arg := p.parseExpression(LOWEST)
	if arg != nil {
		args = append(args, arg)
	}

	for p.peekToken.Type == token.COMMA {
		p.advance() // al ','
		p.advance() // al argumento
		arg = p.parseExpression(LOWEST)
		if arg != nil {
			args = append(args, arg)
		}
	}

	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	return args
}

// ─────────────────────────────────────────────────────────────────────────────
// MÉTODOS AUXILIARES
// ─────────────────────────────────────────────────────────────────────────────

// advance hace avanzar el cursor de tokens: current ← peek ← siguiente.
func (p *Parser) advance() {
	p.currentToken = p.peekToken
	p.peekToken = p.l.NextToken()
}

// expectPeek verifica que el próximo token sea del tipo dado y avanza.
// Si no coincide, registra un error descriptivo y retorna false.
func (p *Parser) expectPeek(t token.TokenType) bool {
	if p.peekToken.Type == t {
		p.advance()
		return true
	}
	p.errors = append(p.errors, fmt.Sprintf(
		"se esperaba %q pero se encontró %q (literal: %q)  [L%d:C%d]",
		t, p.peekToken.Type, p.peekToken.Literal,
		p.peekToken.Pos.Line, p.peekToken.Pos.Column,
	))
	return false
}

// peekPrecedence retorna la precedencia del próximo token.
func (p *Parser) peekPrecedence() Precedence {
	if prec, ok := precedences[p.peekToken.Type]; ok {
		return prec
	}
	return LOWEST
}

// currentPrecedence retorna la precedencia del token actual.
func (p *Parser) currentPrecedence() Precedence {
	if prec, ok := precedences[p.currentToken.Type]; ok {
		return prec
	}
	return LOWEST
}

// noPrefixParseFnError registra un error cuando no hay función prefija.
func (p *Parser) noPrefixParseFnError(t token.TokenType) {
	p.errors = append(p.errors, fmt.Sprintf(
		"no se encontró función de parseo para el token %q  [L%d:C%d]",
		t, p.currentToken.Pos.Line, p.currentToken.Pos.Column,
	))
}
