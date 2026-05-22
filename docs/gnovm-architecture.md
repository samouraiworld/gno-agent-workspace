/!\ WIP /!\
---
 
 ┌──────────────────────────────────────────────────────────────────┐
 │                    .gno source text                             │
 └─────────────────────────┬────────────────────────────────────────┘
                           │
                           ▼
 ┌─────────────────────────────────────────────────────────────────┐
 │  STAGE 1: PARSING                                              │
 │                                                                 │
 │  go/parser  ──►  go2gno.go  ──►  Gno AST (FileNode)           │
 │                  ▲                                              │
 │                  │                                              │
 │                  nodes.go  (AST node type definitions)          │
 │                                                                 │
 └─────────────────────────┬───────────────────────────────────────┘
                           │
                           ▼
 ┌─────────────────────────────────────────────────────────────────┐
 │  STAGE 2: STATIC BLOCK INIT                                    │
 │                                                                 │
 │  preprocess.go ──► initStaticBlocks1  (loop var renaming)      │
 │                ──► initStaticBlocks2  (name reservation)       │
 │                                                                 │
 │  transcribe.go  (generic AST walker, ENTER/BLOCK/LEAVE)        │
 └─────────────────────────┬───────────────────────────────────────┘
                           │
                           ▼
 ┌─────────────────────────────────────────────────────────────────┐
 │  STAGE 3: PREDEFINITION                                        │
 │                                                                 │
 │  preprocess.go ──► PredefineFileSet                            │
 │    1. Imports   2. Types   3. Funcs   4. Values                │
 │                                                                 │
 │  uverse.go  (builtin types/funcs: make, append, len, etc.)    │
 └─────────────────────────┬───────────────────────────────────────┘
                           │
                           ▼
 ┌─────────────────────────────────────────────────────────────────┐
 │  STAGE 4: PREPROCESSING (type-check & transform)               │
 │                                                                 │
 │  preprocess.go ──► preprocess1()                               │
 │    • Name resolution (NameExpr → ValuePath)                    │
 │    • Constant folding (→ *ConstExpr)                           │
 │    • Type inference & checking                                 │
 │                                                                 │
 │  transcribe.go  (AST walk infrastructure)                      │
 │  type_check.go  (operator/operand compatibility rules)         │
 │  types.go       (Type interface & all type definitions)        │
 │                                                                 │
 └─────────────────────────┬───────────────────────────────────────┘
                           │
                           ▼
 ┌─────────────────────────────────────────────────────────────────┐
 │  STAGE 5: EXECUTION (stack-based VM)                           │
 │                                                                 │
 │  machine.go ──► Run() loop: pop Op, dispatch                   │
 │    │                                                            │
 │    ├── op_exec.go        (statement dispatch: for/if/switch)   │
 │    ├── op_eval.go        (expression eval: names, lits, calls) │
 │    ├── op_call.go        (function call/return machinery)      │
 │    ├── op_binary.go      (+ - * / == < && ||)                  │
 │    ├── op_unary.go       (+ - ! ^)                             │
 │    ├── op_types.go       (type construction at runtime)        │
 │    ├── op_assign.go      (= := += etc.)                        │
 │    ├── op_expressions.go (index, slice, selector, star, ref)   │
 │    ├── op_decl.go        (var/type declarations)               │
 │    └── op_inc_dec.go     (++ --)                               │
 │                                                                 │
 │  values.go         (runtime value types: TypedValue, etc.)     │
 │  values_string.go  (Sprint/String for values)                  │
 │  uverse.go         (builtin function bodies: make, append...)  │
 │  realm.go          (persistent state for r/ packages)          │
 │                                                                 │
 └─────────────────────────────────────────────────────────────────┘
 ┌─────────────────────────────────────────────────────────────────┐
 │  CROSS-CUTTING / INFRASTRUCTURE                                │
 │                                                                 │
 │  nodes.go       AST node definitions        (Stages 1-5)      │
 │  types.go       Type system definitions     (Stages 3-5)      │
 │  misc.go        Token→Op mapping utilities  (Stages 4-5)      │
 │  package.go     Amino serialization reg.    (Stages 3-5)      │
 │  transcribe.go  AST walker framework        (Stages 2-4)      │
 └─────────────────────────────────────────────────────────────────┘
