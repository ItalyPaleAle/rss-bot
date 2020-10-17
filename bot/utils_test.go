package bot

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestGetArgs(t *testing.T) {
	cases := []struct {
		in  string
		out []string
	}{
		{`hi`, []string{`hi`}},
		{`hello world`, []string{`hello`, `world`}},
		// Double spaces
		{`hello  world`, []string{`hello`, `world`}},
		// Spaces around
		{` hello  world`, []string{`hello`, `world`}},
		{` hello  world `, []string{`hello`, `world`}},
		// Quotes
		{`"hello world"`, []string{`hello world`}},
		{`"hello world" hi`, []string{`hello world`, `hi`}},
		{` "hello world" hi`, []string{`hello world`, `hi`}},
		{` "hello  world" hi`, []string{`hello  world`, `hi`}},
		{` "hello  world"   hi `, []string{`hello  world`, `hi`}},
		{`hi "hello  world"`, []string{`hi`, `hello  world`}},
		{`hi "hello  world"  `, []string{`hi`, `hello  world`}},
		{`hi "helloworld"  `, []string{`hi`, `helloworld`}},
		{`hi "helloworld" hi `, []string{`hi`, `helloworld`, `hi`}},
		{`hi "helloworld" hi "ciao mondo"`, []string{`hi`, `helloworld`, `hi`, `ciao mondo`}},
		// Unterminated quotes
		{`hi "helloworld" hi "ciao mondo`, []string{`hi`, `helloworld`, `hi`, `ciao mondo`}},
		{`hi "helloworld" hi "ciao mondo `, []string{`hi`, `helloworld`, `hi`, `ciao mondo `}},
	}

	for _, el := range cases {
		res := GetArgs(el.in)
		ok := reflect.DeepEqual(res, el.out)
		if !ok {
			resJ, _ := json.Marshal(res)
			outJ, _ := json.Marshal(el.out)
			t.Fatalf("Expected result for %s to be equal %s, but got %s", el.in, outJ, resJ)
		}
	}
}
