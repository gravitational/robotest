package infra

import (
	"bytes"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"text/scanner"
	"unicode"

	"github.com/gravitational/trace"
)

// ConfigureAgentCommandRunDetached interprets command as an agent download command
// and modifies it to execute the agent in background.
// For example:
//
// curl -s --tlsv1.2 --insecure -a="foo bar" "https://example.com/t/12437/node" | sudo bash
//
// will be modified to become:
//
// curl -s --tlsv1.2 --insecure -a="foo bar" "https://example.com/t/12437/node?bg=true" | sudo bash
//
// The function assumes that the pipeline contains a CURL command - otherwise it will fail.
func ConfigureAgentCommandRunDetached(command string) (string, error) {
	pipe := strings.Split(command, "|")
	var i int
	var curlCmd string
	var cmd string
	for i, cmd = range pipe {
		cmd = strings.TrimSpace(cmd)
		if strings.HasPrefix(cmd, "curl") {
			curlCmd = cmd
			break
		}
	}
	if curlCmd == "" {
		return "", trace.NotFound("unrecognized agent command: failed to find CURL: %q", command)
	}

	args, err := parseCommandLine(curlCmd)
	if err != nil {
		return "", trace.Wrap(err, "failed to parse CURL command %q", curlCmd)
	}

	for _, a := range args {
		switch a := a.(type) {
		case *arg:
			if a.name == "" && strings.HasPrefix(a.value, "http") {
				url, err := url.Parse(a.value)
				if err != nil {
					return "", trace.Wrap(err, "invalid agent download URL: %q", a.value)
				}
				q := url.Query()
				q.Set("bg", "true")
				url.RawQuery = q.Encode()
				a.value = url.String()
				pipe[i] = formatCommandline(args)
				return strings.Join(pipe, "|"), nil
			}
		}
	}
	return "", trace.NotFound("failed to find agent download URL in %q", command)
}

func formatCommandline(args []argument) string {
	output := make([]string, 0, len(args))
	for _, arg := range args {
		output = append(output, arg.format())
	}
	return strings.Join(output, " ")
}

func parseCommandLine(cmd string) (args []argument, err error) {
	p := &parser{}
	p.scanner.Init(strings.NewReader(cmd))
	p.scanner.IsIdentRune = isIdentRune
	p.next()
	for p.token != scanner.EOF && len(p.errors) == 0 {
		arg := p.parseArg()
		args = append(args, arg)
	}
	if len(p.errors) > 0 {
		return nil, trace.NewAggregate(p.errors...)
	}
	return args, nil
}

func isIdentRune(r rune, i int) bool {
	if unicode.IsLetter(r) || unicode.IsDigit(r) {
		// Accept digits even in the first position
		return true
	}
	if r == '-' && i < 2 {
		// Leading `-` or `--`
		return true
	}
	return r == '_' || r == '.'
}

func (r *parser) next() {
	r.token = r.scanner.Scan()
	r.pos = r.scanner.Position
	r.literal = r.scanner.TokenText()
}

func (r *parser) parseArg() argument {
	literal := r.token == scanner.String
	name := r.parseIndentOrLiteral()
	var value string
	if r.token == '=' {
		r.expect('=')
		value = r.parseIndentOrLiteral()
	}
	if name[0] == '-' && value == "" {
		return boolArg{name: name}
	}
	if literal {
		value = name
		name = ""
	}
	return &arg{
		name:  name,
		value: value,
	}
}

func (r *parser) parseIdent() string {
	name := r.literal
	r.expect(scanner.Ident)
	return name
}

func (r *parser) parseIndentOrLiteral() (value string) {
	var err error
	if value, err = strconv.Unquote(r.literal); err != nil {
		value = r.literal
	}
	r.expectOneOf(scanner.String, scanner.Ident)
	return value
}

func (r *parser) expect(token rune) {
	if r.token != token {
		r.error(r.pos, fmt.Sprintf("expected %v but got %v", scanner.TokenString(token), scanner.TokenString(r.token)))
	}
	r.next()
}

func (r *parser) expectOneOf(tokens ...rune) {
	if !tokenInSlice(r.token, tokens) {
		r.error(r.pos, fmt.Sprintf("expected any of %v but got %v", tokenStrings(tokens), scanner.TokenString(r.token)))
	}
	r.next()
}

func (r *parser) error(pos scanner.Position, message string) {
	r.errors = append(r.errors, trace.BadParameter("%v: %v", pos, message))
}

type parser struct {
	errors  []error
	scanner scanner.Scanner
	pos     scanner.Position
	token   rune
	literal string
}

type arg struct {
	name  string
	value string
}

func (r *arg) format() string {
	if r.name != "" {
		if r.value != "" {
			return fmt.Sprintf("%v=%v", r.name, quoteIfNecessary(r.value))
		} else {
			return fmt.Sprintf("%v", r.name)
		}
	} else {
		return fmt.Sprintf("%v", quoteIfNecessary(r.value))
	}
}

type boolArg struct {
	name string
}

func (r boolArg) format() string {
	return fmt.Sprintf("%v", r.name)
}

type argument interface {
	format() string
}

func tokenStrings(tokens []rune) string {
	var output bytes.Buffer
	for i, token := range tokens {
		if i > 0 {
			output.WriteByte(',')
		}
		output.WriteString(scanner.TokenString(token))
	}
	return output.String()
}

func tokenInSlice(needle rune, haystack []rune) bool {
	for _, token := range haystack {
		if token == needle {
			return true
		}
	}
	return false
}

func quoteIfNecessary(s string) string {
	// chars defines a character subset that determines if s will be quoted.
	// Besides whitespace, it contains all characters that can be part of an URL
	const chars = " :/@?&#"
	if strings.ContainsAny(s, chars) {
		return fmt.Sprintf(`"%v"`, s)
	}
	return s
}
