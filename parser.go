package gel

import (
	"bytes"
	"fmt"
	"github.com/amaniak/gel/token"
	"github.com/fatih/color"
	"regexp"
	"strings"
)

const (
	DEBUG = true
)

var trace = func(pos string, msg string) {
	colorize := color.New(color.FgYellow)
	if DEBUG {
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
	CurrentState     token.Token
	Scanner          *Scanner
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

	if len(p.CurrentNamespace) < 0 || p.CurrentNamespace != node.FileInfo.Namespace {
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
	macro := strings.Replace(node.Text(), token.IMFUNC.String()+" ", functionName, -1)

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
	macro := strings.Replace(node.Text(), token.FUNC.String()+" ", functionName, -1)

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
	macro := strings.Replace(node.Text(), token.IMPROC.String()+" ", functionName, -1)

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
	macro := strings.Replace(node.Text(), token.PROC.String()+" ", functionName, -1)

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
	cleaned := strings.Replace(node.Text(), "this.", node.FileInfo.Namespace+".", -1)
	node.Source = strings.Replace(cleaned, "getv(", "vars.getv(", -1)
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

	rp := regexp.MustCompile("{{(.*?)}}")
	rpq := regexp.MustCompile("#{{(.*?)}}")

	statement := strings.Replace(node.Text(), "|> ", "", -1)
	vars := rp.FindAllString(statement, -1)
	qvars := rpq.FindAllString(statement, -1)

	for _, qliteral := range qvars {
		clean := stripchars(qliteral, "#{}")
		statement = strings.Replace(statement, qliteral, "'''||"+clean+"||'''", 1)
	}

	for _, literal := range vars {
		clean := stripchars(literal, "{}")
		statement = strings.Replace(statement, literal, "'||"+clean+"||'", 1)
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

	switch {
	case p.State(token.FUNC):
		p.CloseFuncKeyWord(node, "IMMUTABLE")
	case p.State(token.IMFUNC):
		p.CloseFuncKeyWord(node, "")
	case p.State(token.PROC):
		p.CloseProcKeyWord(node, "IMMUTABLE")
	case p.State(token.IMPROC):
		p.CloseProcKeyWord(node, "")
	}
}

func (p *Parser) Parse() {

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

			slashed, _ := regexp.MatchString("\\", node.Text())
			statement, _ := regexp.MatchString("$SQL", node.Text())

			if slashed || statement {
				p.MacroScopeExpand(node)
				trace(node.Line(), "function macro expanded")
			} else {
				p.MacroExpandDefault(node)
				trace(node.Line(), "default macro\n")
			}

		}
	}

	if !p.State(token.START) {

		node := EmptyAstNode()

		p.CloseStatment(node)

		p.Scanner.Buffer = append(p.Scanner.Buffer, node)

		trace("-999", "EOF")
	}
}
