package token

import "strings"

type Token int

const (
	START Token = iota
	COMMENT
	FUNC
	IMFUNC
	PROC
	IMPROC
	EXEC
	QUERY
	IDENT
	EOF
)

var tokens = [...]string{
	COMMENT: "--",
	FUNC:    "func",
	IMFUNC:  "func!",
	PROC:    "proc",
	IMPROC:  "proc!",
	EXEC:    "exec!",
	QUERY:   "|>",
	IDENT:   "",
}

var keywords map[string]Token

func init() {
	keywords = make(map[string]Token)
	for tok := START + 1; tok < EOF; tok++ {
		keywords[strings.ToLower(tokens[tok])] = tok
	}
}

// Lookup returns the token associated with a given string.
func Lookup(ident string) Token {
	if tok, ok := keywords[strings.ToLower(ident)]; ok {
		return tok
	}
	return IDENT
}

// Cast token to string
func (tok Token) String() string {
	s := ""
	if 0 <= tok && tok < Token(len(tokens)) {
		s = tokens[tok]
	}
	return s
}
