# 🌟 NovaLang

**NovaLang** es un lenguaje de scripting con intérprete completo implementado en **Go**.

Incluye todas las fases de un intérprete real:
**Lexer → Parser → AST → Evaluador**

---

## 📁 Estructura del proyecto

```
novalang/
├── cmd/novalang/
│   └── main.go                  ← CLI: punto de entrada
├── internal/
│   ├── token/token.go           ← Tipos de token + Position
│   ├── lexer/
│   │   ├── lexer.go             ← Analizador léxico
│   │   └── lexer_test.go        ← 27 tests del lexer
│   ├── ast/ast.go               ← Todos los nodos del AST
│   ├── parser/parser.go         ← Pratt Parser
│   ├── object/object.go         ← Sistema de objetos + Environment
│   ├── evaluator/
│   │   ├── evaluator.go         ← Tree-walking interpreter
│   │   └── evaluator_test.go    ← 28 tests del evaluador
│   └── repl/repl.go             ← REPL con colores + 3 modos
├── vscode-extension/
│   ├── package.json             ← Manifiesto de la extensión
│   ├── language-configuration.json
│   ├── syntaxes/novalang.tmLanguage.json  ← Resaltado de sintaxis
│   ├── snippets/novalang.json   ← 14 snippets
│   └── themes/nova-dark-color-theme.json  ← Tema oscuro
├── ejemplo.nv                   ← Programa de demostración
├── go.mod
└── README.md
```

---

## 🚀 Instalación

### Prerrequisitos
- **Go 1.21+** → https://go.dev/dl/

### Compilar en Windows
```powershell
# Dentro de la carpeta novalang (donde está go.mod)
go build -o nova.exe ./cmd/novalang

# Verificar
.\nova.exe -version
```

### Compilar en Linux/Mac
```bash
go build -o nova ./cmd/novalang
./nova -version
```

---

## 🎮 Modos de uso

```powershell
# REPL interactivo (ejecuta código directamente)
.\nova.exe

# Ejecutar un archivo .nv
.\nova.exe ejemplo.nv

# Solo ver tokens de una expresión
.\nova.exe -tokens "let x = 42 + 3.14;"

# Solo ver el AST de una expresión
.\nova.exe -ast "let x = factorial(5);"

# REPL en modo lexer (muestra tokens con colores)
.\nova.exe -lexer

# REPL en modo parser (muestra el AST)
.\nova.exe -parser

# Activar posición línea:columna en tokens
.\nova.exe -lexer -pos

# Sin colores (útil para pipelines)
.\nova.exe -no-color
```

---

## 🖥️ REPL Interactivo

El REPL tiene **3 modos** conmutables con comandos:

| Modo | Comando | Descripción |
|------|---------|-------------|
| Eval | `.eval` | Ejecuta el código (modo por defecto) |
| Lexer | `.lexer` | Muestra tokens con colores |
| Parser | `.parser` | Muestra el árbol sintáctico |

### Comandos del REPL

| Comando | Descripción |
|---------|-------------|
| `.eval` | Modo evaluador + reinicia entorno |
| `.lexer` | Modo lexer (tokens) |
| `.parser` | Modo parser (AST) |
| `.reset` | Reinicia variables del entorno |
| `.pos` | Activa/desactiva posición de tokens |
| `.legend` | Muestra leyenda de colores |
| `.history` | Historial de entradas |
| `.clear` | Limpia la pantalla |
| `.help` | Ayuda completa |
| `.exit` | Sale |

### Colores del modo lexer

| Color | Categoría |
|-------|-----------|
| 🔵 Azul bold | Keywords (`if`, `while`, `function`, `let`...) |
| 🟢 Verde | Literales (`42`, `3.14`, `"hola"`, `true`) |
| 🩵 Cyan | Identificadores (`x`, `miVar`) |
| 🟡 Amarillo | Operadores (`+`, `-`, `==`, `!=`, `+=`) |
| 🟣 Magenta | Delimitadores (`{`, `}`, `(`, `)`, `;`) |
| 🔴 Rojo | Error / ILLEGAL |

---

## 📐 El Lenguaje NovaLang

### Variables
```nova
let x = 42;
let nombre = "Juan";
let pi = 3.14159;
let activo = true;
let vacio = nil;
```

### Aritmética
```nova
let a = 2 + 3 * 4;    // 14  (respeta precedencia)
let b = 2 ^ 8;         // 256 (potencia)
let c = 10 / 4;        // 2.5 (siempre float)
let d = 17 % 5;        // 2   (módulo)
let e = (1 + 2) * 3;  // 9   (agrupación)
```

### Strings
```nova
let s = "hola" + " " + "mundo";  // concatenación
let con_escape = "línea1\nlínea2";
```

### Condicionales
```nova
if (x > 100) {
    print("grande");
} elseif (x > 10) {
    print("mediano");
} else {
    print("pequeño");
}
```

### Bucles
```nova
// While
let i = 0;
while (i < 10) {
    print(i);
    let i = i + 1;
}

// For
for (let i = 0; i < 5; let i = i + 1) {
    print(i);
}

// Break y continue
while (true) {
    if (condicion) { break; }
    if (otra) { continue; }
}
```

### Funciones y closures
```nova
// Función con nombre (soporta recursión)
let factorial = function(n) {
    if (n <= 1) { return 1; }
    return n * factorial(n - 1);
};

print(factorial(6));   // 720

// Closure (captura el entorno)
let makeAdder = function(x) {
    return function(y) { return x + y; };
};
let add5 = makeAdder(5);
print(add5(3));   // 8
```

### Comentarios
```nova
// Comentario de línea

/*
   Comentario de bloque
   múltiples líneas
*/
```

### Operadores completos

| Tipo | Operadores |
|------|-----------|
| Aritméticos | `+` `-` `*` `/` `%` `^` |
| Comparación | `==` `!=` `<` `<=` `>` `>=` |
| Lógicos | `and` `or` `!` |
| Asignación | `=` |
| Prefijos | `-x` `!x` |

---

## 🏗️ Arquitectura del intérprete

```
Código fuente (.nv)
        │
        ▼
  ┌─────────────┐
  │    Lexer    │  lee char a char → produce Tokens con posición (línea, col)
  └─────────────┘
        │  []Token
        ▼
  ┌─────────────┐
  │   Parser    │  Pratt Parser → construye el AST
  └─────────────┘   (Top-Down Operator Precedence)
        │  *ast.Program
        ▼
  ┌─────────────┐
  │  Evaluador  │  Tree-walking interpreter → produce Objects
  └─────────────┘
        │  object.Object
        ▼
     Resultado / Output en consola
```

### Características técnicas

| Característica | Detalle |
|---------------|---------|
| Lexer con `[]rune` | Soporte Unicode en identificadores |
| Rastreo de posición | Cada token tiene línea y columna |
| Recuperación de errores | El lexer emite `ILLEGAL` y continúa |
| Pratt Parser | Manejo correcto de precedencia de operadores |
| Closures reales | Las funciones capturan su entorno léxico |
| Recursión | Funciones pueden llamarse a sí mismas |
| Short-circuit | `and`/`or` no evalúan el segundo operando si no hace falta |
| Propagación de return | `ReturnValue` sube por el call-stack |
| Break/Continue | Señales que se propagan fuera del bloque de bucle |
| División siempre float | `10 / 4 = 2.5` (como Python 3) |
| Truthiness | `0`, `""`, `nil`, `false` son falsos; todo lo demás verdadero |

---

## 🧪 Tests

```powershell
# Todos los tests
go test ./internal/... -v

# Con cobertura
go test ./internal/... -cover

# Solo lexer
go test ./internal/lexer/... -v

# Solo evaluador
go test ./internal/evaluator/... -v
```

**Resumen: 55 tests en total**
- Lexer: 27 tests (cobertura 90.9%)
- Evaluador: 28 tests

---

## 🎨 Extensión VS Code

La carpeta `vscode-extension/` contiene una extensión completa lista para instalar.

### Características
- **Resaltado de sintaxis** con gramática TextMate completa
- **Tema oscuro "Nova Dark"** inspirado en GitHub Dark
- **14 snippets** para patrones comunes
- **Autocompletado de brackets** `{}` `[]` `()` `""`
- **Indentación automática** al abrir `{`
- **Detección de archivos** `.nv`

### Instalar la extensión

**Opción 1 — Copiar directamente (más simple):**
```powershell
# Windows
xcopy /E /I vscode-extension "%USERPROFILE%\.vscode\extensions\novalang-1.0.0"

# Linux / Mac
cp -r vscode-extension ~/.vscode/extensions/novalang-1.0.0
```
Luego reinicia VS Code. Cuando abras un `.nv` te pedirá usar la extensión.

**Opción 2 — Empaquetar con vsce:**
```bash
npm install -g @vscode/vsce
cd vscode-extension
vsce package
code --install-extension novalang-1.0.0.vsix
```

### Activar el tema Nova Dark
1. `Ctrl+Shift+P` → *Color Theme*
2. Selecciona **Nova Dark**

### Snippets disponibles

| Prefix | Genera |
|--------|--------|
| `let` | `let nombre = valor;` |
| `const` | `const NOMBRE = valor;` |
| `fn` | Función completa con return |
| `fna` | Función anónima |
| `if` | Bloque if |
| `ife` | Bloque if-else |
| `ifei` | Bloque if-elseif-else |
| `while` | Bucle while |
| `for` | Bucle for clásico |
| `pr` | `print(expresión);` |
| `ret` | `return valor;` |
| `factorial` | Función factorial recursiva |
| `fibonacci` | Función Fibonacci recursiva |
| `closure` | Closure / función de orden superior |

---

## 👤 Autores

Desarrollado en **Go** como proyecto académico de la materia Lenguajes y Compiladores — EIA.  
Juan Jose Jaramillo Mora y Dylan Alexander Mejia Ceballos
