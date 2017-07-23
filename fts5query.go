package main

import (
	"errors"
)

// Convert a Google search like query into a FTS5 query, i.e. to not
// trigger the column filter misfeature. We don't try to make invalid queries
// work, just make queries that would be interpreted as column filters or the
// like work.
//
// based on: https://github.com/fazalmajid/temboz/blob/master/tembozapp/fts5.py
//
// EBNF:
// query = word
// word = [not space]+
// query = phrase
// phrase = '"' [^"]* '"'
// query = "(" query ")"
// query = query "AND" query
// query = query "OR" query
// query = "NOT" query
func fts5_term(term string) (string, error) {
	in_q := false
	implicit_q := false
	out := ""
	expect := '\000'
	pending := ""
	for _, c := range term {
		switch {
		case c == '\'': // SQL injection guard
			if pending != "" {
				out = out + pending
				pending = ""
			}
			out = out + "''"
		case expect != '\000':
			if c != expect {
				if in_q {
					return "", errors.New("query parse error 1")
				}
				out = out + "\"" + pending + "\"" + string(c)
				expect = '\000'
				pending = ""
			} else {
				pending = pending + string(c)
				switch expect {
				case 'N':
					expect = 'D'
				case 'O':
					expect = 'T'
				case 'D', 'R', 'T':
					if in_q {
						return "", errors.New("query parse error 2")
					}
					expect = '\000'
					out = out + pending
					pending = ""
				default:
					return "", errors.New("query parse error 3")
				}
			}
		case c == '"':
			if implicit_q {
				implicit_q = false
			} else {
				in_q = !in_q
			}
			out = out + string(c)
		case !in_q && (c == 'A' || c == 'O' || c == 'N'):
			// start of AND, OR or NOT
			switch c {
			case 'A':
				expect = 'N'
			case 'O':
				expect = 'R'
			case 'N':
				expect = 'O'
			}
			pending = string(c)
		case c == ' ', c == '\t', c == '\n', c == '(', c == ')':
			if implicit_q {
				out = out + "\""
				implicit_q = false
				in_q = false
			}
			out = out + string(c)
		default:
			if in_q {
				out = out + string(c)
			} else {
				in_q = true
				implicit_q = true
				out = out + "\"" + string(c)
			}
		}
	}
	out = out + pending
	if in_q {
		out = out + "\""
	}
	return out, nil
}
