// =============================================================================
// lexer/lexer.go — Analizador Léxico de NovaLang
//
// El Lexer recorre el código fuente carácter a carácter y produce una
// secuencia de Tokens. Es la PRIMERA etapa del intérprete.
//
// Flujo general:
//   código fuente (string)
//       → Lexer.NextToken() × N
//       → []Token
//       → Parser (fase 2, no implementada aquí)
//
// Características especiales de este Lexer:
//   • Rastreo de línea y columna para mensajes de error precisos
//   • Soporte de comentarios de línea (//) y bloque (/* */)
//   • Strings con secuencias de escape (\n, \t, \\, \")
//   • Números flotantes (3.14) y notación científica (1e10)
//   • Operadores compuestos (+=, -=, ==, !=, <=, >=)
//   • Recuperación ante errores: emite ILLEGAL y continúa
// =============================================================================

package lexer

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/novalang/novalang/internal/token"
)

// Lexer mantiene el estado del analizador léxico durante el escaneo.
type Lexer struct {
	source  []rune // Código fuente como slice de runes (soporte Unicode)
	pos     int    // Posición actual (índice del rune actual)
	readPos int    // Posición de lectura anticipada (look-ahead de 1)
	ch      rune   // Carácter actual bajo análisis

	line   int // Línea actual (comienza en 1)
	column int // Columna actual (comienza en 1)

	errors []LexError // Errores léxicos encontrados (sin detener el escaneo)
}

// LexError representa un error léxico con su ubicación en el código fuente.
type LexError struct {
	Msg    string
	Line   int
	Column int
}

func (e LexError) Error() string {
	return fmt.Sprintf("[NovaLang Lexer] Error en línea %d, col %d: %s", e.Line, e.Column, e.Msg)
}

// New crea e inicializa un nuevo Lexer para el código fuente dado.
//
// Ejemplo:
//
//	l := lexer.New("let x = 42;")
//	for tok := l.NextToken(); !tok.IsEOF(); tok = l.NextToken() {
//	    fmt.Println(tok)
//	}
func New(source string) *Lexer {
	l := &Lexer{
		source: []rune(source),
		line:   1,
		column: 0,
	}
	l.advance() // inicializa l.ch con el primer carácter
	return l
}

// Errors retorna todos los errores léxicos encontrados durante el escaneo.
func (l *Lexer) Errors() []LexError {
	return l.errors
}

// HasErrors reporta si se encontraron errores léxicos.
func (l *Lexer) HasErrors() bool {
	return len(l.errors) > 0
}

// ─────────────────────────────────────────────────────────────────────────────
// MÉTODO PRINCIPAL: NextToken
// ─────────────────────────────────────────────────────────────────────────────

// NextToken lee y retorna el siguiente token del código fuente.
//
// Algoritmo:
//  1. Salta espacios en blanco y comentarios.
//  2. Guarda la posición actual (para el token).
//  3. Analiza l.ch con un switch.
//  4. Para tokens de 1 carácter: crea token, avanza, retorna.
//  5. Para tokens de 2 caracteres (==, !=, +=, etc.): peek y decide.
//  6. Para identificadores/números: delega a métodos de lectura.
func (l *Lexer) NextToken() token.Token {
	l.skipWhitespaceAndComments()

	// Captura la posición ANTES de consumir el carácter
	pos := token.Position{Line: l.line, Column: l.column}

	var tok token.Token

	switch l.ch {

	// ── EOF ───────────────────────────────────────────────────────────────────
	case 0:
		tok = l.makeToken(token.EOF, "", pos)

	// ── Operadores aritméticos ────────────────────────────────────────────────
	case '+':
		if l.peek() == '=' {
			l.advance()
			tok = l.makeToken(token.PLUS_ASSIGN, "+=", pos)
		} else {
			tok = l.makeToken(token.PLUS, "+", pos)
		}
	case '-':
		if l.peek() == '=' {
			l.advance()
			tok = l.makeToken(token.MINUS_ASSIGN, "-=", pos)
		} else {
			tok = l.makeToken(token.MINUS, "-", pos)
		}
	case '*':
		tok = l.makeToken(token.MULTIPLY, "*", pos)
	case '/':
		// Nota: '//' y '/*' ya fueron consumidos por skipWhitespaceAndComments
		tok = l.makeToken(token.DIVISION, "/", pos)
	case '%':
		tok = l.makeToken(token.MOD, "%", pos)
	case '^':
		tok = l.makeToken(token.POW, "^", pos)

	// ── Asignación y comparación ──────────────────────────────────────────────
	case '=':
		if l.peek() == '=' {
			l.advance()
			tok = l.makeToken(token.EQ, "==", pos)
		} else {
			tok = l.makeToken(token.ASSIGN, "=", pos)
		}
	case '!':
		if l.peek() == '=' {
			l.advance()
			tok = l.makeToken(token.NEQ, "!=", pos)
		} else {
			tok = l.makeToken(token.NEGATION, "!", pos)
		}
	case '<':
		if l.peek() == '=' {
			l.advance()
			tok = l.makeToken(token.LTE, "<=", pos)
		} else {
			tok = l.makeToken(token.LT, "<", pos)
		}
	case '>':
		if l.peek() == '=' {
			l.advance()
			tok = l.makeToken(token.GTE, ">=", pos)
		} else {
			tok = l.makeToken(token.GT, ">", pos)
		}

	// ── Delimitadores ─────────────────────────────────────────────────────────
	case ',':
		tok = l.makeToken(token.COMMA, ",", pos)
	case ';':
		tok = l.makeToken(token.SEMICOLON, ";", pos)
	case ':':
		tok = l.makeToken(token.COLON, ":", pos)
	case '(':
		tok = l.makeToken(token.LPAREN, "(", pos)
	case ')':
		tok = l.makeToken(token.RPAREN, ")", pos)
	case '{':
		tok = l.makeToken(token.LBRACE, "{", pos)
	case '}':
		tok = l.makeToken(token.RBRACE, "}", pos)
	case '[':
		tok = l.makeToken(token.LBRACKET, "[", pos)
	case ']':
		tok = l.makeToken(token.RBRACKET, "]", pos)

	// ── String literal ────────────────────────────────────────────────────────
	case '"':
		str, ok := l.readString()
		if !ok {
			l.addError("string no cerrado (falta comilla de cierre)", pos.Line, pos.Column)
			return l.makeToken(token.ILLEGAL, str, pos)
		}
		return l.makeToken(token.STRING, str, pos)

	// ── Identificadores, keywords y números ───────────────────────────────────
	default:
		if isLetter(l.ch) {
			literal := l.readIdentifier()
			tt := token.LookupIdentifier(literal)
			return l.makeToken(tt, literal, pos)
		}
		if unicode.IsDigit(l.ch) {
			literal, tt := l.readNumber()
			return l.makeToken(tt, literal, pos)
		}

		// Carácter ilegal
		l.addError("carácter no reconocido: '"+string(l.ch)+"'", pos.Line, pos.Column)
		tok = l.makeToken(token.ILLEGAL, string(l.ch), pos)
	}

	l.advance()
	return tok
}

// ─────────────────────────────────────────────────────────────────────────────
// MÉTODOS AUXILIARES INTERNOS
// ─────────────────────────────────────────────────────────────────────────────

// advance avanza un carácter en el código fuente.
// Actualiza línea y columna para rastreo de posición exacta.
func (l *Lexer) advance() {
	if l.readPos >= len(l.source) {
		l.ch = 0 // rune 0 = EOF
	} else {
		l.ch = l.source[l.readPos]
	}
	l.pos = l.readPos
	l.readPos++

	// Rastreo de líneas: al pasar un \n, incrementa línea y resetea columna
	if l.ch == '\n' {
		l.line++
		l.column = 0
	} else {
		l.column++
	}
}

// peek devuelve el siguiente carácter SIN avanzar (look-ahead de 1).
// Retorna 0 si estamos al final del archivo.
func (l *Lexer) peek() rune {
	if l.readPos >= len(l.source) {
		return 0
	}
	return l.source[l.readPos]
}

// peekAt devuelve el carácter N posiciones adelante (look-ahead de N).
func (l *Lexer) peekAt(n int) rune {
	idx := l.readPos + n - 1
	if idx >= len(l.source) {
		return 0
	}
	return l.source[idx]
}

// makeToken crea un Token con el tipo, literal y posición dados.
func (l *Lexer) makeToken(tt token.TokenType, literal string, pos token.Position) token.Token {
	return token.Token{Type: tt, Literal: literal, Pos: pos}
}

// addError registra un error léxico sin detener el escaneo.
func (l *Lexer) addError(msg string, line, col int) {
	l.errors = append(l.errors, LexError{Msg: msg, Line: line, Column: col})
}

// skipWhitespaceAndComments avanza mientras el carácter actual sea:
//   - espacio, tab, retorno de carro, salto de línea
//   - inicio de comentario de línea: //
//   - inicio de comentario de bloque: /* ... */
func (l *Lexer) skipWhitespaceAndComments() {
	for {
		switch {
		case l.ch == ' ' || l.ch == '\t' || l.ch == '\r' || l.ch == '\n':
			l.advance()

		case l.ch == '/' && l.peek() == '/':
			// Comentario de línea: salta hasta el fin de línea
			for l.ch != '\n' && l.ch != 0 {
				l.advance()
			}

		case l.ch == '/' && l.peek() == '*':
			// Comentario de bloque: salta hasta encontrar */
			startLine := l.line
			startCol := l.column
			l.advance() // consume '/'
			l.advance() // consume '*'
			for {
				if l.ch == 0 {
					l.addError("comentario de bloque no cerrado (falta */)", startLine, startCol)
					return
				}
				if l.ch == '*' && l.peek() == '/' {
					l.advance() // consume '*'
					l.advance() // consume '/'
					break
				}
				l.advance()
			}

		default:
			return
		}
	}
}

// isLetter reporta si un rune puede ser parte de un identificador NovaLang.
// Permitimos letras Unicode + guion bajo.
func isLetter(ch rune) bool {
	return unicode.IsLetter(ch) || ch == '_'
}

// readIdentifier lee una secuencia de letras/dígitos/guiones_bajos
// y retorna el string completo.
//
// Ejemplo: 'miVariable123' → "miVariable123"
func (l *Lexer) readIdentifier() string {
	var sb strings.Builder
	for isLetter(l.ch) || unicode.IsDigit(l.ch) {
		sb.WriteRune(l.ch)
		l.advance()
	}
	return sb.String()
}

// readNumber lee un número entero o flotante (con punto decimal o notación e).
//
// Formatos soportados:
//   - Entero:   42, 1000
//   - Flotante: 3.14, 0.5
//   - Científico: 1.5e10, 2e-3  (se clasifica como FLOAT)
//   - Hexadecimal: 0xFF, 0x1A   (se clasifica como INTEGER)
//
// Retorna (literal, TokenType).
func (l *Lexer) readNumber() (string, token.TokenType) {
	var sb strings.Builder
	tt := token.INTEGER

	// Soporte hexadecimal: 0x...
	if l.ch == '0' && (l.peek() == 'x' || l.peek() == 'X') {
		sb.WriteRune(l.ch) // '0'
		l.advance()
		sb.WriteRune(l.ch) // 'x'
		l.advance()
		for isHexDigit(l.ch) {
			sb.WriteRune(l.ch)
			l.advance()
		}
		return sb.String(), token.INTEGER
	}

	// Parte entera
	for unicode.IsDigit(l.ch) {
		sb.WriteRune(l.ch)
		l.advance()
	}

	// Parte decimal: .dígitos
	if l.ch == '.' && unicode.IsDigit(l.peek()) {
		tt = token.FLOAT
		sb.WriteRune(l.ch) // '.'
		l.advance()
		for unicode.IsDigit(l.ch) {
			sb.WriteRune(l.ch)
			l.advance()
		}
	}

	// Notación científica: e+N, e-N, eN
	if l.ch == 'e' || l.ch == 'E' {
		tt = token.FLOAT
		sb.WriteRune(l.ch)
		l.advance()
		if l.ch == '+' || l.ch == '-' {
			sb.WriteRune(l.ch)
			l.advance()
		}
		for unicode.IsDigit(l.ch) {
			sb.WriteRune(l.ch)
			l.advance()
		}
	}

	return sb.String(), tt
}

// isHexDigit reporta si un rune es un dígito hexadecimal válido.
func isHexDigit(ch rune) bool {
	return (ch >= '0' && ch <= '9') ||
		(ch >= 'a' && ch <= 'f') ||
		(ch >= 'A' && ch <= 'F')
}

// readString lee el contenido de un string entre comillas dobles,
// procesando secuencias de escape estándar.
//
// Escapes soportados: \n \t \r \\ \"
//
// Retorna (contenido, ok). ok=false si el string no fue cerrado.
func (l *Lexer) readString() (string, bool) {
	l.advance() // salta la comilla de apertura '"'
	var sb strings.Builder

	for {
		switch l.ch {
		case 0: // EOF sin cerrar el string
			return sb.String(), false

		case '"': // comilla de cierre
			l.advance() // salta '"'
			return sb.String(), true

		case '\\': // secuencia de escape
			l.advance()
			switch l.ch {
			case 'n':
				sb.WriteRune('\n')
			case 't':
				sb.WriteRune('\t')
			case 'r':
				sb.WriteRune('\r')
			case '\\':
				sb.WriteRune('\\')
			case '"':
				sb.WriteRune('"')
			default:
				// Escape desconocido: lo incluimos tal cual
				sb.WriteRune('\\')
				sb.WriteRune(l.ch)
			}
			l.advance()

		default:
			sb.WriteRune(l.ch)
			l.advance()
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// HELPER: Tokenize completo
// ─────────────────────────────────────────────────────────────────────────────

// Tokenize es una función de conveniencia que retorna TODOS los tokens
// del código fuente de una sola vez (útil para tests y herramientas).
func Tokenize(source string) ([]token.Token, []LexError) {
	l := New(source)
	var tokens []token.Token
	for {
		tok := l.NextToken()
		tokens = append(tokens, tok)
		if tok.IsEOF() {
			break
		}
	}
	return tokens, l.Errors()
}
