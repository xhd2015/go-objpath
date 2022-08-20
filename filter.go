package objpath

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// example:
//    {
//	     "A":"10",
//       "B":"20"
//    }
// assert:
//   {
//	    "A":{
//         "$lt":"20",
//         "$gt":"$.B"
//      },
//      "B":"20"
//   }
//
// ObjectFilter assert against a value
type ObjectFilter interface {
	Filter(v []Object, root Object) ([]Object, Result)
}

type StringAssert string

// Check implements Assert
func (c StringAssert) Filter(v []Object, root Object) ([]Object, Result) {
	return filterPrimitive(string(c), v, root, func(curVal, incomingVal string) bool {
		return curVal == incomingVal
	})
}

func filterPrimitive(expectVal string, actualVals []Object, root Object, check func(actualVal string, expectVal string) bool) ([]Object, Result) {
	if len(actualVals) == 0 {
		return nil, nil
	}
	var errRes Result
	if strings.HasPrefix(expectVal, "$.") {
		path := expectVal[len("$."):]
		objs, err := QueryObject(root, path)
		if err != nil {
			return nil, Result{{BadSyntax: fmt.Sprintf("query path:%v %v", expectVal, err)}}
		}
		// no value found,return nil
		if len(objs) == 0 {
			return nil, nil
		}
		// both must be primitive
		// if map to multiple values, each value must be tested
		res := make([]Object, 0)
		for _, o := range actualVals {
			prim, ok := o.(Primitive)
			if !ok {
				continue
			}
			matchAll := true
			for _, obj := range objs {
				objPrim, ok := obj.(Primitive)
				if !ok {
					matchAll = false
					errRes.Append(&FailDetail{
						Expect: fmt.Sprintf("<%s => object>", expectVal),
						Actual: prim.StrValue(),
					})
					break
				}
				primStr := prim.StrValue()
				objPrimStr := objPrim.StrValue()
				if !check(primStr, objPrimStr) {
					matchAll = false
					errRes.Append(&FailDetail{
						Expect: objPrimStr,
						Actual: primStr,
					})
					break
				}
			}
			if matchAll {
				res = append(res, o)
			}
		}
		if len(res) > 0 {
			// clear debug errors
			errRes = nil
		}
		return res, errRes
	}
	res := make([]Object, 0)
	for _, o := range actualVals {
		// must be be simple primitive
		prim, ok := o.(Primitive)
		if !ok {
			// speical properties like $length
			act := "<object>"
			if o == nil {
				act = "null"
			}
			errRes.Append(&FailDetail{
				Expect: expectVal,
				Actual: act,
			})
			continue
		}
		actVal := prim.StrValue()

		if check(actVal, expectVal) {
			res = append(res, o)
		} else {
			errRes.Append(&FailDetail{
				Expect: expectVal,
				Actual: actVal,
			})
		}
	}
	if len(res) > 0 {
		// clear debug errors
		errRes = nil
	}
	return res, errRes
}

// == "A"
// < 120
// > B
//      $lt, $gt, $
//
// all kv must match
// {"A":"B"},{"D":"E"}
// CompositeFilter denotes a list of conditions and sub conditions must be held.
// if the key is an operator,like "$eq", that filter current value against
//   init: v = []Object{arg}
//         for each given filter:
//                 v = filter(v)
// a filter may consist of (currentObjects []Object, op Op,opTargets []Object)
//     for op, each currentObject must match all opTargets
//     for field, each currentObject[field]
type CompositeFilter map[string]ObjectFilter

func (c CompositeFilter) Filter(v []Object, root Object) ([]Object, Result) {
	var errRes Result
	objRes := make([]Object, 0)

	assertAll := false
	if false {
		// TODO: support assert all
		allFlag, ok := c["$all"]
		if ok {
			if allFlag, ok := allFlag.(StringAssert); ok && allFlag == "true" {
				assertAll = true
			}
		}
		_ = assertAll
	}

	for _, actVal := range v {
		match := true
		for key, expectFilter := range c {
			if key == "$all" {
				continue
			}
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			if key[0] == '#' {
				// comment
				continue
			}
			var childErrRes Result
			var objsByOp []Object
			//  special
			if key[0] == '$' {
				// for special assert, the value must be a string
				// otherwise
				expectVal, ok := expectFilter.(StringAssert)
				if !ok {
					errRes.Append(&FailDetail{
						Field:     key,
						BadSyntax: fmt.Sprintf("expect value to be string, found object"),
					})
					match = false
					break
				}
				// take special properties
				argActVal := actVal
				op := Op(key)
				if key == "$length" {
					op = "$eq"
					switch e := actVal.(type) {
					case Primitive:
						n := len(e.StrValue())
						argActVal = NewPrimitve(n, strconv.FormatInt(int64(n), 10))
					case Composite:
						argActVal = NewPrimitve(e.ChildrenLen(), strconv.FormatInt(int64(e.ChildrenLen()), 10))
					}
				}
				objsByOp, childErrRes = filterPrimitive(string(expectVal), []Object{argActVal}, root, op.Check)
			} else {
				var objs []Object
				var qerr error
				objs, qerr = QueryObject(actVal, key)
				if qerr != nil {
					errRes.Append(&FailDetail{
						Field:     key,
						BadSyntax: fmt.Sprintf("query path:%v %v", key, qerr),
					})
					match = false
					break
				}
				objsByOp, childErrRes = expectFilter.Filter(objs, root)
			}
			for _, childErr := range childErrRes {
				if childErr.Field != "" {
					childErr.Field = key + "." + childErr.Field
				} else {
					childErr.Field = key
				}
			}
			errRes.Append(childErrRes...)

			if len(objsByOp) == 0 {
				if childErrRes.Ok() {
					// if no child res, add reason
					errRes.Append(&FailDetail{
						Field: key,
					})
				}
				match = false
				break
			}
		}

		if match {
			if assertAll {
				// check uncovered primitive keys
			}
			objRes = append(objRes, actVal)
		}
	}
	if len(objRes) > 0 {
		// clear debug errors
		errRes = nil
	}
	return objRes, errRes
}

type Op string

const (
	OpEq         Op = "$eq"
	OpNeq        Op = "$neq"
	OpLt         Op = "$lt"
	OpLe         Op = "$le"
	OpGt         Op = "$gt"
	OpGe         Op = "$ge"
	OpContains   Op = "$contains"
	OpStartsWith Op = "$startsWith"
	OpEndsWith   Op = "$endsWith"
)

func (c Op) Check(curVal string, incomingVal string) bool {
	switch c {
	case OpEq:
		return curVal == incomingVal
	case OpNeq:
		return curVal != incomingVal
	case OpContains:
		return strings.Contains(curVal, incomingVal)
	case OpStartsWith:
		return strings.HasPrefix(curVal, incomingVal)
	case OpEndsWith:
		return strings.HasSuffix(curVal, incomingVal)
	case OpLt, OpLe, OpGe, OpGt:
		a, err := strconv.ParseFloat(curVal, 64)
		if err != nil {
			return false
		}
		b, err := strconv.ParseFloat(incomingVal, 64)
		if err != nil {
			return false
		}
		switch c {
		case OpLt:
			return a < b
		case OpLe:
			return a <= b
		case OpGt:
			return a > b
		case OpGe:
			return a >= b
		default:
			panic(fmt.Errorf("unexpected operator:%v", c))
		}
	default:
		// unknown operator
		return false
	}
}

type OperatorAssert struct {
	Op Op
}

func (c *OperatorAssert) Check(v Object, root Object) bool {
	switch c.Op {
	case OpEq:

	}
	return false
}

type JSONAssert struct {
	Assert ObjectFilter
}

func (c *JSONAssert) UnmarshalJSON(data []byte) error {
	assert, err := ParseJSONFilterBytes(data)
	if err != nil {
		return nil
	}
	c.Assert = assert
	return nil
}

func parseJSONFilter(data string) (ObjectFilter, error) {
	return ParseJSONFilterBytes([]byte(data))
}
func ParseJSONFilterBytes(data []byte) (ObjectFilter, error) {
	if len(data) == 0 {
		return nil, nil
	}
	var m interface{}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	err := dec.Decode(&m)
	if err != nil {
		return nil, err
	}
	return build(m)
}

func build(m interface{}) (ObjectFilter, error) {
	if m == nil {
		return nil, nil
	}

	switch m := m.(type) {
	case string:
		return StringAssert(m), nil
	case int64, int32, int16, int8, int:
		return StringAssert(fmt.Sprint(m)), nil
	case uint64, uint32, uint16, uint8, uint:
		return StringAssert(fmt.Sprint(m)), nil
	case float64, float32:
		return StringAssert(fmt.Sprint(m)), nil
	case bool:
		return StringAssert(strconv.FormatBool(m)), nil
	case json.Number:
		return StringAssert(m.String()), nil
	case map[string]interface{}:
		compositeAssert := make(CompositeFilter)
		for k, v := range m {
			k = strings.TrimLeftFunc(k, unicode.IsSpace)
			if k == "" || k[0] == '#' {
				// empty or comment
				continue
			}
			f, err := build(v)
			if err != nil {
				return nil, err
			}
			compositeAssert[k] = f
		}
		return compositeAssert, nil
	default:
		return nil, fmt.Errorf("unrecognized type:%v", m)
	}
}
