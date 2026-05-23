// =============================================================================
// evaluator/evaluator_test.go — Tests del intérprete completo de NovaLang
// =============================================================================

package evaluator

import (
	"testing"

	"github.com/novalang/novalang/internal/lexer"
	"github.com/novalang/novalang/internal/object"
	"github.com/novalang/novalang/internal/parser"
)

// helper: evalúa un string de código y retorna el Object resultante
func testEval(input string) object.Object {
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	env := object.NewEnvironment()
	return Eval(program, env)
}

func testInteger(t *testing.T, obj object.Object, expected int64) {
	t.Helper()
	result, ok := obj.(*object.Integer)
	if !ok {
		t.Fatalf("se esperaba *object.Integer, se obtuvo %T (%s)", obj, obj.Inspect())
	}
	if result.Value != expected {
		t.Errorf("valor esperado=%d, obtenido=%d", expected, result.Value)
	}
}

func testFloat(t *testing.T, obj object.Object, expected float64) {
	t.Helper()
	result, ok := obj.(*object.Float)
	if !ok {
		t.Fatalf("se esperaba *object.Float, se obtuvo %T (%s)", obj, obj.Inspect())
	}
	if result.Value != expected {
		t.Errorf("valor esperado=%f, obtenido=%f", expected, result.Value)
	}
}

func testBool(t *testing.T, obj object.Object, expected bool) {
	t.Helper()
	result, ok := obj.(*object.Boolean)
	if !ok {
		t.Fatalf("se esperaba *object.Boolean, se obtuvo %T (%s)", obj, obj.Inspect())
	}
	if result.Value != expected {
		t.Errorf("valor esperado=%v, obtenido=%v", expected, result.Value)
	}
}

func testString(t *testing.T, obj object.Object, expected string) {
	t.Helper()
	result, ok := obj.(*object.String)
	if !ok {
		t.Fatalf("se esperaba *object.String, se obtuvo %T (%s)", obj, obj.Inspect())
	}
	if result.Value != expected {
		t.Errorf("valor esperado=%q, obtenido=%q", expected, result.Value)
	}
}

func testError(t *testing.T, obj object.Object, expectedMsg string) {
	t.Helper()
	errObj, ok := obj.(*object.Error)
	if !ok {
		t.Fatalf("se esperaba *object.Error, se obtuvo %T (%s)", obj, obj.Inspect())
	}
	if errObj.Message != expectedMsg {
		t.Errorf("mensaje esperado=%q, obtenido=%q", expectedMsg, errObj.Message)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// TESTS DE LITERALES
// ─────────────────────────────────────────────────────────────────────────────

func TestLiteralEntero(t *testing.T) {
	cases := []struct {
		input    string
		expected int64
	}{
		{"5;", 5},
		{"10;", 10},
		{"0;", 0},
		{"-42;", -42},
	}
	for _, c := range cases {
		testInteger(t, testEval(c.input), c.expected)
	}
}

func TestLiteralFlotante(t *testing.T) {
	cases := []struct {
		input    string
		expected float64
	}{
		{"3.14;", 3.14},
		{"-1.5;", -1.5},
		{"0.0;", 0.0},
	}
	for _, c := range cases {
		testFloat(t, testEval(c.input), c.expected)
	}
}

func TestLiteralString(t *testing.T) {
	result := testEval(`"hola mundo";`)
	testString(t, result, "hola mundo")
}

func TestLiteralBooleano(t *testing.T) {
	testBool(t, testEval("true;"), true)
	testBool(t, testEval("false;"), false)
}

// ─────────────────────────────────────────────────────────────────────────────
// TESTS DE OPERADORES ARITMÉTICOS
// ─────────────────────────────────────────────────────────────────────────────

func TestAritmetica(t *testing.T) {
	cases := []struct {
		input    string
		expected int64
	}{
		{"5 + 3;", 8},
		{"10 - 4;", 6},
		{"3 * 4;", 12},
		{"10 % 3;", 1},
		{"2 ^ 10;", 1024},
		{"(2 + 3) * 4;", 20},
		{"10 + 2 * 5;", 20}, // precedencia: 2*5=10, 10+10=20
	}
	for _, c := range cases {
		testInteger(t, testEval(c.input), c.expected)
	}
}

func TestDivisionProduceFloat(t *testing.T) {
	result := testEval("10 / 4;")
	testFloat(t, result, 2.5)
}

func TestAritmeticaFloat(t *testing.T) {
	result := testEval("1 + 2.5;")
	testFloat(t, result, 3.5)
}

// ─────────────────────────────────────────────────────────────────────────────
// TESTS DE COMPARACIÓN Y LÓGICA
// ─────────────────────────────────────────────────────────────────────────────

func TestComparacion(t *testing.T) {
	cases := []struct {
		input    string
		expected bool
	}{
		{"5 == 5;", true},
		{"5 != 3;", true},
		{"3 < 5;", true},
		{"5 <= 5;", true},
		{"5 > 3;", true},
		{"3 >= 3;", true},
		{"5 == 3;", false},
		{"5 < 3;", false},
		{"true == true;", true},
		{"true != false;", true},
	}
	for _, c := range cases {
		testBool(t, testEval(c.input), c.expected)
	}
}

func TestOperadoresLogicos(t *testing.T) {
	testBool(t, testEval("true and true;"), true)
	testBool(t, testEval("true and false;"), false)
	testBool(t, testEval("false or true;"), true)
	testBool(t, testEval("false or false;"), false)
	testBool(t, testEval("!false;"), true)
	testBool(t, testEval("!true;"), false)
}

// ─────────────────────────────────────────────────────────────────────────────
// TESTS DE VARIABLES
// ─────────────────────────────────────────────────────────────────────────────

func TestLetStatement(t *testing.T) {
	cases := []struct {
		input    string
		expected int64
	}{
		{"let x = 5; x;", 5},
		{"let x = 5; let y = 10; x + y;", 15},
		{"let x = 5; let y = x * 2; y;", 10},
	}
	for _, c := range cases {
		testInteger(t, testEval(c.input), c.expected)
	}
}

func TestVariableNoDefinida(t *testing.T) {
	result := testEval("z;")
	testError(t, result, `variable no definida: "z"`)
}

// ─────────────────────────────────────────────────────────────────────────────
// TESTS DE CONCATENACIÓN DE STRINGS
// ─────────────────────────────────────────────────────────────────────────────

func TestConcatenacion(t *testing.T) {
	result := testEval(`"hola" + " " + "mundo";`)
	testString(t, result, "hola mundo")
}

// ─────────────────────────────────────────────────────────────────────────────
// TESTS DE CONDICIONALES
// ─────────────────────────────────────────────────────────────────────────────

func TestIfElse(t *testing.T) {
	cases := []struct {
		input    string
		expected int64
	}{
		{"if (true) { 10; } else { 20; }", 10},
		{"if (false) { 10; } else { 20; }", 20},
		{"if (5 > 3) { 100; }", 100},
	}
	for _, c := range cases {
		testInteger(t, testEval(c.input), c.expected)
	}
}

func TestElseIf(t *testing.T) {
	input := `
		let x = 5;
		if (x > 10) {
			100;
		} elseif (x > 3) {
			50;
		} else {
			0;
		}
	`
	testInteger(t, testEval(input), 50)
}

// ─────────────────────────────────────────────────────────────────────────────
// TESTS DE FUNCIONES Y RETURN
// ─────────────────────────────────────────────────────────────────────────────

func TestFuncionSimple(t *testing.T) {
	input := `
		let suma = function(a, b) { return a + b; };
		suma(3, 7);
	`
	testInteger(t, testEval(input), 10)
}

func TestFuncionSinReturn(t *testing.T) {
	input := `
		let doble = function(x) { x * 2; };
		doble(5);
	`
	testInteger(t, testEval(input), 10)
}

func TestRecursion(t *testing.T) {
	input := `
		let factorial = function(n) {
			if (n <= 1) { return 1; }
			return n * factorial(n - 1);
		};
		factorial(5);
	`
	testInteger(t, testEval(input), 120)
}

func TestFibonacci(t *testing.T) {
	input := `
		let fib = function(n) {
			if (n <= 1) { return n; }
			return fib(n - 1) + fib(n - 2);
		};
		fib(10);
	`
	testInteger(t, testEval(input), 55)
}

func TestArgumentosIncorrectos(t *testing.T) {
	input := `
		let f = function(a, b) { return a + b; };
		f(1);
	`
	result := testEval(input)
	if result.Type() != object.ERROR_OBJ {
		t.Errorf("se esperaba un error por argumentos incorrectos, se obtuvo %s", result.Inspect())
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// TESTS DE CLOSURES
// ─────────────────────────────────────────────────────────────────────────────

func TestClosure(t *testing.T) {
	input := `
		let makeAdder = function(x) {
			return function(y) { return x + y; };
		};
		let add5 = makeAdder(5);
		add5(3);
	`
	testInteger(t, testEval(input), 8)
}

// ─────────────────────────────────────────────────────────────────────────────
// TESTS DE WHILE
// ─────────────────────────────────────────────────────────────────────────────

func TestWhile(t *testing.T) {
	input := `
		let suma = 0;
		let i = 1;
		while (i <= 10) {
			let suma = suma + i;
			let i = i + 1;
		}
		suma;
	`
	testInteger(t, testEval(input), 55)
}

func TestWhileBreak(t *testing.T) {
	input := `
		let x = 0;
		while (true) {
			let x = x + 1;
			if (x == 5) { break; }
		}
		x;
	`
	testInteger(t, testEval(input), 5)
}

func TestWhileContinue(t *testing.T) {
	// Suma solo números pares del 1 al 10
	input := `
		let suma = 0;
		let i = 0;
		while (i < 10) {
			let i = i + 1;
			if (i % 2 != 0) { continue; }
			let suma = suma + i;
		}
		suma;
	`
	testInteger(t, testEval(input), 30) // 2+4+6+8+10
}

// ─────────────────────────────────────────────────────────────────────────────
// TESTS DE FOR
// ─────────────────────────────────────────────────────────────────────────────

func TestFor(t *testing.T) {
	input := `
		let suma = 0;
		for (let i = 1; i <= 5; let i = i + 1) {
			let suma = suma + i;
		}
		suma;
	`
	testInteger(t, testEval(input), 15)
}

// ─────────────────────────────────────────────────────────────────────────────
// TESTS DE ERRORES EN TIEMPO DE EJECUCIÓN
// ─────────────────────────────────────────────────────────────────────────────

func TestDivisionPorCero(t *testing.T) {
	result := testEval("10 / 0;")
	testError(t, result, "división por cero")
}

func TestTiposIncompatibles(t *testing.T) {
	result := testEval(`5 + "hola";`)
	if result.Type() != object.ERROR_OBJ {
		t.Errorf("se esperaba error de tipos, se obtuvo %s", result.Inspect())
	}
}

func TestLlamarNoFuncion(t *testing.T) {
	result := testEval("let x = 5; x(1, 2);")
	if result.Type() != object.ERROR_OBJ {
		t.Errorf("se esperaba error al llamar no-función, se obtuvo %s", result.Inspect())
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// TEST DE PROGRAMA COMPLETO
// ─────────────────────────────────────────────────────────────────────────────

func TestProgramaCompleto(t *testing.T) {
	input := `
		let factorial = function(n) {
			if (n <= 1) { return 1; }
			return n * factorial(n - 1);
		};

		let suma_lista = function(n) {
			let total = 0;
			let i = 1;
			while (i <= n) {
				let total = total + i;
				let i = i + 1;
			}
			return total;
		};

		let fac = factorial(6);
		let sum = suma_lista(10);
		fac + sum;
	`
	// factorial(6) = 720, suma_lista(10) = 55, total = 775
	testInteger(t, testEval(input), 775)
}
