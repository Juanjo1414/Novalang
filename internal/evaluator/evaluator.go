// =============================================================================
// evaluator/evaluator.go — Evaluador del AST (Tree-Walking Interpreter)
//
// Recorre el AST nodo por nodo y ejecuta el programa produciendo Objects.
//
// Manejo de control de flujo:
//   - 'return'   → se propaga como ReturnValue hasta salir de la función.
//   - 'break'    → se propaga como BreakSignal hasta salir del bucle.
//   - 'continue' → se propaga como ContinueSignal al inicio del bucle.
//   - Errores    → se propagan hasta el nivel más alto sin seguir evaluando.
// =============================================================================

package evaluator

import (
	"fmt"
	"math"

	"github.com/novalang/novalang/internal/ast"
	"github.com/novalang/novalang/internal/object"
)

// ─────────────────────────────────────────────────────────────────────────────
// FUNCIÓN PRINCIPAL
// ─────────────────────────────────────────────────────────────────────────────

// Eval evalúa un nodo del AST en el entorno dado y retorna un Object.
//
// Es el punto de entrada recursivo: cada tipo de nodo delega a una función
// auxiliar específica. Los errores y señales de control se propagan hacia arriba.
func Eval(node ast.Node, env *object.Environment) object.Object {
	switch node := node.(type) {

	// ── Nodo raíz ──────────────────────────────────────────────────────────
	case *ast.Program:
		return evalProgram(node, env)

	// ── Bloques ────────────────────────────────────────────────────────────
	case *ast.BlockStatement:
		return evalBlockStatement(node, env)

	// ── Sentencias ─────────────────────────────────────────────────────────
	case *ast.ExpressionStatement:
		return Eval(node.Expression, env)

	case *ast.LetStatement:
		return evalLetStatement(node, env)

	case *ast.ReturnStatement:
		val := Eval(node.ReturnValue, env)
		if isError(val) {
			return val
		}
		return &object.ReturnValue{Value: val}

	case *ast.PrintStatement:
		val := Eval(node.Value, env)
		if isError(val) {
			return val
		}
		fmt.Println(val.Inspect())
		return object.NULL

	case *ast.WhileStatement:
		return evalWhileStatement(node, env)

	case *ast.ForStatement:
		return evalForStatement(node, env)

	case *ast.BreakStatement:
		return object.BREAK

	case *ast.ContinueStatement:
		return object.CONTINUE

	// ── Literales ──────────────────────────────────────────────────────────
	case *ast.IntegerLiteral:
		return &object.Integer{Value: node.Value}

	case *ast.FloatLiteral:
		return &object.Float{Value: node.Value}

	case *ast.StringLiteral:
		return &object.String{Value: node.Value}

	case *ast.BooleanLiteral:
		return object.NativeBoolToBooleanObject(node.Value)

	case *ast.NilLiteral:
		return object.NULL

	// ── Identificadores ────────────────────────────────────────────────────
	case *ast.Identifier:
		return evalIdentifier(node, env)

	// ── Expresiones ────────────────────────────────────────────────────────
	case *ast.PrefixExpression:
		right := Eval(node.Right, env)
		if isError(right) {
			return right
		}
		return evalPrefixExpression(node.Operator, right)

	case *ast.InfixExpression:
		left := Eval(node.Left, env)
		if isError(left) {
			return left
		}
		right := Eval(node.Right, env)
		if isError(right) {
			return right
		}
		return evalInfixExpression(node.Operator, left, right)

	case *ast.IfExpression:
		return evalIfExpression(node, env)

	case *ast.FunctionLiteral:
		fn := &object.Function{
			Parameters: node.Parameters,
			Body:       node.Body,
			Env:        env,
			Name:       node.Name,
		}
		// Registra la función en el entorno para que pueda llamarse recursivamente
		if node.Name != "" {
			env.Set(node.Name, fn)
		}
		return fn

	case *ast.CallExpression:
		return evalCallExpression(node, env)
	}

	return object.NULL
}

// ─────────────────────────────────────────────────────────────────────────────
// EVALUACIÓN DEL PROGRAMA Y BLOQUES
// ─────────────────────────────────────────────────────────────────────────────

// evalProgram evalúa todas las sentencias del programa.
// Si encuentra un ReturnValue lo desenvuelve (return en nivel global).
// Si encuentra un Error lo propaga inmediatamente.
func evalProgram(program *ast.Program, env *object.Environment) object.Object {
	var result object.Object

	for _, stmt := range program.Statements {
		result = Eval(stmt, env)

		switch result := result.(type) {
		case *object.ReturnValue:
			return result.Value // desenvuelve el return
		case *object.Error:
			return result
		}
	}

	return result
}

// evalBlockStatement evalúa un bloque { ... } SIN desenvolver ReturnValue.
// Propaga ReturnValue, Error, BreakSignal y ContinueSignal hacia arriba.
func evalBlockStatement(block *ast.BlockStatement, env *object.Environment) object.Object {
	var result object.Object

	for _, stmt := range block.Statements {
		result = Eval(stmt, env)

		if result != nil {
			rt := result.Type()
			if rt == object.RETURN_VALUE_OBJ || rt == object.ERROR_OBJ {
				return result // propaga sin desenvolver
			}
			if rt == object.BREAK_OBJ || rt == object.CONTINUE_OBJ {
				return result // propaga señales de bucle
			}
		}
	}

	return result
}

// ─────────────────────────────────────────────────────────────────────────────
// EVALUACIÓN DE SENTENCIAS
// ─────────────────────────────────────────────────────────────────────────────

func evalLetStatement(node *ast.LetStatement, env *object.Environment) object.Object {
	val := Eval(node.Value, env)
	if isError(val) {
		return val
	}
	env.Set(node.Name.Value, val)
	return object.NULL
}

func evalIdentifier(node *ast.Identifier, env *object.Environment) object.Object {
	if val, ok := env.Get(node.Value); ok {
		return val
	}
	return newError("variable no definida: %q", node.Value)
}

// evalWhileStatement evalúa: while (cond) { body }
func evalWhileStatement(node *ast.WhileStatement, env *object.Environment) object.Object {
	for {
		cond := Eval(node.Condition, env)
		if isError(cond) {
			return cond
		}
		if !isTruthy(cond) {
			break
		}

		result := evalBlockStatement(node.Body, env)
		if result != nil {
			switch result.Type() {
			case object.BREAK_OBJ:
				return object.NULL
			case object.CONTINUE_OBJ:
				continue
			case object.RETURN_VALUE_OBJ, object.ERROR_OBJ:
				return result
			}
		}
	}
	return object.NULL
}

// evalForStatement evalúa: for (init; cond; update) { body }
func evalForStatement(node *ast.ForStatement, env *object.Environment) object.Object {
	// El for tiene su propio entorno local para que 'let i' sea local al bucle
	forEnv := object.NewEnclosedEnvironment(env)

	// Inicialización (se ejecuta una sola vez)
	if node.Init != nil {
		if result := Eval(node.Init, forEnv); isError(result) {
			return result
		}
	}

	for {
		// Condición (si no hay → bucle infinito)
		if node.Condition != nil {
			cond := Eval(node.Condition, forEnv)
			if isError(cond) {
				return cond
			}
			if !isTruthy(cond) {
				break
			}
		}

		// Cuerpo
		result := evalBlockStatement(node.Body, forEnv)
		if result != nil {
			switch result.Type() {
			case object.BREAK_OBJ:
				return object.NULL
			case object.CONTINUE_OBJ:
				// Salta al update antes de re-evaluar la condición
			case object.RETURN_VALUE_OBJ, object.ERROR_OBJ:
				return result
			}
		}

		// Actualización
		if node.Update != nil {
			if upd := Eval(node.Update, forEnv); isError(upd) {
				return upd
			}
		}
	}

	return object.NULL
}

// ─────────────────────────────────────────────────────────────────────────────
// EVALUACIÓN DE EXPRESIONES
// ─────────────────────────────────────────────────────────────────────────────

// evalPrefixExpression evalúa: !expr  o  -expr
func evalPrefixExpression(operator string, right object.Object) object.Object {
	switch operator {
	case "!":
		return evalBangOperator(right)
	case "-":
		return evalMinusPrefixOperator(right)
	default:
		return newError("operador prefijo desconocido: %s", operator)
	}
}

// evalBangOperator evalúa: !expr
// Reglas de truthiness: !false → true, !null → true, !demás → false
func evalBangOperator(right object.Object) object.Object {
	switch right {
	case object.FALSE:
		return object.TRUE
	case object.NULL:
		return object.TRUE
	default:
		return object.FALSE
	}
}

// evalMinusPrefixOperator evalúa: -expr (solo para números)
func evalMinusPrefixOperator(right object.Object) object.Object {
	switch r := right.(type) {
	case *object.Integer:
		return &object.Integer{Value: -r.Value}
	case *object.Float:
		return &object.Float{Value: -r.Value}
	default:
		return newError("operador \"-\" no soportado para %s", right.Type())
	}
}

// evalInfixExpression evalúa: left OP right
func evalInfixExpression(operator string, left, right object.Object) object.Object {
	// Operadores lógicos con short-circuit (antes de verificar tipos)
	switch operator {
	case "and":
		if !isTruthy(left) {
			return object.FALSE
		}
		return object.NativeBoolToBooleanObject(isTruthy(right))
	case "or":
		if isTruthy(left) {
			return object.TRUE
		}
		return object.NativeBoolToBooleanObject(isTruthy(right))
	}

	switch {
	// Ambos enteros
	case left.Type() == object.INTEGER_OBJ && right.Type() == object.INTEGER_OBJ:
		return evalIntegerInfix(operator, left.(*object.Integer), right.(*object.Integer))

	// Al menos uno es float → promovemos ambos
	case isNumeric(left) && isNumeric(right):
		return evalFloatInfix(operator, toFloat(left), toFloat(right))

	// Strings
	case left.Type() == object.STRING_OBJ && right.Type() == object.STRING_OBJ:
		return evalStringInfix(operator, left.(*object.String), right.(*object.String))

	// Comparaciones de booleanos y null por identidad
	case operator == "==":
		return object.NativeBoolToBooleanObject(left == right)
	case operator == "!=":
		return object.NativeBoolToBooleanObject(left != right)

	// Tipos incompatibles
	case left.Type() != right.Type():
		return newError("tipos incompatibles: %s %s %s", left.Type(), operator, right.Type())

	default:
		return newError("operador %q no soportado entre %s y %s",
			operator, left.Type(), right.Type())
	}
}

func evalIntegerInfix(op string, left, right *object.Integer) object.Object {
	l, r := left.Value, right.Value
	switch op {
	case "+":
		return &object.Integer{Value: l + r}
	case "-":
		return &object.Integer{Value: l - r}
	case "*":
		return &object.Integer{Value: l * r}
	case "/":
		if r == 0 {
			return newError("división por cero")
		}
		return &object.Float{Value: float64(l) / float64(r)}
	case "%":
		if r == 0 {
			return newError("módulo por cero")
		}
		return &object.Integer{Value: l % r}
	case "^":
		result := math.Pow(float64(l), float64(r))
		if r >= 0 {
			return &object.Integer{Value: int64(result)}
		}
		return &object.Float{Value: result}
	case "<":
		return object.NativeBoolToBooleanObject(l < r)
	case "<=":
		return object.NativeBoolToBooleanObject(l <= r)
	case ">":
		return object.NativeBoolToBooleanObject(l > r)
	case ">=":
		return object.NativeBoolToBooleanObject(l >= r)
	case "==":
		return object.NativeBoolToBooleanObject(l == r)
	case "!=":
		return object.NativeBoolToBooleanObject(l != r)
	default:
		return newError("operador desconocido para enteros: %s", op)
	}
}

func evalFloatInfix(op string, left, right *object.Float) object.Object {
	l, r := left.Value, right.Value
	switch op {
	case "+":
		return &object.Float{Value: l + r}
	case "-":
		return &object.Float{Value: l - r}
	case "*":
		return &object.Float{Value: l * r}
	case "/":
		if r == 0.0 {
			return newError("división por cero")
		}
		return &object.Float{Value: l / r}
	case "%":
		return &object.Float{Value: math.Mod(l, r)}
	case "^":
		return &object.Float{Value: math.Pow(l, r)}
	case "<":
		return object.NativeBoolToBooleanObject(l < r)
	case "<=":
		return object.NativeBoolToBooleanObject(l <= r)
	case ">":
		return object.NativeBoolToBooleanObject(l > r)
	case ">=":
		return object.NativeBoolToBooleanObject(l >= r)
	case "==":
		return object.NativeBoolToBooleanObject(l == r)
	case "!=":
		return object.NativeBoolToBooleanObject(l != r)
	default:
		return newError("operador desconocido para flotantes: %s", op)
	}
}

func evalStringInfix(op string, left, right *object.String) object.Object {
	switch op {
	case "+":
		return &object.String{Value: left.Value + right.Value}
	case "==":
		return object.NativeBoolToBooleanObject(left.Value == right.Value)
	case "!=":
		return object.NativeBoolToBooleanObject(left.Value != right.Value)
	default:
		return newError("operador %q no soportado entre strings", op)
	}
}

// evalIfExpression evalúa: if (cond) { } [elseif (c) { }]* [else { }]
func evalIfExpression(node *ast.IfExpression, env *object.Environment) object.Object {
	cond := Eval(node.Condition, env)
	if isError(cond) {
		return cond
	}

	if isTruthy(cond) {
		return evalBlockStatement(node.Consequence, env)
	}

	// Ramas elseif
	for _, alt := range node.Alternatives {
		altCond := Eval(alt.Condition, env)
		if isError(altCond) {
			return altCond
		}
		if isTruthy(altCond) {
			return evalBlockStatement(alt.Body, env)
		}
	}

	// Rama else
	if node.ElseBlock != nil {
		return evalBlockStatement(node.ElseBlock, env)
	}

	return object.NULL
}

// evalCallExpression evalúa: func(args)
//
// Proceso:
//  1. Evalúa la expresión función.
//  2. Evalúa los argumentos.
//  3. Crea un entorno hijo del entorno de la función (closure).
//  4. Enlaza parámetros con argumentos.
//  5. Evalúa el cuerpo.
//  6. Desenvuelve el ReturnValue si existe.
func evalCallExpression(node *ast.CallExpression, env *object.Environment) object.Object {
	function := Eval(node.Function, env)
	if isError(function) {
		return function
	}

	fn, ok := function.(*object.Function)
	if !ok {
		return newError("%q no es una función (es %s)", node.Function, function.Type())
	}

	args := evalExpressions(node.Arguments, env)
	if len(args) == 1 && isError(args[0]) {
		return args[0]
	}

	extendedEnv, err := extendFunctionEnv(fn, args)
	if err != nil {
		return err
	}

	// Registra la función en su entorno para la recursión
	if fn.Name != "" {
		extendedEnv.Set(fn.Name, fn)
	}

	evaluated := evalBlockStatement(fn.Body, extendedEnv)

	// Desenvuelve el ReturnValue
	if rv, ok := evaluated.(*object.ReturnValue); ok {
		return rv.Value
	}

	if evaluated == nil {
		return object.NULL
	}
	return evaluated
}

// evalExpressions evalúa una lista de expresiones (argumentos de función).
// Si alguna produce error, retorna [error].
func evalExpressions(exprs []ast.Expression, env *object.Environment) []object.Object {
	var result []object.Object
	for _, e := range exprs {
		evaluated := Eval(e, env)
		if isError(evaluated) {
			return []object.Object{evaluated}
		}
		result = append(result, evaluated)
	}
	return result
}

// extendFunctionEnv crea el entorno de ejecución de una función.
// Enlaza parámetros formales con argumentos reales.
func extendFunctionEnv(fn *object.Function, args []object.Object) (*object.Environment, *object.Error) {
	env := object.NewEnclosedEnvironment(fn.Env)

	if len(args) != len(fn.Parameters) {
		return nil, &object.Error{
			Message: fmt.Sprintf(
				"número incorrecto de argumentos: se esperaban %d, se recibieron %d",
				len(fn.Parameters), len(args),
			),
		}
	}

	for i, param := range fn.Parameters {
		env.Set(param.Value, args[i])
	}

	return env, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// FUNCIONES AUXILIARES
// ─────────────────────────────────────────────────────────────────────────────

// isTruthy determina si un objeto es "verdadero" para condicionales y bucles.
//
// Reglas:
//   - null  → false
//   - false → false
//   - 0     → false
//   - 0.0   → false
//   - ""    → false
//   - todo lo demás → true
func isTruthy(obj object.Object) bool {
	switch o := obj.(type) {
	case *object.Null:
		return false
	case *object.Boolean:
		return o.Value
	case *object.Integer:
		return o.Value != 0
	case *object.Float:
		return o.Value != 0.0
	case *object.String:
		return o.Value != ""
	default:
		return true
	}
}

// isError reporta si un Object es un Error.
func isError(obj object.Object) bool {
	if obj == nil {
		return false
	}
	return obj.Type() == object.ERROR_OBJ
}

// newError crea un nuevo objeto Error con un mensaje formateado.
func newError(format string, a ...interface{}) *object.Error {
	return &object.Error{Message: fmt.Sprintf(format, a...)}
}

// isNumeric reporta si un Object es Integer o Float.
func isNumeric(obj object.Object) bool {
	return obj.Type() == object.INTEGER_OBJ || obj.Type() == object.FLOAT_OBJ
}

// toFloat convierte un Integer o Float a *object.Float.
func toFloat(obj object.Object) *object.Float {
	switch o := obj.(type) {
	case *object.Integer:
		return &object.Float{Value: float64(o.Value)}
	case *object.Float:
		return o
	}
	return &object.Float{Value: 0}
}
