// =============================================================================
// cmd/novalang/main.go — CLI de NovaLang (Intérprete Completo)
//
// Uso:
//   nova                    → REPL interactivo (eval)
//   nova archivo.nv         → ejecuta un archivo
//   nova -tokens "expr"     → muestra tokens de una expresión
//   nova -ast "expr"        → muestra el AST de una expresión
//   nova -lexer             → REPL en modo lexer
//   nova -parser            → REPL en modo parser
//   nova -pos               → activa posiciones en el REPL lexer
//   nova -version           → versión
// =============================================================================

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/novalang/novalang/internal/evaluator"
	"github.com/novalang/novalang/internal/lexer"
	"github.com/novalang/novalang/internal/object"
	"github.com/novalang/novalang/internal/parser"
	"github.com/novalang/novalang/internal/repl"
)

const version = "1.0.0"

func main() {
	showVersion := flag.Bool("version", false, "Muestra la versión de NovaLang")
	showPos     := flag.Bool("pos", false, "Muestra posición de tokens en el REPL")
	noColor     := flag.Bool("no-color", false, "Desactiva colores ANSI")
	noLegend    := flag.Bool("no-legend", false, "Omite la leyenda al iniciar")
	modeLexer   := flag.Bool("lexer", false, "Inicia el REPL en modo lexer")
	modeParser  := flag.Bool("parser", false, "Inicia el REPL en modo parser")
	tokensFlag  := flag.String("tokens", "", "Tokeniza una expresión inline")
	astFlag     := flag.String("ast", "", "Muestra el AST de una expresión")

	flag.Usage = printUsage
	flag.Parse()

	if *showVersion {
		fmt.Printf("NovaLang v%s — Intérprete Completo\n", version)
		fmt.Println("Lexer + Parser + Evaluador · github.com/novalang/novalang")
		os.Exit(0)
	}

	cfg := repl.DefaultConfig()
	cfg.ShowPos     = *showPos
	cfg.ColorOutput = !*noColor
	cfg.ShowLegend  = !*noLegend

	switch {
	case *modeLexer:
		cfg.Mode = repl.ModeLexer
	case *modeParser:
		cfg.Mode = repl.ModeParser
	default:
		cfg.Mode = repl.ModeEval
	}

	r := repl.New(cfg)

	// ── Modo: tokenizar expresión inline ─────────────────────────────────────
	if *tokensFlag != "" {
		r.ProcessInline(*tokensFlag)
		os.Exit(0)
	}

	// ── Modo: mostrar AST de expresión inline ─────────────────────────────────
	if *astFlag != "" {
		l := lexer.New(*astFlag)
		p := parser.New(l)
		program := p.ParseProgram()
		if p.HasErrors() {
			for _, e := range p.Errors() {
				fmt.Fprintln(os.Stderr, "  ✗", e)
			}
			os.Exit(1)
		}
		fmt.Println(program.String())
		os.Exit(0)
	}

	// ── Modo: ejecutar archivo ────────────────────────────────────────────────
	if flag.NArg() > 0 {
		path := flag.Arg(0)
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: no se pudo leer %q: %v\n", path, err)
			os.Exit(1)
		}

		l := lexer.New(string(data))
		p := parser.New(l)
		program := p.ParseProgram()

		if p.HasErrors() {
			fmt.Fprintln(os.Stderr, "══ Errores de sintaxis ══════════════════")
			for _, e := range p.Errors() {
				fmt.Fprintln(os.Stderr, "  ✗", e)
			}
			os.Exit(1)
		}

		env := object.NewEnvironment()
		result := evaluator.Eval(program, env)

		if result != nil && result.Type() == object.ERROR_OBJ {
			fmt.Fprintln(os.Stderr, "══ Error de ejecución ═══════════════════")
			fmt.Fprintln(os.Stderr, " ", result.Inspect())
			os.Exit(1)
		}
		os.Exit(0)
	}

	// ── Modo: REPL interactivo ────────────────────────────────────────────────
	r.Start()
}

func printUsage() {
	fmt.Println(`
  ╔══════════════════════════════════════════╗
  ║       NovaLang v1.0 — Uso de la CLI      ║
  ╚══════════════════════════════════════════╝

  nova                    → REPL interactivo (ejecuta código)
  nova archivo.nv         → ejecuta un archivo .nv
  nova -tokens "let x=5;" → tokeniza una expresión
  nova -ast "let x=5;"    → muestra el AST de una expresión
  nova -lexer             → REPL en modo lexer (tokens)
  nova -parser            → REPL en modo parser (AST)
  nova -pos               → muestra posiciones de tokens
  nova -no-color          → sin colores ANSI
  nova -version           → versión

Flags:`)
	flag.PrintDefaults()
	fmt.Println()
}
