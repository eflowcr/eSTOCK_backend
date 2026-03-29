package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type validStruct struct {
	Name  string `validate:"required"`
	Email string `validate:"required,email"`
	Age   int    `validate:"min=0,max=150"`
}

func TestValidateStruct_Valid(t *testing.T) {
	v := validStruct{Name: "Alice", Email: "alice@test.com", Age: 30}
	errs := ValidateStruct(v)
	assert.Nil(t, errs)
}

func TestValidateStruct_MissingRequired(t *testing.T) {
	v := validStruct{Email: "alice@test.com"}
	errs := ValidateStruct(v)
	require.NotNil(t, errs)
	require.Len(t, errs, 1)
	assert.Equal(t, "Name", errs[0].Field)
	assert.Equal(t, "required", errs[0].Message)
}

func TestValidateStruct_MultipleErrors(t *testing.T) {
	v := validStruct{}
	errs := ValidateStruct(v)
	require.NotNil(t, errs)
	assert.GreaterOrEqual(t, len(errs), 2)
}

func TestValidateStruct_InvalidEmail(t *testing.T) {
	v := validStruct{Name: "Bob", Email: "not-an-email", Age: 25}
	errs := ValidateStruct(v)
	require.NotNil(t, errs)
	require.Len(t, errs, 1)
	assert.Equal(t, "Email", errs[0].Field)
}

func TestValidateStruct_OutOfRange(t *testing.T) {
	v := validStruct{Name: "Carol", Email: "carol@test.com", Age: 200}
	errs := ValidateStruct(v)
	require.NotNil(t, errs)
	require.Len(t, errs, 1)
	assert.Equal(t, "Age", errs[0].Field)
	assert.Contains(t, errs[0].Message, "max")
}

func TestValidateStruct_WithParam(t *testing.T) {
	// validate that Message contains the param (e.g. "max:150")
	v := validStruct{Name: "Dave", Email: "dave@test.com", Age: -1}
	errs := ValidateStruct(v)
	require.NotNil(t, errs)
	assert.Contains(t, errs[0].Message, "min")
}
