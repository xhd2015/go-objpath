package objpath

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

type pathExpr interface {
	// example: original map: {"a.b.c":"1"} => {"b.c":"1"}
	// ${prefix}.${field}
	// or ${prefix}.*{}
	// Filter filter and navigate through candidates
	Filter(candidates []Object) []Object
}
type literalField string
type verbatim string

type variableField struct {
	field     string // empty for any
	condition map[string]string
}

func (c literalField) Filter(objects []Object) []Object {
	var res []Object
	s := string(c)
	hasWildcard := strings.Contains(s, "*")

	for _, obj := range objects {
		if obj == nil {
			continue
		}
		switch obj := obj.(type) {
		case Primitive:
			if c == "$length" {
				length := len(obj.StrValue())
				n := strconv.FormatInt(int64(length), 10)
				res = append(res, NewPrimitve(length, n))
				continue
			}
		case Composite:
			if c == "*" {
				// a special version of *{}
				obj.RangeChildren(func(key string, child Object) bool {
					res = append(res, child)
					return true
				})
				continue
			} else if c == "$length" {
				childLen := obj.ChildrenLen()
				n := strconv.FormatInt(int64(childLen), 10)
				res = append(res, NewPrimitve(childLen, n))
				continue
			}
			if !hasWildcard {
				v, ok := obj.GetChild(s)
				if !ok {
					// check if method matches
					meth, ok := obj.Method(s)
					if !ok {
						continue
					}
					methRes, err := CallFn(meth, nil)
					if err != nil {
						panic(fmt.Errorf("call %s: %v", s, err))
					}
					if len(methRes) == 0 {
						// no return
						continue
					}
					if len(methRes) > 1 {
						panic(fmt.Errorf("call %s returns more than 1 result:%d", s, len(methRes)))
					}
					res = append(res, NewObject(methRes[0]))
					continue
				}

				res = append(res, v)
			} else {
				obj.RangeChildren(func(key string, child Object) bool {
					if globMatch(key, s) {
						res = append(res, child)
					}
					return true
				})
			}
		default:
			panic(fmt.Errorf("unhandled obj:%T", obj))
		}
	}

	return res
}
func (c verbatim) Filter(objects []Object) []Object {
	var res []Object
	for _, obj := range objects {
		if obj == nil {
			continue
		}
		switch obj := obj.(type) {
		case Primitive:
			// ignore
		case Composite:
			v, ok := obj.GetChild(string(c))
			if !ok {
				continue
			}
			res = append(res, v)
		default:
			panic(fmt.Errorf("unhandled obj:%T", obj))
		}
	}
	return res
}

// TODO: make it util
func prefixedOrEquals(s string, prefix string, split string) (next string, ok bool) {
	hasPrefix := strings.HasPrefix(s, prefix)
	if !hasPrefix {
		return "", false
	}
	if len(s) == len(prefix) {
		return "", true
	}
	suffix := s[len(prefix):]
	if split == "" {
		return suffix, true
	}
	if !strings.HasPrefix(suffix, split) {
		return "", false
	}
	return suffix[len(split):], true
}

func (c *variableField) Filter(objects []Object) []Object {
	res := make([]Object, 0)
	for _, obj := range objects {
		if obj == nil {
			continue
		}
		switch obj := obj.(type) {
		case Primitive:
			// ignore
		case Composite:
			obj.RangeChildren(func(key string, child Object) bool {
				if !globMatch(key, c.field) {
					return true
				}

				if len(c.condition) == 0 {
					res = append(res, child)
					return true
				}

				ch := []Object{child}
				for k, v := range c.condition {
					var err error
					ch, err = QueryObjects(ch, k)
					if err != nil {
						// invalid path, skip
						break
					}
					if len(ch) == 0 {
						break
					}
					j := 0
					for i := 0; i < len(ch); i++ {
						prim, ok := ch[i].(Primitive)
						if !ok {
							continue
						}
						if prim.StrValue() == v {
							ch[j] = ch[i]
							j++
						}
					}
					ch = ch[:j]
					if len(ch) == 0 {
						break
					}
				}
				if len(ch) > 0 {
					res = append(res, child)
				}
				return true
			})
		default:
			panic(fmt.Errorf("unhandled obj:%T", obj))
		}
	}
	return res
}

func debugString(p pathExpr) string {
	switch p := p.(type) {
	case literalField:
		return string(p)
	case verbatim:
		return "verbatim:" + string(p)
	case *variableField:
		return fmt.Sprintf("condition:%v", p.field)
	default:
		panic(fmt.Errorf("unrecognized type:%T", p))
	}
}

// g(P,s):
//    if P not startsWith *, then P[:i]==s[:i] && g(P[i:],s[i:])
//    else g(P,s[1:]) or g(P[1:],s)
func globMatch(key string, glob string) bool {
	if glob == "" {
		return key == ""
	}

	if true {
		matched, _ := filepath.Match(glob, key)
		return matched
	}

	if glob == "*" {
		return true
	}

	strings.SplitAfter(glob, "*")

	prefixGlob := strings.HasPrefix(glob, "*")
	suffixGlob := strings.HasSuffix(glob, "*")

	str := strings.TrimPrefix(glob, "*")
	str = strings.TrimSuffix(str, "*")
	if prefixGlob && suffixGlob {
		return strings.Contains(key, str)
	}
	if prefixGlob {
		return strings.HasSuffix(key, str)
	}
	if suffixGlob {
		return strings.HasPrefix(key, str)
	}
	return key == str
}
