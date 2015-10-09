package gel

import (
	"bufio"
	"bytes"
	"github.com/amaniak/gel/token"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const (
	SQL_EXTENSTION = ".sql"
)

type FileInfo struct {
	Namespace string
	FileName  string
	Location  string
}
type Ast struct {
	*FileInfo
	Position int
	Raw      string
	Bytes    []byte
	Source   string
}

func EmptyAstNode() *Ast {
	buf := new([]byte)
	a := &Ast{&FileInfo{}, 0, "", *buf, ""}
	return a
}

func (node *Ast) IsImFunc() bool {
	match, _ := regexp.MatchString(token.IMFUNC.String(), node.Text())
	return match
}

func (node *Ast) IsFunc() bool {
	match, _ := regexp.MatchString(token.FUNC.String(), node.Text())
	return match
}

func (node *Ast) IsImProc() bool {
	match, _ := regexp.MatchString(token.IMPROC.String(), node.Text())
	return match
}

func (node *Ast) IsProc() bool {
	match, _ := regexp.MatchString(token.PROC.String(), node.Text())
	return match
}

func (node *Ast) IsExec() bool {
	match, _ := regexp.MatchString(token.EXEC.String(), node.Text())
	return match
}

func (node *Ast) IsQuery() bool {
	match, _ := regexp.MatchString(token.QUERY.String(), node.Text())
	return match
}

func (node *Ast) IsComment() bool {
	match, _ := regexp.MatchString(token.COMMENT.String(), node.Text())
	return match
}

func (node *Ast) IsEmpty() bool {
	check := bytes.TrimSpace(node.Bytes)

	if len(check) == 0 {
		return true
	}
	return false
}

func (node *Ast) IsNewLine() bool {
	return node.Text() == "\n"
}

func (node *Ast) IsWhiteSpace() bool {
	reg, _ := regexp.Compile(`^\s`)
	return reg.MatchString(node.Text())
}

func (node *Ast) Line() string {
	return strconv.Itoa(node.Position)
}

func (node *Ast) Text() string {
	return node.Raw
}

type Scanner struct {
	Path   string
	Buffer []*Ast
	Tree   map[string]*Ast
}

func (s *Scanner) Load() {
	err := filepath.Walk(s.Path, s.findSQLFiles)
	if err != nil {
		log.Fatal("Something went wrong")
	}
}

func (s *Scanner) findSQLFiles(pathToFile string, f os.FileInfo, err error) error {
	if filepath.Ext(f.Name()) == SQL_EXTENSTION {

		namespace := strings.TrimSuffix(f.Name(), filepath.Ext(f.Name()))
		file, err := os.Open(pathToFile)

		defer file.Close()

		buffer := bufio.NewScanner(file)

		if err != nil {
			log.Fatal(err)
		}

		line := 1

		for buffer.Scan() {
			fileInfo := &FileInfo{namespace, f.Name(), pathToFile}

			node := &Ast{fileInfo, line,
				buffer.Text(), buffer.Bytes(), ""}
			s.Buffer = append(s.Buffer, node)
			line = line + 1
		}
	}
	return nil
}
