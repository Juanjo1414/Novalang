// =============================================================================
// object/object.go — Sistema de Objetos en Tiempo de Ejecución de NovaLang
//
// Todo valor producido por el evaluador es un Object. Esto permite que
// el evaluador maneje todos los tipos de forma uniforme.
//
// Jerarquía:
//   Object (interfaz)
//   ├── Integer       ← 42
//   ├── Float         ← 3.14
//   ├── String        ← "hola"
//   ├── Boolean       ← true / false
//   ├── Null          ← nil / ausencia de valor
//   ├── ReturnValue   ← envuelve un valor durante 'return'
//   ├── Error         ← errores en tiempo de ejecución
//   └── Function      ← closure (función + entorno capturado)
//
// También define Environment: la tabla de variables del intérprete.
// =============================================================================

package object

import (
	"fmt"
	"strings"

	"github.com/novalang/novalang/internal/ast"
)

// ─────────────────────────────────────────────────────────────────────────────
// TIPOS DE OBJETO
// ─────────────────────────────────────────────────────────────────────────────

type ObjectType string

const (
	INTEGER_OBJ      ObjectType = "INTEGER"
	FLOAT_OBJ        ObjectType = "FLOAT"
	STRING_OBJ       ObjectType = "STRING"
	BOOLEAN_OBJ      ObjectType = "BOOLEAN"
	NULL_OBJ         ObjectType = "NULL"
	RETURN_VALUE_OBJ ObjectType = "RETURN_VALUE"
	ERROR_OBJ        ObjectType = "ERROR"
	FUNCTION_OBJ     ObjectType = "FUNCTION"
	BREAK_OBJ        ObjectType = "BREAK"    // señal interna
	CONTINUE_OBJ     ObjectType = "CONTINUE" // señal interna
)

// Object es la interfaz que implementan todos los valores del lenguaje.
type Object interface {
	Type() ObjectType
	Inspect() string // representación legible del valor
}

// ─────────────────────────────────────────────────────────────────────────────
// TIPOS DE DATO CONCRETOS
// ─────────────────────────────────────────────────────────────────────────────

// Integer representa un número entero: 42
type Integer struct {
	Value int64
}

func (i *Integer) Type() ObjectType { return INTEGER_OBJ }
func (i *Integer) Inspect() string  { return fmt.Sprintf("%d", i.Value) }

// Float representa un número decimal: 3.14
type Float struct {
	Value float64
}

func (f *Float) Type() ObjectType { return FLOAT_OBJ }
func (f *Float) Inspect() string  { return fmt.Sprintf("%g", f.Value) }

// String representa una cadena de texto: "hola"
type String struct {
	Value string
}

func (s *String) Type() ObjectType { return STRING_OBJ }
func (s *String) Inspect() string  { return s.Value }

// Boolean representa true o false.
// Usamos singletons (TRUE y FALSE) para eficiencia.
type Boolean struct {
	Value bool
}

func (b *Boolean) Type() ObjectType { return BOOLEAN_OBJ }
func (b *Boolean) Inspect() string {
	if b.Value {
		return "true"
	}
	return "false"
}

// Null representa la ausencia de valor (nil).
type Null struct{}

func (n *Null) Type() ObjectType { return NULL_OBJ }
func (n *Null) Inspect() string  { return "null" }

// ─────────────────────────────────────────────────────────────────────────────
// OBJETOS DE CONTROL DE FLUJO
// ─────────────────────────────────────────────────────────────────────────────

// ReturnValue envuelve un valor para propagarlo por el call-stack con 'return'.
//
// El evaluador de bloques detecta ReturnValue y detiene la ejecución.
// El evaluador de llamadas a función lo desenvuelve.
type ReturnValue struct {
	Value Object
}

func (rv *ReturnValue) Type() ObjectType { return RETURN_VALUE_OBJ }
func (rv *ReturnValue) Inspect() string  { return rv.Value.Inspect() }

// Error representa un error en tiempo de ejecución.
//
// Los errores se propagan hacia arriba automáticamente hasta el nivel más alto.
type Error struct {
	Message string
}

func (e *Error) Type() ObjectType { return ERROR_OBJ }
func (e *Error) Inspect() string  { return "[ERROR] " + e.Message }

// BreakSignal es una señal interna para propagar 'break' fuera de un bucle.
type BreakSignal struct{}

func (b *BreakSignal) Type() ObjectType { return BREAK_OBJ }
func (b *BreakSignal) Inspect() string  { return "break" }

// ContinueSignal es una señal interna para propagar 'continue'.
type ContinueSignal struct{}

func (c *ContinueSignal) Type() ObjectType { return CONTINUE_OBJ }
func (c *ContinueSignal) Inspect() string  { return "continue" }

// ─────────────────────────────────────────────────────────────────────────────
// FUNCIÓN (closure)
// ─────────────────────────────────────────────────────────────────────────────

// Function representa una función en tiempo de ejecución.
//
// Es un CLOSURE: captura el entorno en que fue definida (Env), lo que
// permite que las funciones accedan a variables del ámbito padre y que
// la recursión funcione correctamente.
type Function struct {
	Parameters []*ast.Identifier
	Body       *ast.BlockStatement
	Env        *Environment
	Name       string // nombre si fue asignada con let
}

func (f *Function) Type() ObjectType { return FUNCTION_OBJ }
func (f *Function) Inspect() string {
	params := make([]string, len(f.Parameters))
	for i, p := range f.Parameters {
		params[i] = p.Value
	}
	nameStr := ""
	if f.Name != "" {
		nameStr = " " + f.Name
	}
	return fmt.Sprintf("function%s(%s) { ... }", nameStr, strings.Join(params, ", "))
}

// ─────────────────────────────────────────────────────────────────────────────
// SINGLETONS GLOBALES
// ─────────────────────────────────────────────────────────────────────────────

// En lugar de crear nuevas instancias cada vez, reutilizamos estas.
var (
	TRUE  = &Boolean{Value: true}
	FALSE = &Boolean{Value: false}
	NULL  = &Null{}
	BREAK    = &BreakSignal{}
	CONTINUE = &ContinueSignal{}
)

// NativeBoolToBooleanObject convierte un bool de Go a los singletons TRUE/FALSE.
func NativeBoolToBooleanObject(input bool) *Boolean {
	if input {
		return TRUE
	}
	return FALSE
}

// ─────────────────────────────────────────────────────────────────────────────
// ENVIRONMENT (tabla de símbolos)
// ─────────────────────────────────────────────────────────────────────────────

// Environment es el entorno de ejecución: tabla nombre → valor.
//
// Los entornos pueden estar anidados: cuando una función busca una variable
// que no existe localmente, sube al entorno padre (Outer). Esto implementa
// el alcance léxico (lexical scoping) y los closures.
//
// Ejemplo de cadena para factorial(5):
//
//	global:   { factorial → Function }
//	          ↑ outer
//	llamada1: { n → Integer(5) }
//	          ↑ outer
//	llamada2: { n → Integer(4) }   ← llamada recursiva
type Environment struct {
	store map[string]Object
	outer *Environment
}

// NewEnvironment crea un entorno global (sin padre).
func NewEnvironment() *Environment {
	return &Environment{store: make(map[string]Object)}
}

// NewEnclosedEnvironment crea un entorno hijo enlazado al padre dado.
// Se usa al llamar una función.
func NewEnclosedEnvironment(outer *Environment) *Environment {
	env := NewEnvironment()
	env.outer = outer
	return env
}

// Get busca una variable por nombre (local → padre → abuelo → ...).
// Retorna (valor, encontrado).
func (e *Environment) Get(name string) (Object, bool) {
	obj, ok := e.store[name]
	if !ok && e.outer != nil {
		obj, ok = e.outer.Get(name)
	}
	return obj, ok
}

// Set asigna o actualiza una variable.
//
// Estrategia (igual que el profesor):
//   - Si la variable ya existe en algún nivel de la cadena → la actualiza allí.
//   - Si es nueva → la crea en el entorno local.
//
// Esto es esencial para que `let i = i + 1` dentro de un bucle actualice
// la `i` declarada en el entorno exterior.
func (e *Environment) Set(name string, val Object) Object {
	// Busca el entorno más cercano que ya tenga la variable
	env := e.findEnv(name)
	if env != nil {
		env.store[name] = val
	} else {
		e.store[name] = val
	}
	return val
}

// findEnv busca el entorno más cercano que contenga la variable.
// Retorna nil si no existe en ningún nivel.
func (e *Environment) findEnv(name string) *Environment {
	if _, ok := e.store[name]; ok {
		return e
	}
	if e.outer != nil {
		return e.outer.findEnv(name)
	}
	return nil
}
