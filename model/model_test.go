package model_test

import (
	"errors"
	"github.com/ServiceComb/go-sc-client/model"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestModelException(t *testing.T) {
	t.Log("Testing modelReg.Error function")
	var modelReg *model.RegistyException = new(model.RegistyException)
	modelReg.Message = "Go-chassis"
	modelReg.Title = "fakeTitle"

	str := modelReg.Error()
	assert.Equal(t, "fakeTitle(Go-chassis)", str)

}

func TestModelExceptionOrglErr(t *testing.T) {
	t.Log("Testing modelReg.Error with title")
	var modelReg *model.RegistyException = new(model.RegistyException)
	modelReg.Message = "Go-chassis"
	modelReg.Title = "fakeTitle"
	modelReg.OrglErr = errors.New("Invalid")

	str := modelReg.Error()
	assert.Equal(t, "fakeTitle(Invalid), Go-chassis", str)

}
func TestNewCommonException(t *testing.T) {
	t.Log("Testing NewCommonException function")
	var re *model.RegistyException = new(model.RegistyException)
	re.OrglErr = nil
	re.Title = "Common exception"
	re.Message = "fakeformat"
	err := model.NewCommonException("fakeformat")
	assert.Equal(t, re, err)
}
func TestNewJsonException(t *testing.T) {
	t.Log("Testing NewJSONException function")
	var re1 *model.RegistyException = new(model.RegistyException)
	re1.OrglErr = errors.New("Invalid")
	re1.Title = "JSON exception"
	re1.Message = "args1"

	err := model.NewJSONException(errors.New("Invalid"), "args1")
	assert.Equal(t, re1, err)

	var re2 *model.RegistyException = new(model.RegistyException)
	re2.OrglErr = errors.New("Invalid")
	re2.Title = "JSON exception"
	re2.Message = ""

	err = model.NewJSONException(errors.New("Invalid"))
	assert.Equal(t, re2, err)

	var re3 *model.RegistyException = new(model.RegistyException)
	re3.OrglErr = errors.New("Invalid")
	re3.Title = "JSON exception"
	re3.Message = "[1]"

	err = model.NewJSONException(errors.New("Invalid"), 1)
	assert.Equal(t, re3, err)

}

func TestNewIOException(t *testing.T) {
	t.Log("Testing NewIOException function")
	var re1 *model.RegistyException = new(model.RegistyException)
	re1.OrglErr = errors.New("Invalid")
	re1.Title = "IO exception"
	re1.Message = "args1"

	err := model.NewIOException(errors.New("Invalid"), "args1")
	assert.Equal(t, re1, err)

	var re2 *model.RegistyException = new(model.RegistyException)
	re2.OrglErr = errors.New("Invalid")
	re2.Title = "IO exception"
	re2.Message = ""

	err = model.NewIOException(errors.New("Invalid"))
	assert.Equal(t, re2, err)

	var re3 *model.RegistyException = new(model.RegistyException)
	re3.OrglErr = errors.New("Invalid")
	re3.Title = "IO exception"
	re3.Message = "[1]"

	err = model.NewIOException(errors.New("Invalid"), 1)
	assert.Equal(t, re3, err)

}
