package objpath

func Query(v interface{}, path string) ([]Object, error) {
	if v == nil {
		return nil, nil
	}
	return QueryObjects([]Object{NewObject(v)}, path)
}

func QueryObject(v Object, path string) ([]Object, error) {
	return QueryObjects([]Object{v}, path)
}
func QueryObjects(v []Object, path string) ([]Object, error) {
	if len(v) == 0 {
		return nil, nil
	}
	exprs, err := parsePath(path)
	if err != nil {
		return nil, err
	}

	for _, expr := range exprs {
		v = expr.Filter(v)
		if len(v) == 0 {
			return nil, nil
		}
	}
	return v, nil
}
