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
	Filter(v []Object, root Object) ([]Object, error)
}

type StringAssert string

// Check implements Assert
func (c StringAssert) Filter(v []Object, root Object) ([]Object, error) {
	return filterPrimitive(string(c), v, root, func(curVal, incomingVal string) bool {
		return curVal == incomingVal
	})
}

func filterPrimitive(incomingValOrExpr string, curVals []Object, root Object, check func(curVal string, incomingVal string) bool) ([]Object, error) {
	if len(curVals) == 0 {
		return nil, nil
	}
	s := incomingValOrExpr
	if strings.HasPrefix(s, "$.") {
		path := s[len("$."):]
		objs, err := QueryObject(root, path)
		if err != nil {
			return nil, fmt.Errorf("query path error:%v %v", s, err)
		}
		// no value found,return nil
		if len(objs) == 0 {
			return nil, nil
		}
		// both must be primitive
		// if map to multiple values, each value must be tested
		res := make([]Object, 0)
		for _, o := range curVals {
			prim, ok := o.(Primitive)
			if !ok {
				continue
			}
			matchAll := true
			for _, obj := range objs {
				objPrim, ok := obj.(Primitive)
				if !ok || !check(prim.StrValue(), objPrim.StrValue()) {
					matchAll = false
					break
				}
			}
			if matchAll {
				res = append(res, o)
			}
		}
		return res, nil
	}
	res := make([]Object, 0)
	for _, o := range curVals {
		prim, ok := o.(Primitive)
		if !ok {
			continue
		}
		if check(prim.StrValue(), s) {
			res = append(res, o)
		}
	}
	return res, nil
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

func (c CompositeFilter) Filter(v []Object, root Object) ([]Object, error) {
	res := make([]Object, 0)
	for _, e := range v {
		match := true
		for key, filter := range c {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			if key[0] == '#' {
				// comment
				continue
			}
			var err error
			var objsByOp []Object
			//  special
			if key[0] == '$' {
				// for special assert, the value must be a string
				// otherwise
				strVal, ok := filter.(StringAssert)
				if !ok {
					match = false
					break
				}
				objsByOp, err = filterPrimitive(string(strVal), []Object{e}, root, Op(key).Check)
			} else {
				var objs []Object
				objs, err = QueryObject(e, key)
				if err != nil {
					return nil, err
				}
				objsByOp, err = filter.Filter(objs, root)
			}
			if err != nil {
				return nil, err
			}
			if len(objsByOp) == 0 {
				match = false
				break
			}
		}
		if match {
			res = append(res, e)
		}
	}
	return res, nil
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
