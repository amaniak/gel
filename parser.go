package gel

import (
  "fmt"
  "strings"
  "regexp"
  "bytes"
  "github.com/fatih/color"
  "github.com/amaniak/gel/token"
)

const(
  DEBUG = true
)

var trace = func(pos string, msg string) {
  colorize := color.New(color.FgYellow)
  if(DEBUG) {
    colorize.Println(pos + ": " + msg)
  }
}

func stripchars(str, chr string) string {
  return strings.Map(func(r rune) rune {
      if strings.IndexRune(chr, r) < 0 {
          return r
      }
      return -1
  }, str)
}

type Parser struct {
  CurrentNamespace string
  CurrentState token.Token
  Scanner *Scanner
}

func SQLParser(path string) *Parser {
  scanner := &Scanner{Path: path}
  parser := &Parser{CurrentState: token.START, Scanner: scanner}
  return parser
}

func (p *Parser) State(s token.Token) bool {
  return p.CurrentState == s
}

func (p *Parser) Ast() []*Ast {
  return p.Scanner.Buffer
}

func (p *Parser) functionName(node *Ast) string {
  trim := strings.Trim(p.CurrentNamespace, " ")
  return fmt.Sprintf("function %s.", trim)
}

func (p *Parser) CreateSchemaDefinition(node *Ast) {

  if (len(p.CurrentNamespace) < 0 || p.CurrentNamespace != node.FileInfo.Namespace) {
    p.CurrentNamespace = node.FileInfo.Namespace

    // Create buffer
    var buffer bytes.Buffer

    dropSchemaMacro := fmt.Sprintf("drop schema if exists %s cascade;", p.CurrentNamespace)
    createSchemaMacro := fmt.Sprintf("create schema %s;", p.CurrentNamespace)

    // Build macro
    buffer.WriteString(dropSchemaMacro)
    buffer.WriteString("\n")
    buffer.WriteString(createSchemaMacro)
    buffer.WriteString("\n")

    node.Source = buffer.String()
  }
}

func (p *Parser) OpenImFuncKeyWord(node *Ast) {

  functionName := p.functionName(node)
  macro := strings.Replace(node.Text(), token.IMFUNC.String() + " ", functionName , -1)

  // Create buffer
  var buffer bytes.Buffer

  // Build macro
  buffer.WriteString("CREATE OR REPLACE ")
  buffer.WriteString(macro)
  buffer.WriteString("\n")
  buffer.WriteString("LANGUAGE sql AS $$")

  // Set state
  p.CurrentState = token.IMFUNC

  // update node
  node.Source = buffer.String()
}

func (p *Parser) OpenFuncKeyWord(node *Ast) {

  functionName := p.functionName(node)
  macro := strings.Replace(node.Text(), token.FUNC.String() + " ", functionName , -1)

  // Create buffer
  var buffer bytes.Buffer

  // Build macro
  buffer.WriteString("CREATE OR REPLACE ")
  buffer.WriteString(macro)
  buffer.WriteString("\n")
  buffer.WriteString("LANGUAGE sql AS $$")

  // Set state
  p.CurrentState = token.FUNC

  // update node
  node.Source = buffer.String()
}

func (p *Parser) OpenImProcKeyWord(node *Ast) {

  functionName := p.functionName(node)
  macro := strings.Replace(node.Text(), token.IMPROC.String() + " ", functionName , -1)

  // Create buffer
  var buffer bytes.Buffer

  // Build macro
  buffer.WriteString("CREATE OR REPLACE ")
  buffer.WriteString(macro)
  buffer.WriteString("\n")
  buffer.WriteString(" LANGUAGE plpgsql AS $$ ")
  buffer.WriteString("\n")
  buffer.WriteString("DECLARE")

  // Set state
  p.CurrentState = token.IMPROC

  // update node
  node.Source = buffer.String()
}

func (p *Parser) OpenProcKeyWord(node *Ast) {

  functionName := p.functionName(node)
  macro := strings.Replace(node.Text(), token.PROC.String() + " ", functionName , -1)

  // Create buffer
  var buffer bytes.Buffer

  // Build macro
  buffer.WriteString("CREATE OR REPLACE ")
  buffer.WriteString(macro)
  buffer.WriteString("\n")
  buffer.WriteString(" LANGUAGE plpgsql AS $$ ")
  buffer.WriteString("\n")
  buffer.WriteString("DECLARE")


  // Set state
  p.CurrentState = token.PROC

  // update node
  node.Source = buffer.String()
}

func (p *Parser) MacroScopeExpand(node *Ast) {
  cleaned := strings.Replace(node.Text(), "this.", node.FileInfo.Namespace + ".", -1)
  node.Source = strings.Replace(cleaned,  "getv(", "vars.getv(", -1)
}


func (p *Parser) MacroExpandDefault(node *Ast) {
  macro := fmt.Sprintf("%s -- %s:%s", node.Text(), node.FileInfo.Namespace, node.Line())
  trace(node.Line(), macro)
  node.Source = macro
}

func (p *Parser) MacroExpandExec(node *Ast) {
  macro := strings.Replace(node.Text(), "exec!", "EXECUTE ", -1)
  trace(node.Line(), macro)
  node.Source = macro
}

func (p *Parser) MacroExpandQuery(node *Ast) {

  rp  := regexp.MustCompile("{{(.*?)}}")
  rpq := regexp.MustCompile("#{{(.*?)}}")

  statement  := strings.Replace(node.Text(), "|> ", "", -1)
  vars       := rp.FindAllString(statement, -1)
  qvars      := rpq.FindAllString(statement, -1)

  for _,qliteral := range qvars {
    clean := stripchars(qliteral, "#{}")
    statement = strings.Replace(statement, qliteral, "'''||" + clean + "||'''", 1)
  }

  for _,literal := range vars {
    clean := stripchars(literal, "{}")
    statement = strings.Replace(statement, literal, "'||" + clean + "||'", 1)
  }

  // Append a space
  statement = strings.Replace(statement, "'", "' ", 1)

  //Create buffer
  var buffer bytes.Buffer

  buffer.WriteString(statement)

  // source
  node.Source = buffer.String()
}

func (p *Parser) CloseFuncKeyWord(node *Ast, state string) {

  //Create buffer
  var buffer bytes.Buffer

  buffer.WriteString("$$ " + state + ";")
  buffer.WriteString(" ")
  buffer.WriteString("\n")

  // source
  node.Source = buffer.String()

}

//
func (p *Parser) CloseExecKeyWord(node *Ast) {
  //Create buffer
  var buffer bytes.Buffer

  buffer.WriteString(node.Text())
  buffer.WriteString(" ")
  buffer.WriteString("\n")

  // source
  node.Source = buffer.String()
}

// TODO: closing statement needs IMMUTABLE

func (p *Parser) CloseProcKeyWord(node *Ast, state string) {

  //Create buffer
  var buffer bytes.Buffer

  buffer.WriteString("END; $$ " + state + ";")
  buffer.WriteString(" ")
  buffer.WriteString("\n")

  // source
  node.Source = buffer.String()
}


func (p *Parser) CloseStatment(node *Ast) {

  switch  {
    case p.State(token.FUNC):
      p.CloseFuncKeyWord(node, "IMMUTABLE");
    case p.State(token.IMFUNC):
      p.CloseFuncKeyWord(node, "");
    case p.State(token.PROC):
      p.CloseProcKeyWord(node, "IMMUTABLE");
    case p.State(token.IMPROC):
      p.CloseProcKeyWord(node, "");
  }
}

func(p *Parser) Parse() {

  p.Compile()

  var buffer bytes.Buffer
  for _, node := range p.Ast() {
    buffer.WriteString(node.Source)
    buffer.WriteString("\n")
  }

  trace("Result", "Result:")
  fmt.Println(buffer.String())
}


func (p *Parser) Compile() {

  fmt.Println("Started parser", p.CurrentState, token.START)

  // Load sources into scanner
  p.Scanner.Load()

  // Loop AST -> parse to valid SQL
  for _, node := range p.Ast() {

    // create schema
    p.CreateSchemaDefinition(node)

    switch {

    // skip empty lines
    case node.IsEmpty():
      trace(node.Line(), ".")
      continue

    // check whitespace and state<start> both false
    case !node.IsWhiteSpace() && !p.State(token.START):
      trace(node.Line(), "......")
      p.CloseStatment(node)
      p.CurrentState = token.START

    // Tokenize immutable function
    case p.State(token.START) && node.IsImFunc():
      p.OpenImFuncKeyWord(node)
      trace(node.Line(), "immutable func!")

    // Tokenize function
    case p.State(token.START) && node.IsFunc():
      p.OpenFuncKeyWord(node)
      trace(node.Line(), "function func")


    // Tokenize immutable procedure
    case p.State(token.START) && node.IsImProc():
      p.OpenImProcKeyWord(node)
      trace(node.Line(), "function proc!")

    // Tokenize immutable procedure
    case p.State(token.START) && node.IsProc():
      p.OpenProcKeyWord(node)
      trace(node.Line(), "function proc")

    // Tokenize executor
    case p.State(token.IMPROC) && node.IsExec():
      p.MacroExpandExec(node)
      trace(node.Line(), "function exec!")

    case p.State(token.IMPROC) && node.IsQuery():
      p.MacroExpandQuery(node)
      trace(node.Line(), "macro |>")

    // Last statement
    case !node.IsNewLine():
      trace(node.Line(), "... SQL")

      slashed, _:= regexp.MatchString("\\", node.Text())
      statement, _:= regexp.MatchString("$SQL", node.Text())

      if(slashed || statement){
        p.MacroScopeExpand(node)
        trace(node.Line(), "function macro expanded")
      } else {
        p.MacroExpandDefault(node)
        trace(node.Line(), "default macro\n")
      }

    }
  }

  fmt.Println("ATTTTTT  CLOSE >>>>>>>>>>>> ", p.State(token.START), p.CurrentState)

  if(!p.State(token.START)) {

    node := EmptyAstNode()

    p.CloseStatment(node)

    p.Scanner.Buffer = append(p.Scanner.Buffer, node)

    trace("-999", "EOF")
  }
}






//
// var parserErrorHint = color.New(color.FgRed).PrintfFunc()
// var macroHint  = color.New(color.FgBlue).PrintfFunc()
// var rapport = func(lineNumber int) {
//
//   green := color.New(color.FgGreen)
//   boldGreen := green.Add(color.Bold)
//
//   boldGreen.Println("\n-- Rapport --")
//   boldGreen.Println(" * Number of lines:", lineNumber)
//   boldGreen.Println("\n")
//
// }
//
// func stripchars(str, chr string) string {
//     return strings.Map(func(r rune) rune {
//         if strings.IndexRune(chr, r) < 0 {
//             return r
//         }
//         return -1
//     }, str)
// }
//
// // basic SQL file parser
//
// type SQLParser struct {
//
//   // read files from this disk-based location
//   Folder string
//
//   // FSM
//   State string
//
//   // Current namespace
//   Namespace string
//
//   // preprocessed result
//   Buffer bytes.Buffer
//
//   //  hooks into different writable mode
//   Writer io.Writer
//
// }
//
// func(p * SQLParser) addNewLine() {
//   p.Buffer.WriteString("\n")
// }
//
// func (p *SQLParser) open_proc(fn string) {
//
//   p.Buffer.WriteString("CREATE OR REPLACE ")
//   p.addNewLine()
//   p.Buffer.WriteString(fn)
//   p.addNewLine()
//   p.Buffer.WriteString(" LANGUAGE plpgsql AS $$ ")
//   p.addNewLine()
//   p.Buffer.WriteString("DECLARE ")
//   p.addNewLine()
// }
//
// func (p *SQLParser) open_fn(fn string) {
//   p.Buffer.WriteString("CREATE OR REPLACE")
//   p.addNewLine()
//   p.Buffer.WriteString(fn)
//   p.addNewLine()
//   p.Buffer.WriteString("LANGUAGE sql AS $$")
//   p.addNewLine()
// }
//
// func (p *SQLParser) open_jsfn(fn string) {
//   p.Buffer.WriteString("CREATE OR REPLACE")
//   p.addNewLine()
//   p.Buffer.WriteString(fn)
//   p.addNewLine()
//   p.Buffer.WriteString("LANGUAGE plv8 AS $$")
//   p.addNewLine()
// }
//
// func (p *SQLParser) open_exec_f(fn string) {
//   p.Buffer.WriteString(fn)
//   p.addNewLine()
// }
//
// func (p *SQLParser) open_expect_raise(fn string) {
//   fmt.Println("Not implemented yet")
// }
//
// func (p *SQLParser) macroexpand(fileName string, fn string, linenumber int)  string {
//   cleaned := strings.Replace(fn, "this.", p.Namespace + ".", -1)
//   return strings.Replace(cleaned, "getv(", "vars.getv(", -1)
// }
//
// func (p *SQLParser) close_fn(fn string) {
//   p.Buffer.WriteString("$$ " + fn + ";")
//   p.Buffer.WriteString(" ")
//   p.addNewLine()
// }
//
// func (p *SQLParser) close_exec(fn string) {
//   p.addNewLine()
//   p.Buffer.WriteString(fn)
//   p.addNewLine()
// }
//
// func (p *SQLParser) close_proc(fn string) {
//   p.Buffer.WriteString("  END; $$ " + fn + ";")
//   p.Buffer.WriteString(" ")
//   p.addNewLine()
// }
//
// func (p *SQLParser) close_stmt(fn string) {
//
//   if(p.State == "fn") {
//       p.close_fn("IMMUTABLE")
//   } else if(p.State ==  "fn!") {
//       p.close_fn("")
//   } else if(p.State ==  "pr") {
//       p.close_proc("IMMUTABLE")
//   } else if(p.State ==  "pr!") {
//       p.close_proc("")
//   } else if(p.State ==  "jsfn") {
//       p.close_proc("")
//   }
// }
//
// // Processor
// func (p *SQLParser) ProcessFile(fileName string) {
//
//   // lineNumber counter
//   var lineNumber int = 0
//
//   // Set namespace
//   p.Namespace = strings.TrimSuffix(fileName, filepath.Ext(fileName))
//
//   // Default macro's
//   dropSchemaMacro := fmt.Sprintf("drop schema if exists %s cascade;", p.Namespace)
//   createSchemaMacro := fmt.Sprintf("create schema %s;", p.Namespace)
//
//   // Write lines to bufffer
//   p.Buffer.WriteString(dropSchemaMacro)
//   p.addNewLine()
//
//   p.Buffer.WriteString(createSchemaMacro)
//   p.addNewLine()
//   p.addNewLine()
//
//   //
//   file, err := os.Open(fileName)
//   if err != nil {
//       log.Fatal(err)
//   }
//   defer file.Close()
//
//   scanner := bufio.NewScanner(file)
//
//   for scanner.Scan() {
//
//     // bump line
//     lineNumber = lineNumber + 1
//
//     // Skip empty lines
//     line := bytes.TrimSpace(scanner.Bytes())
// 		if len(line) == 0 {
// 			continue
// 		}
//
//     // check whitespace chars and send close statement
//     reg, _:= regexp.Compile(`^\s`)
//     whitespace := reg.MatchString(scanner.Text())
//
//     if(p.State != "start" && !whitespace) {
//       macroHint(" > ... \n")
//       fmt.Println(scanner.Text())
//
//       p.close_stmt(scanner.Text())
//       p.State = "start"
//     }
//
//     // syntax -> keywords regex's
//     proc_f, _ := regexp.MatchString("proc!", scanner.Text())
//     proc, _ := regexp.MatchString("proc", scanner.Text())
//     func_f, _:= regexp.MatchString("func!", scanner.Text())
//     func_d, _:= regexp.MatchString("func", scanner.Text())
//     jsfn, _:= regexp.MatchString("jsfn", scanner.Text())
//
//     // macro's
//     exec_f,  _:= regexp.MatchString("exec!", scanner.Text())
//     //query_f, _:= regexp.MatchString("query!", scanner.Text())
//     query_f, _:= regexp.MatchString("|>", scanner.Text())
//
//     // visitors
//
//     fname := fmt.Sprintf("function %s.", strings.Trim(p.Namespace, " "))

//
//     if(p.State == "start" && proc_f) {
//       newLine := strings.Replace(scanner.Text(), "proc! ", fname, -1)
//       p.open_proc(newLine)
//       p.State = "pr!"
//       fmt.Println("proc! found", lineNumber, fname)
//
//
//     } else if(p.State == "start" && proc ) {
//
//       text := strings.Trim(scanner.Text(), " ")
//
//       newLine := strings.Replace(text, "proc ", fname, -1)
//       p.open_proc(newLine)
//       p.State = "pr"
//       fmt.Println("proc found", lineNumber)
//
//     } else if(p.State == "start" && func_f ) {
//
//       text := strings.Trim(scanner.Text(), " ")
//
//       newLine := strings.Replace(text, "func! ", fname, -1)
//       p.open_fn(newLine)
//       p.State = "fn!"
//       fmt.Println("func! found", lineNumber)
//
//
//     } else if(p.State == "start" && func_d) {
//
//       text := strings.Trim(scanner.Text(), " ")
//       newLine := strings.Replace(text, "func ", fname, -1)
//
//       fmt.Println(newLine)
//
//       p.open_fn(newLine)
//       p.State = "fn"
//       fmt.Println("func found", lineNumber)
//
//     } else if(p.State == "start" && jsfn) {
//       newLine := strings.Replace(scanner.Text(), "jsfn ", fname, -1)
//       p.open_jsfn(newLine)
//       p.State = "jsfn"
//       fmt.Println("jsfn found", lineNumber)
//
//     } else if(scanner.Text() != "\n") {
//
//       slashed, _:= regexp.MatchString("\\", scanner.Text())
//       statement, _:= regexp.MatchString("$SQL", scanner.Text())
//
//       if(slashed || statement){
//         p.Buffer.WriteString(p.macroexpand(fileName, scanner.Text(), lineNumber))
//         p.addNewLine()
//
//
//       } else if(p.State == "jsfn") {
//
//         macro := fmt.Sprintf("%s // %s:%d", p.macroexpand(fileName, scanner.Text(), lineNumber), p.Namespace, lineNumber)
//         p.Buffer.WriteString(macro)
//         p.addNewLine()
//
//         macroHint("macro jsfn found")
//
//       } else if(p.State == "pr!" && exec_f) {
//
//         macroHint("macro exec! found\n")
//         fmt.Println(p.State)
//         newLine := strings.Replace(scanner.Text(), "exec!", "EXECUTE ", -1)
//         p.open_exec_f(newLine)
//
//
//       } else if(p.State == "pr!" && query_f) {
//         newLine := strings.Replace(scanner.Text(), "|> ", "", -1)
//
//         macroHint("macro query_f! found\n", lineNumber)
//         fmt.Println(newLine, query_f)
//
//         rp := regexp.MustCompile("{{(.*?)}}")
//         rpq := regexp.MustCompile("#{{(.*?)}}")
//
//         matched := rp.FindAllString(newLine, -1)
//         qmatched := rpq.FindAllString(newLine, -1)
//
//         for _,qliteral := range qmatched {
//           clean := stripchars(qliteral, "#{}")
//           newLine = strings.Replace(newLine, qliteral, "'''||" + clean + "||'''", 1)
//         }
//
//         for _,literal := range matched {
//           clean := stripchars(literal, "{}")
//           newLine = strings.Replace(newLine, literal, "'||" + clean + "||'", 1)
//         }
//
//         // Append a space
//         newLine = strings.Replace(newLine, "'", "' ", 1)
//
//         p.Buffer.WriteString(newLine)
//         p.addNewLine()
//
//       } else {
//         macro := fmt.Sprintf("%s -- %s:%d", p.macroexpand(fileName, scanner.Text(), lineNumber), p.Namespace, lineNumber)
//         p.Buffer.WriteString(macro)
//         p.addNewLine()
//
//         macroHint("default macro\n")
//       }
//     }
//   }
//
//   if(p.State != "start"){
//     p.close_stmt("\n")
//     macroHint("end-of-file")
//   }
//
//   rapport(lineNumber)
//
//   if err := scanner.Err(); err != nil {
//       log.Fatal(err)
//   }
//
// }
//
// // Create a new parser
// func NewSQLParser() *SQLParser {
//
//   fmt.Println("Parser started")
//
// 	return &SQLParser{
// 		Folder:       "sql",
//     State:        "start",
// 		Writer:       os.Stdout,
// 	}
// }
