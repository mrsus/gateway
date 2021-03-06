package vm

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/robertkrimen/otto"
)

var locationRx = regexp.MustCompile(`(\d+):(\d+)`)

type jsError struct {
	err        error
	code       interface{}
	numContext int64
}

func (e *jsError) Error() string {
	errString := e.err.Error()
	if ottoErr, ok := e.err.(*otto.Error); ok {
		errString = ottoErr.String()
	}
	defaultError := fmt.Sprintf("JavaScript Error: %v\n\n--\n\n%v", errString, e.code)

	script, ok := e.code.(string)
	if !ok {
		return defaultError
	}
	match := locationRx.FindStringSubmatch(errString)
	if match == nil {
		return defaultError
	}
	lineNo, err := strconv.ParseInt(match[1], 10, 64)
	if err != nil {
		return defaultError
	}
	colNo, err := strconv.ParseInt(match[2], 10, 64)
	if err != nil {
		return defaultError
	}

	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf("JavaScript Error: %v\n\n--\n\n", errString))
	lines := strings.Split(script, "\n")
	for index, line := range lines {
		if index >= int(lineNo-e.numContext-1) && index < int(lineNo+e.numContext) {
			buffer.WriteString(line)
			buffer.WriteString("\n")
		}
		if index == (int(lineNo) - 1) {
			buffer.WriteString(strings.Repeat(" ", int(colNo)-1))
			buffer.WriteString("^ Error!\n")
		}
	}
	return buffer.String()
}
