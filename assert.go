package objpath

import (
	"encoding/json"
	"fmt"
	"runtime"
	"strconv"
	"strings"
)

// Asserts syntax:
//    {
//        "a.b":{}
//    }
//
type Asserts struct {
	filter ObjectFilter
}

// Assert
func Assert(v interface{}, asserts string) {
	Check(v, asserts).Verify()
}
func AssertOk(str string, v bool) {
	CheckOk(str, v).Verify()
}
func AssertNoError(err error) {
	CheckNoError(err).Verify()
}
func AssertError(err error, str string) {
	CheckError(err, str).Verify()
}
func AssertResNoErr(res interface{}, err error, asserts string) {
	CheckResNoErr(res, err, asserts).Verify()
}

type T interface {
	Fatalf(format string, args ...interface{})
}

func AssertT(t T, v interface{}, asserts string) {
	Check(v, asserts).VerifyT(t)
}
func AssertOkT(t T, str string, v bool) {
	CheckOk(str, v).VerifyT(t)
}
func AssertNotOkT(t T, str string, v bool) {
	CheckNotOk(str, v).VerifyT(t)
}
func AssertNoErrorT(t T, err error) {
	CheckNoError(err).VerifyT(t)
}
func AssertErrorT(t T, err error, str string) {
	CheckError(err, str).VerifyT(t)
}
func AssertResNoErrT(t T, res interface{}, err error, asserts string) {
	CheckResNoErr(res, err, asserts).VerifyT(t)
}

func Check(v interface{}, asserts string) Result {
	asserter, err := ParseJSONAsserts(asserts)
	if err != nil {
		return Result{{BadSyntax: fmt.Sprintf("parsing assert: %v", err.Error())}}
	}
	if m, ok := asserter.filter.(CompositeFilter); ok {
		if len(m) == 0 {
			return Result{{NoAssert: true}}
		}
	}
	return asserter.Check(v)
}
func CheckOk(str string, v bool) Result {
	if !v {
		return Result{{Field: str, Expect: "true", Actual: "false"}}
	}
	return nil
}
func CheckNotOk(str string, v bool) Result {
	if !v {
		return Result{{Field: str, Expect: "false", Actual: "true"}}
	}
	return nil
}

func CheckNoError(err error) Result {
	if err == nil {
		return nil
	}
	return Result{{ForError: true, Expect: err.Error()}}
}
func CheckError(err error, str string) Result {
	if str == "" {
		return CheckNoError(err)
	}
	if err == nil {
		return Result{{ForError: true, Actual: str}}
	}
	errStr := err.Error()
	if strings.Contains(errStr, str) {
		return nil
	}
	return Result{{ForError: true, Expect: errStr, Actual: str}}
}
func CheckResNoErr(res interface{}, err error, asserts string) Result {
	asstRes := CheckNoError(err)
	if !asstRes.Ok() {
		return asstRes
	}
	return Check(res, asserts)
}

func (c *Asserts) Check(v interface{}) Result {
	root := NewObject(v)
	liveVals, res := c.filter.Filter([]Object{root}, root)
	if !res.Ok() {
		return res
	}
	if len(liveVals) == 0 {
		return Result{{Field: "<root>", Expect: "match", Actual: "no match"}}
	}
	return nil
}

func ParseJSONAsserts(asserts string) (*Asserts, error) {
	filter, err := parseJSONFilter(asserts)
	if err != nil {
		return nil, fmt.Errorf("parse assert: %v", err)
	}
	return &Asserts{
		filter: filter,
	}, nil
}
func MustParseJSONAsserts(asserts string) *Asserts {
	asserter, err := ParseJSONAsserts(asserts)
	if err != nil {
		panic(err)
	}
	return asserter
}

// UnmarshalJSON implements json.Unmarshaler
func (c *Asserts) UnmarshalJSON(data []byte) error {
	filter, err := ParseJSONFilterBytes(data)
	if err != nil {
		return err
	}
	*c = Asserts{
		filter: filter,
	}
	return nil
}

type Result []*FailDetail

func (c *Result) Append(d ...*FailDetail) {
	if len(d) > 0 {
		*c = append(*c, d...)
	}
}
func (c Result) Ok() bool {
	return len(c) == 0
}
func (c Result) Verify() {
	if len(c) != 0 {
		panic(fmt.Errorf("assert error: %s", c.String()))
	}
}
func (c Result) VerifyT(t T) {
	if len(c) != 0 {
		fileLine := FormatFileLine(3)
		t.Fatalf("\r    %s: assert error: %s", fileLine, c.String())
	}
}
func FormatFileLine(callDepth int) string {
	var pc [1]uintptr
	n := runtime.Callers(callDepth+1, pc[:]) // N(this pkg)+1(runtime.Callers itself)
	var file string
	var line int
	if n > 0 {
		frames := runtime.CallersFrames(pc[:n])
		frame, _ := frames.Next()
		if frame.Func != nil {
			frameFile := frame.File
			line = frame.Line
			if line == 0 {
				line = 1
			}
			if frameFile != "" {
				// Truncate file name at last file name separator.
				if index := strings.LastIndex(frameFile, "/"); index >= 0 {
					file = frameFile[index+1:]
				} else if index = strings.LastIndex(frameFile, "\\"); index >= 0 {
					file = frameFile[index+1:]
				}
			}
		}
	}
	if file == "" {
		file = "???"
	}
	return fmt.Sprintf("%s:%d", file, line)
}
func (c Result) String() string {
	var b strings.Builder
	for i, detail := range c {
		b.WriteString(detail.String())
		if i < len(c)-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

type FailDetail struct {
	Field     string `json:"field,omitempty"`
	ForError  bool   `json:"for_error,omitempty"`
	Expect    string `json:"expect,omitempty"`
	Actual    string `json:"actual,omitempty"`
	NoAssert  bool   `json:"no_assert,omitempty"`
	BadSyntax string `json:"bad_syntax,omitempty"`
	Str       string `json:"str"` // representation of this detail
}

func (c *FailDetail) MarshalJSON() ([]byte, error) {
	// ensure that Str is update-to-date
	var wrap = &struct {
		FailDetail
	}{
		FailDetail: *c,
	}
	wrap.Str = c.String()
	return json.Marshal(wrap)
}

func (c *FailDetail) String() string {
	if c.BadSyntax != "" {
		return fmt.Sprintf("bad syntax at %s: %s", c.Field, c.BadSyntax)
	}
	if c.NoAssert {
		return "no assert"
	}
	if c.ForError {
		if c.Expect == "" {
			return fmt.Sprintf("expect no error, actual: %s", c.Actual)
		}
		if c.Actual == "" {
			return fmt.Sprintf("expect error: %s, actual no error", c.Expect)
		}
		return fmt.Sprintf("expect error: %s, actual: %v", c.Expect, c.Actual)
	}
	if c.Field != "" {
		return fmt.Sprintf("expect %s to be %s, actual: %s", c.Field, strconv.Quote(c.Expect), strconv.Quote(c.Actual))
	}
	// general fail message
	return "fail"
}
