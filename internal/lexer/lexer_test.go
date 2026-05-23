// =============================================================================
// lexer/lexer_test.go — Tests del Lexer de NovaLang
//
// Ejecutar: go test ./internal/lexer/... -v
// Cobertura: go test ./internal/lexer/... -cover
// =============================================================================

package lexer

import (
	"testing"

	"github.com/novalang/novalang/internal/token"
)

// helper: verifica que los tokens producidos coincidan con los esperados
func checkTokens(t *testing.T, input string, expected []token.Token) {
	t.Helper()
	tokens, errs := Tokenize(input)

	for _, e := range errs {
		t.Logf("LexError: %s", e.Error())
	}

	// Ignoramos el EOF final al comparar
	got := tokens
	if len(got) > 0 && got[len(got)-1].IsEOF() {
		got = got[:len(got)-1]
	}

	if len(got) != len(expected) {
		t.Fatalf("input=%q: se esperaban %d tokens, se obtuvieron %d\ngot: %v",
			input, len(expected), len(got), got)
	}

	for i, exp := range expected {
		if got[i].Type != exp.Type {
			t.Errorf("token[%d] tipo: esperado=%q, obtenido=%q (literal=%q)",
				i, exp.Type, got[i].Type, got[i].Literal)
		}
		if got[i].Literal != exp.Literal {
			t.Errorf("token[%d] literal: esperado=%q, obtenido=%q",
				i, exp.Literal, got[i].Literal)
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// TESTS DE OPERADORES
// ─────────────────────────────────────────────────────────────────────────────

func TestOperadoresSimples(t *testing.T) {
	checkTokens(t, "+ - * / % ^", []token.Token{
		{Type: token.PLUS, Literal: "+"},
		{Type: token.MINUS, Literal: "-"},
		{Type: token.MULTIPLY, Literal: "*"},
		{Type: token.DIVISION, Literal: "/"},
		{Type: token.MOD, Literal: "%"},
		{Type: token.POW, Literal: "^"},
	})
}

func TestOperadoresCompuestos(t *testing.T) {
	checkTokens(t, "+= -=", []token.Token{
		{Type: token.PLUS_ASSIGN, Literal: "+="},
		{Type: token.MINUS_ASSIGN, Literal: "-="},
	})
}

func TestOperadoresComparacion(t *testing.T) {
	checkTokens(t, "== != < <= > >=", []token.Token{
		{Type: token.EQ, Literal: "=="},
		{Type: token.NEQ, Literal: "!="},
		{Type: token.LT, Literal: "<"},
		{Type: token.LTE, Literal: "<="},
		{Type: token.GT, Literal: ">"},
		{Type: token.GTE, Literal: ">="},
	})
}

func TestAsignacion(t *testing.T) {
	checkTokens(t, "x = 5", []token.Token{
		{Type: token.IDENTIFIER, Literal: "x"},
		{Type: token.ASSIGN, Literal: "="},
		{Type: token.INTEGER, Literal: "5"},
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// TESTS DE LITERALES
// ─────────────────────────────────────────────────────────────────────────────

func TestLiteralEntero(t *testing.T) {
	checkTokens(t, "42 0 1000", []token.Token{
		{Type: token.INTEGER, Literal: "42"},
		{Type: token.INTEGER, Literal: "0"},
		{Type: token.INTEGER, Literal: "1000"},
	})
}

func TestLiteralFlotante(t *testing.T) {
	checkTokens(t, "3.14 0.5 100.0", []token.Token{
		{Type: token.FLOAT, Literal: "3.14"},
		{Type: token.FLOAT, Literal: "0.5"},
		{Type: token.FLOAT, Literal: "100.0"},
	})
}

func TestLiteralHexadecimal(t *testing.T) {
	checkTokens(t, "0xFF 0x1A", []token.Token{
		{Type: token.INTEGER, Literal: "0xFF"},
		{Type: token.INTEGER, Literal: "0x1A"},
	})
}

func TestLiteralNotacionCientifica(t *testing.T) {
	checkTokens(t, "1e10 2.5e-3", []token.Token{
		{Type: token.FLOAT, Literal: "1e10"},
		{Type: token.FLOAT, Literal: "2.5e-3"},
	})
}

func TestLiteralString(t *testing.T) {
	checkTokens(t, `"hola mundo"`, []token.Token{
		{Type: token.STRING, Literal: "hola mundo"},
	})
}

func TestLiteralStringConEscapes(t *testing.T) {
	checkTokens(t, `"línea1\nlínea2"`, []token.Token{
		{Type: token.STRING, Literal: "línea1\nlínea2"},
	})
}

func TestLiteralesBooleanos(t *testing.T) {
	checkTokens(t, "true false", []token.Token{
		{Type: token.TRUE, Literal: "true"},
		{Type: token.FALSE, Literal: "false"},
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// TESTS DE PALABRAS RESERVADAS
// ─────────────────────────────────────────────────────────────────────────────

func TestKeywords(t *testing.T) {
	input := "function let const return if elseif else while for in break continue print import nil"
	expected := []token.Token{
		{Type: token.FUNCTION, Literal: "function"},
		{Type: token.LET, Literal: "let"},
		{Type: token.CONST, Literal: "const"},
		{Type: token.RETURN, Literal: "return"},
		{Type: token.IF, Literal: "if"},
		{Type: token.ELSEIF, Literal: "elseif"},
		{Type: token.ELSE, Literal: "else"},
		{Type: token.WHILE, Literal: "while"},
		{Type: token.FOR, Literal: "for"},
		{Type: token.IN, Literal: "in"},
		{Type: token.BREAK, Literal: "break"},
		{Type: token.CONTINUE, Literal: "continue"},
		{Type: token.PRINT, Literal: "print"},
		{Type: token.IMPORT, Literal: "import"},
		{Type: token.NIL, Literal: "nil"},
	}
	checkTokens(t, input, expected)
}

func TestOperadoresLogicos(t *testing.T) {
	checkTokens(t, "and or !", []token.Token{
		{Type: token.AND, Literal: "and"},
		{Type: token.OR, Literal: "or"},
		{Type: token.NEGATION, Literal: "!"},
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// TESTS DE IDENTIFICADORES
// ─────────────────────────────────────────────────────────────────────────────

func TestIdentificadores(t *testing.T) {
	checkTokens(t, "x miVariable _privado camelCase snake_case var123", []token.Token{
		{Type: token.IDENTIFIER, Literal: "x"},
		{Type: token.IDENTIFIER, Literal: "miVariable"},
		{Type: token.IDENTIFIER, Literal: "_privado"},
		{Type: token.IDENTIFIER, Literal: "camelCase"},
		{Type: token.IDENTIFIER, Literal: "snake_case"},
		{Type: token.IDENTIFIER, Literal: "var123"},
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// TESTS DE DELIMITADORES
// ─────────────────────────────────────────────────────────────────────────────

func TestDelimitadores(t *testing.T) {
	checkTokens(t, "( ) { } [ ] , ; :", []token.Token{
		{Type: token.LPAREN, Literal: "("},
		{Type: token.RPAREN, Literal: ")"},
		{Type: token.LBRACE, Literal: "{"},
		{Type: token.RBRACE, Literal: "}"},
		{Type: token.LBRACKET, Literal: "["},
		{Type: token.RBRACKET, Literal: "]"},
		{Type: token.COMMA, Literal: ","},
		{Type: token.SEMICOLON, Literal: ";"},
		{Type: token.COLON, Literal: ":"},
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// TESTS DE COMENTARIOS
// ─────────────────────────────────────────────────────────────────────────────

func TestComentarioLinea(t *testing.T) {
	checkTokens(t, "let x = 5; // esto es un comentario", []token.Token{
		{Type: token.LET, Literal: "let"},
		{Type: token.IDENTIFIER, Literal: "x"},
		{Type: token.ASSIGN, Literal: "="},
		{Type: token.INTEGER, Literal: "5"},
		{Type: token.SEMICOLON, Literal: ";"},
	})
}

func TestComentarioBloque(t *testing.T) {
	checkTokens(t, "let /* comentario de bloque */ x = 1;", []token.Token{
		{Type: token.LET, Literal: "let"},
		{Type: token.IDENTIFIER, Literal: "x"},
		{Type: token.ASSIGN, Literal: "="},
		{Type: token.INTEGER, Literal: "1"},
		{Type: token.SEMICOLON, Literal: ";"},
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// TESTS DE EXPRESIONES COMPLETAS
// ─────────────────────────────────────────────────────────────────────────────

func TestDeclaracionVariable(t *testing.T) {
	checkTokens(t, `let nombre = "Juan";`, []token.Token{
		{Type: token.LET, Literal: "let"},
		{Type: token.IDENTIFIER, Literal: "nombre"},
		{Type: token.ASSIGN, Literal: "="},
		{Type: token.STRING, Literal: "Juan"},
		{Type: token.SEMICOLON, Literal: ";"},
	})
}

func TestDeclaracionFuncion(t *testing.T) {
	checkTokens(t, "function suma(a, b) { return a + b; }", []token.Token{
		{Type: token.FUNCTION, Literal: "function"},
		{Type: token.IDENTIFIER, Literal: "suma"},
		{Type: token.LPAREN, Literal: "("},
		{Type: token.IDENTIFIER, Literal: "a"},
		{Type: token.COMMA, Literal: ","},
		{Type: token.IDENTIFIER, Literal: "b"},
		{Type: token.RPAREN, Literal: ")"},
		{Type: token.LBRACE, Literal: "{"},
		{Type: token.RETURN, Literal: "return"},
		{Type: token.IDENTIFIER, Literal: "a"},
		{Type: token.PLUS, Literal: "+"},
		{Type: token.IDENTIFIER, Literal: "b"},
		{Type: token.SEMICOLON, Literal: ";"},
		{Type: token.RBRACE, Literal: "}"},
	})
}

func TestCondicional(t *testing.T) {
	checkTokens(t, "if (x > 0) { print x; }", []token.Token{
		{Type: token.IF, Literal: "if"},
		{Type: token.LPAREN, Literal: "("},
		{Type: token.IDENTIFIER, Literal: "x"},
		{Type: token.GT, Literal: ">"},
		{Type: token.INTEGER, Literal: "0"},
		{Type: token.RPAREN, Literal: ")"},
		{Type: token.LBRACE, Literal: "{"},
		{Type: token.PRINT, Literal: "print"},
		{Type: token.IDENTIFIER, Literal: "x"},
		{Type: token.SEMICOLON, Literal: ";"},
		{Type: token.RBRACE, Literal: "}"},
	})
}

func TestBucleWhile(t *testing.T) {
	checkTokens(t, "while (i < 10) { i += 1; }", []token.Token{
		{Type: token.WHILE, Literal: "while"},
		{Type: token.LPAREN, Literal: "("},
		{Type: token.IDENTIFIER, Literal: "i"},
		{Type: token.LT, Literal: "<"},
		{Type: token.INTEGER, Literal: "10"},
		{Type: token.RPAREN, Literal: ")"},
		{Type: token.LBRACE, Literal: "{"},
		{Type: token.IDENTIFIER, Literal: "i"},
		{Type: token.PLUS_ASSIGN, Literal: "+="},
		{Type: token.INTEGER, Literal: "1"},
		{Type: token.SEMICOLON, Literal: ";"},
		{Type: token.RBRACE, Literal: "}"},
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// TESTS DE POSICIÓN (línea y columna)
// ─────────────────────────────────────────────────────────────────────────────

func TestPosicionTokens(t *testing.T) {
	tokens, _ := Tokenize("let x = 5;")

	cases := []struct {
		idx  int
		line int
		col  int
	}{
		{0, 1, 1}, // let  → L1, C1
		{1, 1, 5}, // x    → L1, C5
		{2, 1, 7}, // =    → L1, C7
		{3, 1, 9}, // 5    → L1, C9
		{4, 1, 10}, // ;    → L1, C10
	}

	for _, c := range cases {
		tok := tokens[c.idx]
		if tok.Pos.Line != c.line || tok.Pos.Column != c.col {
			t.Errorf("token[%d]=%q: posición esperada L%d:C%d, obtenida L%d:C%d",
				c.idx, tok.Literal, c.line, c.col, tok.Pos.Line, tok.Pos.Column)
		}
	}
}

func TestPosicionMultilinea(t *testing.T) {
	src := "let x = 1;\nlet y = 2;"
	tokens, _ := Tokenize(src)
	// 'let' de la segunda línea
	letL2 := tokens[5]
	if letL2.Literal != "let" || letL2.Pos.Line != 2 {
		t.Errorf("esperaba 'let' en línea 2, obtenido %q en L%d", letL2.Literal, letL2.Pos.Line)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// TESTS DE ERRORES LÉXICOS
// ─────────────────────────────────────────────────────────────────────────────

func TestCaracterIlegal(t *testing.T) {
	tokens, errs := Tokenize("let x = @;")
	if len(errs) == 0 {
		t.Error("se esperaba al menos un error léxico por '@'")
	}
	// Debe continuar y producir el resto de tokens
	found := false
	for _, tok := range tokens {
		if tok.Type == token.ILLEGAL {
			found = true
			break
		}
	}
	if !found {
		t.Error("se esperaba un token ILLEGAL en la salida")
	}
}

func TestStringNocerrado(t *testing.T) {
	_, errs := Tokenize(`"string sin cerrar`)
	if len(errs) == 0 {
		t.Error("se esperaba error por string no cerrado")
	}
}

func TestComentarioBloqueNoFinalizado(t *testing.T) {
	_, errs := Tokenize("let x = /* comentario sin cerrar")
	if len(errs) == 0 {
		t.Error("se esperaba error por comentario de bloque no cerrado")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// TESTS DE PROGRAMA COMPLETO
// ─────────────────────────────────────────────────────────────────────────────

func TestProgramaCompleto(t *testing.T) {
	src := `
// Cálculo del factorial
function factorial(n) {
    if (n <= 1) {
        return 1;
    }
    return n * factorial(n - 1);
}

let resultado = factorial(5);
print resultado;
`
	tokens, errs := Tokenize(src)

	if len(errs) > 0 {
		t.Errorf("no se esperaban errores, se obtuvieron %d", len(errs))
	}

	// Debe haber tokens (sin contar EOF)
	tokenCount := len(tokens) - 1
	if tokenCount < 20 {
		t.Errorf("se esperaban al menos 20 tokens, se obtuvieron %d", tokenCount)
	}
}
