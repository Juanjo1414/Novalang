// =============================================================================
// repl/repl.go — REPL interactivo de NovaLang (Lexer + Parser + Evaluador)
// =============================================================================

package repl

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/novalang/novalang/internal/evaluator"
	"github.com/novalang/novalang/internal/lexer"
	"github.com/novalang/novalang/internal/object"
	"github.com/novalang/novalang/internal/parser"
	"github.com/novalang/novalang/internal/token"
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
	colorCyan   = "\033[36m"
	colorWhite  = "\033[37m"
	colorBold   = "\033[1m"
	colorDim    = "\033[2m"
)

type Mode string

const (
	ModeEval   Mode = "eval"
	ModeLexer  Mode = "lexer"
	ModeParser Mode = "parser"
)

type Config struct {
	ShowPos     bool
	ColorOutput bool
	ShowLegend  bool
	Mode        Mode
}

func DefaultConfig() Config {
	return Config{
		ShowPos:     false,
		ColorOutput: true,
		ShowLegend:  true,
		Mode:        ModeEval,
	}
}

type REPL struct {
	cfg     Config
	scanner *bufio.Scanner
	env     *object.Environment
	history []string
}

func New(cfg Config) *REPL {
	return &REPL{
		cfg:     cfg,
		scanner: bufio.NewScanner(os.Stdin),
		env:     object.NewEnvironment(),
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// BUCLE PRINCIPAL
// ─────────────────────────────────────────────────────────────────────────────

func (r *REPL) Start() {
	r.printBanner()
	if r.cfg.ShowLegend && r.cfg.Mode == ModeLexer {
		r.printLegend()
	}

	for {
		fmt.Print(r.prompt())

		if !r.scanner.Scan() {
			fmt.Println("\n" + colorDim + "Hasta luego. ¡Sigue programando!" + colorReset)
			return
		}

		input := strings.TrimSpace(r.scanner.Text())
		if input == "" {
			continue
		}

		if strings.HasPrefix(input, ".") {
			if r.handleCommand(input) {
				return
			}
			continue
		}

		r.history = append(r.history, input)
		r.processInput(input)
	}
}

func (r *REPL) prompt() string {
	modeStr := ""
	switch r.cfg.Mode {
	case ModeLexer:
		modeStr = colorYellow + "[lexer]" + colorReset + " "
	case ModeParser:
		modeStr = colorPurple + "[parser]" + colorReset + " "
	}
	return colorCyan + colorBold + "nova» " + colorReset + modeStr
}

func (r *REPL) RunFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("no se pudo leer el archivo: %w", err)
	}
	fmt.Printf("\n%s%s Ejecutando: %s%s\n\n", colorBold, colorCyan, path, colorReset)
	r.processInput(string(data))
	return nil
}

func (r *REPL) ProcessInline(input string) {
	fmt.Printf("%sAnalizando expresión:%s %s%s%s\n\n",
		colorDim, colorReset, colorWhite+colorBold, input, colorReset)
	r.cfg.Mode = ModeLexer
	r.processInput(input)
}

// ─────────────────────────────────────────────────────────────────────────────
// PROCESAMIENTO CENTRAL
// ─────────────────────────────────────────────────────────────────────────────

func (r *REPL) processInput(input string) {
	switch r.cfg.Mode {
	case ModeLexer:
		r.runLexer(input)
	case ModeParser:
		r.runParser(input)
	default:
		r.runEval(input)
	}
}

func (r *REPL) runLexer(input string) {
	start := time.Now()
	l := lexer.New(input)
	var tokens []token.Token
	for {
		tok := l.NextToken()
		if tok.IsEOF() {
			break
		}
		tokens = append(tokens, tok)
	}
	elapsed := time.Since(start)

	for _, e := range l.Errors() {
		fmt.Printf("%s%s ✗ %s%s\n", colorBold, colorRed, e.Error(), colorReset)
	}

	fmt.Println()
	illegalCount := 0
	for _, tok := range tokens {
		if tok.IsIllegal() {
			illegalCount++
		}
		r.printToken(tok)
	}
	r.printStats(len(tokens), illegalCount, len(l.Errors()), elapsed)
}

func (r *REPL) runParser(input string) {
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if p.HasErrors() {
		r.printParseErrors(p.Errors())
		return
	}

	fmt.Println(colorDim + "── AST ─────────────────────────────────────" + colorReset)
	fmt.Println(program.String())
}

// runEval: el mismo lexer se pasa al parser (no se crea uno nuevo)
// así los tokens producidos no se pierden.
func (r *REPL) runEval(input string) {
	l := lexer.New(input)
	p := parser.New(l) // el parser llama l.NextToken() internamente

	program := p.ParseProgram()

	// Mostrar errores léxicos con consejos
	for _, lexErr := range l.Errors() {
		fmt.Printf("%s✗ Error léxico: %s%s%s\n",
			colorRed, lexErr.Error(), lexErrHint(lexErr.Error()), colorReset)
	}

	if p.HasErrors() {
		r.printParseErrors(p.Errors())
		return
	}

	result := evaluator.Eval(program, r.env)

	if result != nil && result.Type() == object.ERROR_OBJ {
		fmt.Printf("%s%s✗ %s%s\n\n", colorBold, colorRed, result.Inspect(), colorReset)
		return
	}

	if result != nil && result.Type() != object.NULL_OBJ {
		fmt.Printf("%s→ %s%s\n\n", colorGreen, result.Inspect(), colorReset)
	}
}

// lexErrHint da un consejo contextual según el tipo de error léxico.
func lexErrHint(errMsg string) string {
	switch {
	case strings.Contains(errMsg, "string no cerrado"):
		return "\n  💡 Cierra el string con comillas dobles: \"tu texto aquí\""
	case strings.Contains(errMsg, "comentario de bloque"):
		return "\n  💡 Cierra el comentario de bloque con */"
	case strings.Contains(errMsg, "no reconocido"):
		return "\n  💡 NovaLang no reconoce ese carácter. Usa solo ASCII estándar."
	default:
		return ""
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// COMANDOS ESPECIALES
// ─────────────────────────────────────────────────────────────────────────────

func (r *REPL) handleCommand(cmd string) bool {
	switch cmd {
	case ".exit", ".salir", ".quit":
		fmt.Println(colorDim + "Hasta luego. ¡Sigue programando!" + colorReset)
		return true

	case ".help", ".ayuda":
		r.printHelp()

	case ".lexer":
		r.cfg.Mode = ModeLexer
		fmt.Println(colorYellow + "  Modo: LEXER (muestra tokens)" + colorReset)

	case ".parser":
		r.cfg.Mode = ModeParser
		fmt.Println(colorPurple + "  Modo: PARSER (muestra AST)" + colorReset)

	case ".eval":
		r.cfg.Mode = ModeEval
		r.env = object.NewEnvironment()
		fmt.Println(colorGreen + "  Modo: EVAL — entorno reiniciado" + colorReset)

	case ".reset":
		r.env = object.NewEnvironment()
		fmt.Println(colorYellow + "  Entorno reiniciado." + colorReset)

	case ".legend", ".leyenda":
		r.printLegend()

	case ".pos", ".position":
		r.cfg.ShowPos = !r.cfg.ShowPos
		if r.cfg.ShowPos {
			fmt.Println(colorGreen + "  Posición de tokens: ACTIVADA" + colorReset)
		} else {
			fmt.Println(colorYellow + "  Posición de tokens: DESACTIVADA" + colorReset)
		}

	case ".history", ".historial":
		r.printHistory()

	case ".clear", ".cls":
		fmt.Print("\033[H\033[2J")
		r.printBanner()

	default:
		fmt.Printf("%s  Comando desconocido: '%s'. Escribe .help para ayuda.%s\n",
			colorRed, cmd, colorReset)
	}
	return false
}

// ─────────────────────────────────────────────────────────────────────────────
// IMPRESIÓN DE TOKENS
// ─────────────────────────────────────────────────────────────────────────────

func (r *REPL) printToken(tok token.Token) {
	typeColor := r.tokenColor(tok.Type)
	typeStr := fmt.Sprintf("%-14s", string(tok.Type))
	literal := tok.Literal
	if tok.Type == token.STRING {
		literal = `"` + literal + `"`
	}

	if r.cfg.ShowPos {
		fmt.Printf("  %s%s%s  %-20s  %s(L:%d C:%d)%s\n",
			typeColor, typeStr, colorReset,
			colorWhite+colorBold+literal+colorReset,
			colorDim, tok.Pos.Line, tok.Pos.Column, colorReset)
	} else {
		fmt.Printf("  %s%s%s  %s%s%s\n",
			typeColor, typeStr, colorReset,
			colorWhite+colorBold, literal, colorReset)
	}
}

func (r *REPL) tokenColor(tt token.TokenType) string {
	if !r.cfg.ColorOutput {
		return ""
	}
	switch tt {
	case token.FUNCTION, token.LET, token.CONST, token.RETURN,
		token.IF, token.ELSEIF, token.ELSE,
		token.WHILE, token.FOR, token.IN,
		token.BREAK, token.CONTINUE,
		token.PRINT, token.IMPORT, token.NIL,
		token.AND, token.OR:
		return colorBold + colorBlue
	case token.INTEGER, token.FLOAT, token.STRING, token.TRUE, token.FALSE:
		return colorGreen
	case token.IDENTIFIER:
		return colorCyan
	case token.PLUS, token.MINUS, token.MULTIPLY, token.DIVISION,
		token.MOD, token.POW, token.EQ, token.NEQ, token.LT, token.LTE,
		token.GT, token.GTE, token.ASSIGN, token.PLUS_ASSIGN,
		token.MINUS_ASSIGN, token.NEGATION:
		return colorYellow
	case token.LPAREN, token.RPAREN, token.LBRACE, token.RBRACE,
		token.LBRACKET, token.RBRACKET, token.COMMA, token.SEMICOLON, token.COLON:
		return colorPurple
	default:
		return colorRed
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// UI / IMPRESIÓN
// ─────────────────────────────────────────────────────────────────────────────

func (r *REPL) printBanner() {
	fmt.Println()
	fmt.Println(colorBold + colorCyan + `  ╔════════════════════════════════════════════╗` + colorReset)
	fmt.Println(colorBold + colorCyan + `  ║         N O V A L A N G  v1.0             ║` + colorReset)
	fmt.Println(colorBold + colorCyan + `  ║   Intérprete Completo · Lexer+Parser+Eval  ║` + colorReset)
	fmt.Println(colorBold + colorCyan + `  ╚════════════════════════════════════════════╝` + colorReset)
	fmt.Println()
	fmt.Println(colorDim + `  Escribe código NovaLang y presiona Enter.` + colorReset)
	fmt.Println(colorDim + `  Usa .help para ver los comandos disponibles.` + colorReset)
	fmt.Println()
}

func (r *REPL) printLegend() {
	fmt.Println(colorBold + "  ── Leyenda de Colores (modo lexer) ─────────" + colorReset)
	fmt.Printf("  %s%-14s%s  Palabras reservadas\n", colorBold+colorBlue, "KEYWORD", colorReset)
	fmt.Printf("  %s%-14s%s  Literales (números, strings, booleans)\n", colorGreen, "LITERAL", colorReset)
	fmt.Printf("  %s%-14s%s  Identificadores\n", colorCyan, "IDENTIFIER", colorReset)
	fmt.Printf("  %s%-14s%s  Operadores\n", colorYellow, "OPERATOR", colorReset)
	fmt.Printf("  %s%-14s%s  Delimitadores\n", colorPurple, "DELIMITER", colorReset)
	fmt.Printf("  %s%-14s%s  Error léxico\n", colorRed, "ILLEGAL", colorReset)
	fmt.Println()
}

func (r *REPL) printHelp() {
	fmt.Println()
	fmt.Println(colorBold + "  ── Comandos del REPL ───────────────────────" + colorReset)
	cmds := [][2]string{
		{".eval", "Modo evaluador (ejecuta código) — defecto"},
		{".lexer", "Modo léxico (muestra tokens)"},
		{".parser", "Modo parser (muestra AST)"},
		{".reset", "Reinicia el entorno de ejecución"},
		{".pos", "Activa/desactiva posición de tokens"},
		{".legend", "Muestra la leyenda de colores"},
		{".history", "Muestra el historial de entradas"},
		{".clear", "Limpia la pantalla"},
		{".exit", "Sale del REPL"},
	}
	for _, c := range cmds {
		fmt.Printf("  %s%-22s%s  %s\n", colorYellow, c[0], colorReset, c[1])
	}
	fmt.Println()
	fmt.Println(colorBold + "  ── Sintaxis de NovaLang ─────────────────────" + colorReset)
	ejemplos := []struct{ desc, code string }{
		{"Número entero",    `42`},
		{"Número decimal",   `3.14`},
		{"String",           `"hola mundo"`},
		{"Booleano",         `true`},
		{"Variable",         `let x = 42;`},
		{"Print",            `print("hola");`},
		{"Aritmética",       `2 + 3 * 4`},
		{"Función",          `let suma = function(a, b) { return a + b; };`},
		{"Llamada",          `suma(10, 20);`},
		{"Condicional",      `if (x > 10) { print("mayor"); } else { print("menor"); }`},
		{"Bucle while",      `let i = 0; while (i < 5) { print(i); let i = i + 1; }`},
		{"Bucle for",        `for (let i = 0; i < 5; let i = i + 1) { print(i); }`},
	}
	for _, e := range ejemplos {
		fmt.Printf("  %s%-18s%s  %s%s%s\n",
			colorDim, e.desc, colorReset,
			colorWhite, e.code, colorReset)
	}
	fmt.Println()
	fmt.Println(colorBold + "  ── Errores comunes ──────────────────────────" + colorReset)
	errores := [][2]string{
		{`hola`, `✗ variable no definida — escribe: "hola" (con comillas)`},
		{`"texto`, `✗ string no cerrado — falta la comilla de cierre: "texto"`},
		{`hola?`, `✗ carácter ilegal '?' — NovaLang no reconoce ese símbolo`},
	}
	for _, e := range errores {
		fmt.Printf("  %s%-12s%s → %s\n", colorRed, e[0], colorReset, e[1])
	}
	fmt.Println()
}

func (r *REPL) printHistory() {
	if len(r.history) == 0 {
		fmt.Println(colorDim + "  (historial vacío)" + colorReset)
		return
	}
	fmt.Println(colorBold + "  ── Historial ───────────────────────────────" + colorReset)
	for i, h := range r.history {
		fmt.Printf("  %s%3d%s  %s\n", colorDim, i+1, colorReset, h)
	}
	fmt.Println()
}

func (r *REPL) printParseErrors(errors []string) {
	fmt.Printf("%s%s── Errores de sintaxis ──────────────────────%s\n",
		colorBold, colorRed, colorReset)
	for _, e := range errors {
		fmt.Printf("  %s✗ %s%s\n", colorRed, e, colorReset)
	}
	// Pista extra para el error más común: ILLEGAL
	for _, e := range errors {
		if strings.Contains(e, "ILLEGAL") {
			fmt.Printf("  %s💡 ¿Escribiste un carácter como ?, @, # o una comilla sin cerrar?%s\n",
				colorYellow, colorReset)
			break
		}
		if strings.Contains(e, "función de parseo") {
			fmt.Printf("  %s💡 ¿Olvidaste las comillas? Para texto usa: \"tu texto\"%s\n",
				colorYellow, colorReset)
			break
		}
	}
	fmt.Println()
}

func (r *REPL) printStats(total, illegal, errCount int, elapsed time.Duration) {
	fmt.Println()
	status := colorGreen + "✓ OK" + colorReset
	if errCount > 0 || illegal > 0 {
		status = fmt.Sprintf("%s✗ %d error(es)%s", colorRed, errCount+illegal, colorReset)
	}
	fmt.Printf("  %s  —  %s%d token(s)%s  —  %s%v%s\n\n",
		status,
		colorDim, total, colorReset,
		colorDim, elapsed.Round(time.Microsecond), colorReset)
}
