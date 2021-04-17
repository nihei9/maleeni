package compiler

import "fmt"

type SyntaxError struct {
	Message string
}

func newSyntaxError(msg string) *SyntaxError {
	return &SyntaxError{
		Message: msg,
	}
}

func (e *SyntaxError) Error() string {
	return fmt.Sprintf("syntax error: %s", e.Message)
}

var (
	// lexical errors
	synErrIncompletedEscSeq = newSyntaxError("incompleted escape sequence; unexpected EOF following \\")
	synErrInvalidEscSeq     = newSyntaxError("invalid escape sequence")

	// syntax errors
	synErrUnexpectedToken   = newSyntaxError("unexpected token")
	synErrNullPattern       = newSyntaxError("a pattern must be a non-empty byte sequence")
	synErrAltLackOfOperand  = newSyntaxError("an alternation expression must have operands")
	synErrRepNoTarget       = newSyntaxError("a repeat expression must have an operand")
	synErrGroupNoElem       = newSyntaxError("a grouping expression must include at least one character")
	synErrGroupUnclosed     = newSyntaxError("unclosed grouping expression")
	synErrGroupNoInitiator  = newSyntaxError(") needs preceding (")
	synErrGroupInvalidForm  = newSyntaxError("invalid grouping expression")
	synErrBExpNoElem        = newSyntaxError("a bracket expression must include at least one character")
	synErrBExpUnclosed      = newSyntaxError("unclosed bracket expression")
	synErrBExpInvalidForm   = newSyntaxError("invalid bracket expression")
	synErrRangeInvalidOrder = newSyntaxError("a range expression with invalid order")
)
