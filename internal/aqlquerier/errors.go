package aqlquerier

import (
	"fmt"
)

var (
	ErrNotImplemented      = fmt.Errorf("Not implemented")
	ErrInvalidQuery        = fmt.Errorf("Invalid AQL query sintax")
	ErrInvalidOperandCount = fmt.Errorf("Invalid AQL operands count")
	ErrInvalidWhere        = fmt.Errorf("Invalid WHERE block in AQL query")
)
