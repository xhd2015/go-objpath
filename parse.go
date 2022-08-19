package objpath

import (
	"fmt"
	"strings"
)

// example: a{k=v}
// example: a.[x,y,z]
func parsePath(path string) ([]pathExpr, error) {
	var exprs []pathExpr
	for {
		// in each iteration, we
		// find the following:
		//  A.B
		//  A{}.B
		//  A.[BB]
		if path == "" {
			return exprs, nil
		}
		// []
		// a.[]
		// a[]
		// .[]

		// lookup special index: .{[
		idx := strings.IndexAny(path, "{[.")
		if idx < 0 {
			// end
			exprs = append(exprs, literalField(path))
			return exprs, nil
		}

		// A{}
		// TODO: support {}{}
		if path[idx] == '{' {
			eidx, err := lookClose(path, idx, "{", "}")
			if err != nil {
				return nil, err
			}

			key := path[:idx]
			kvs := path[idx+1 : eidx]
			path = path[eidx+1:]

			condition := parsePairs(kvs)
			exprs = append(exprs, &variableField{
				field:     key,
				condition: condition,
			})
			continue
		}
		if path[idx] == '.' {
			// TODO: don't allow A..B, allow only A.B, A.[B]
			if idx > 0 {
				// non-empty
				exprs = append(exprs, literalField(path[:idx]))
			}
			path = path[idx+1:]
			continue
		}
		if path[idx] == '[' {
			if idx > 0 {
				// non-empty
				exprs = append(exprs, literalField(path[:idx]))
				path = path[idx:]
				idx = 0
			}
			// verbatim
			eidx, err := lookClose(path, idx, "[", "]")
			if err != nil {
				return nil, err
			}
			vk := path[idx+1 : eidx]
			if vk == "" {
				return nil, fmt.Errorf("invalid syntax: found empty '[]' at %v", path)
			}
			if false {
				// TODO: check when to use verbatim
				exprs = append(exprs, verbatim(vk))
			} else {
				exprs = append(exprs, literalField(vk))
			}

			path = path[eidx+1:]
			continue
		}
		return nil, fmt.Errorf("invalid syntax: unrecognized symbol at %v", path)
	}
}

// parsePairs k1=v1,k2=v2,...
func parsePairs(s string) map[string]string {
	m := make(map[string]string)
	kvs := strings.Split(s, ",")
	for _, kv := range kvs {
		kvSplit := strings.SplitN(kv, "=", 2)
		var k string
		var v string
		if len(kvSplit) > 0 {
			k = kvSplit[0]
		}
		if len(kvSplit) > 1 {
			v = kvSplit[1]
		}
		if k == "" {
			// ignore
			continue
		}
		m[k] = v
	}
	return m
}

func lookClose(path string, idx int, open string, close string) (int, error) {
	endIdx := -1
	tmpIdx := strings.Index(path[idx+1:], close)
	if tmpIdx >= 0 {
		endIdx = tmpIdx + idx + 1
	}
	if endIdx < 0 {
		return -1, fmt.Errorf("invalid syntax: found '%s', but missing '%s' at %v", open, close, path)
	}
	return endIdx, nil
}
