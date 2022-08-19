package objpath

import "fmt"

// Asserts syntax:
//    {
//        "a.b":{}
//    }
//
type Asserts struct {
	filter ObjectFilter
}

// Assert
func Assert(v interface{}, asserts string) (bool, error) {
	asserter, err := ParseJSONAsserts(asserts)
	if err != nil {
		return false, err
	}
	return asserter.Assert(v)
}

func (c *Asserts) Assert(v interface{}) (bool, error) {
	root := NewObject(v)
	liveVals, err := c.filter.Filter([]Object{root}, root)
	if err != nil {
		return false, err
	}
	return len(liveVals) > 0, nil
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
