// =============================================================================
// token/token.go — Definición formal de tokens para NovaLang
//
// NovaLang es un lenguaje de scripting de propósito general inspirado en
// la simplicidad de Python y la expresividad de Go.
//
// Un TOKEN es la unidad atómica de significado: el Lexer descompone el
// código fuente en estos bloques antes de que el Parser los analice.
// =============================================================================

package token

import "fmt"

// TokenType es el tipo que identifica a qué categoría pertenece un token.
type TokenType string

// ─────────────────────────────────────────────────────────────────────────────
// CONSTANTES DE TIPOS DE TOKEN
// ─────────────────────────────────────────────────────────────────────────────

const (
	// ── Especiales ────────────────────────────────────────────────────────────
	EOF     TokenType = "EOF"     // Fin de archivo / input
	ILLEGAL TokenType = "ILLEGAL" // Carácter no reconocido

	// ── Identificadores y literales ───────────────────────────────────────────
	IDENTIFIER TokenType = "IDENTIFIER" // nombre de variable o función
	INTEGER    TokenType = "INTEGER"    // 42
	FLOAT      TokenType = "FLOAT"      // 3.14
	STRING     TokenType = "STRING"     // "hola mundo"
	TRUE       TokenType = "TRUE"       // true
	FALSE      TokenType = "FALSE"      // false

	// ── Operadores aritméticos ────────────────────────────────────────────────
	PLUS     TokenType = "+"  // Suma
	MINUS    TokenType = "-"  // Resta
	MULTIPLY TokenType = "*"  // Multiplicación
	DIVISION TokenType = "/"  // División
	MOD      TokenType = "%"  // Módulo
	POW      TokenType = "^"  // Potencia

	// ── Operadores de comparación ─────────────────────────────────────────────
	EQ  TokenType = "==" // Igual
	NEQ TokenType = "!=" // Diferente
	LT  TokenType = "<"  // Menor que
	LTE TokenType = "<=" // Menor o igual
	GT  TokenType = ">"  // Mayor que
	GTE TokenType = ">=" // Mayor o igual

	// ── Operadores lógicos ────────────────────────────────────────────────────
	AND     TokenType = "AND" // and
	OR      TokenType = "OR"  // or
	NEGATION TokenType = "!"   // Negación lógica

	// ── Asignación ────────────────────────────────────────────────────────────
	ASSIGN     TokenType = "="  // Asignación simple
	PLUS_ASSIGN TokenType = "+=" // Suma y asigna
	MINUS_ASSIGN TokenType = "-=" // Resta y asigna

	// ── Delimitadores ─────────────────────────────────────────────────────────
	COMMA     TokenType = ","  // Separador de argumentos
	SEMICOLON TokenType = ";"  // Fin de sentencia
	COLON     TokenType = ":"  // Para futura sintaxis dict/map
	LPAREN    TokenType = "("  // Paréntesis izquierdo
	RPAREN    TokenType = ")"  // Paréntesis derecho
	LBRACE    TokenType = "{"  // Llave izquierda
	RBRACE    TokenType = "}"  // Llave derecha
	LBRACKET  TokenType = "["  // Corchete izquierdo (para listas)
	RBRACKET  TokenType = "]"  // Corchete derecho

	// ── Palabras reservadas ───────────────────────────────────────────────────
	FUNCTION TokenType = "function" // Definir función
	LET      TokenType = "let"      // Declarar variable
	CONST    TokenType = "const"    // Variable constante
	RETURN   TokenType = "return"   // Retornar valor
	IF       TokenType = "if"       // Condicional
	ELSEIF   TokenType = "elseif"   // Rama adicional
	ELSE     TokenType = "else"     // Rama por defecto
	WHILE    TokenType = "while"    // Bucle while
	FOR      TokenType = "for"      // Bucle for
	IN       TokenType = "in"       // Iterador (for x in lista)
	BREAK    TokenType = "break"    // Romper bucle
	CONTINUE TokenType = "continue" // Saltar iteración
	PRINT    TokenType = "print"    // Imprimir en consola
	IMPORT   TokenType = "import"   // Importar módulo
	NIL      TokenType = "nil"      // Valor nulo
)

// ─────────────────────────────────────────────────────────────────────────────
// TABLA DE PALABRAS RESERVADAS
// ─────────────────────────────────────────────────────────────────────────────

// keywords mapea texto → TokenType para todas las palabras reservadas.
// LookupIdentifier usa este mapa para distinguir keywords de identificadores.
var keywords = map[string]TokenType{
	"function": FUNCTION,
	"let":      LET,
	"const":    CONST,
	"return":   RETURN,
	"if":       IF,
	"elseif":   ELSEIF,
	"else":     ELSE,
	"while":    WHILE,
	"for":      FOR,
	"in":       IN,
	"break":    BREAK,
	"continue": CONTINUE,
	"true":     TRUE,
	"false":    FALSE,
	"and":      AND,
	"or":       OR,
	"print":    PRINT,
	"import":   IMPORT,
	"nil":      NIL,
}

// LookupIdentifier decide si un literal es keyword o identificador de usuario.
//
// Ejemplo:
//
//	LookupIdentifier("if")       → IF
//	LookupIdentifier("miVar")    → IDENTIFIER
func LookupIdentifier(literal string) TokenType {
	if tt, ok := keywords[literal]; ok {
		return tt
	}
	return IDENTIFIER
}

// IsKeyword reporta si un literal es una palabra reservada del lenguaje.
func IsKeyword(literal string) bool {
	_, ok := keywords[literal]
	return ok
}

// ─────────────────────────────────────────────────────────────────────────────
// ESTRUCTURA TOKEN
// ─────────────────────────────────────────────────────────────────────────────

// Position almacena la ubicación exacta del token en el código fuente.
// Esto permite mensajes de error precisos: "error en línea 5, columna 12".
type Position struct {
	Line   int // Número de línea (comienza en 1)
	Column int // Número de columna (comienza en 1)
}

// Token representa una unidad léxica del código fuente NovaLang.
//
// Ejemplo para `let x = 42;`:
//
//	Token{Type: LET,        Literal: "let",  Pos: {1,1}}
//	Token{Type: IDENTIFIER, Literal: "x",    Pos: {1,5}}
//	Token{Type: ASSIGN,     Literal: "=",    Pos: {1,7}}
//	Token{Type: INTEGER,    Literal: "42",   Pos: {1,9}}
//	Token{Type: SEMICOLON,  Literal: ";",    Pos: {1,11}}
type Token struct {
	Type    TokenType // Categoría del token
	Literal string    // Texto original tal como aparece en el código fuente
	Pos     Position  // Ubicación en el código fuente (línea, columna)
}

// String retorna una representación legible del token para debug y REPL.
func (t Token) String() string {
	return fmt.Sprintf("Token{Type:%-12s Literal:%-15q Line:%d Col:%d}",
		string(t.Type), t.Literal, t.Pos.Line, t.Pos.Column)
}

// IsEOF reporta si el token es fin de archivo.
func (t Token) IsEOF() bool {
	return t.Type == EOF
}

// IsIllegal reporta si el token contiene un carácter inválido.
func (t Token) IsIllegal() bool {
	return t.Type == ILLEGAL
}
