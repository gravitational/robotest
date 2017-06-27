package config

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/gravitational/robotest/infra/gravity"

	"github.com/gravitational/trace"

	"gopkg.in/go-playground/validator.v9"
)

// ConfigFn is a function which takes default parameters and returns a pre-configured test function
type ConfigFn func(param interface{}) (gravity.TestFunc, error)

type entry struct {
	fn       ConfigFn
	defaults interface{}
}

// Entry is a pair of initialized test function and its parameters
type Entry struct {
	TestFunc gravity.TestFunc
	Param    interface{}
}

type Config struct {
	entries map[string]entry
}

func New() *Config {
	return &Config{map[string]entry{}}
}

// Add adds new entry to configuration
func (c *Config) Add(key string, fn ConfigFn, defaults interface{}) {
	c.entries[key] = entry{fn, defaults}
}

// Parse will take list of function=JSON, base config map, and return list of initialized test functions to run
func (c *Config) Parse(args []string) (fns map[string]Entry, err error) {
	var errs []error
	fns = map[string]Entry{}

	for _, arg := range args {
		var key string
		var data string

		split := withArgs.FindStringSubmatch(arg)
		if len(split) == 3 {
			key = split[1]
			data = split[2]
		} else {
			key = arg
		}

		entry, there := c.entries[key]
		if !there {
			errs = append(errs, trace.NotFound("no such function: %q in %+v", key, c))
			continue
		}

		e, err := makeFunction(entry.fn, data, entry.defaults)
		if err != nil {
			errs = append(errs, trace.Errorf("%s : %v", key, err))
			continue
		}

		fns[key] = *e
	}

	if len(errs) != 0 {
		return nil, trace.NewAggregate(errs...)
	}

	return fns, nil
}

var withArgs = regexp.MustCompile(`^(\S+)=(.+)$`)

func makeFunction(fn ConfigFn, data string, defaults interface{}) (*Entry, error) {
	param, err := parseJSON(data, defaults)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	err = Validate(param)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	var testFn gravity.TestFunc
	testFn, err = fn(param)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	return &Entry{testFn, param}, nil
}

// parseJSON parses JSON data using defaults object
func parseJSON(data string, defaults interface{}) (interface{}, error) {
	if data == "" {
		return defaults, nil
	}

	// use reflection as otherwise json.Unmarshal will set map[] object type
	// in many cases, overriding `defaults` object type

	decoder := reflect.ValueOf(json.Unmarshal)

	// make an object of underlying type of `defaults` and make it a copy
	out := reflect.New(reflect.TypeOf(defaults))
	out.Elem().Set(reflect.ValueOf(defaults))

	dataBytes := reflect.ValueOf([]byte(data))

	ret := decoder.Call([]reflect.Value{dataBytes, out})
	if ret[0].IsNil() {
		return reflect.Indirect(out).Interface(), nil
	}

	return nil, trace.Errorf("JSON decode %q failed: %v", data, ret[0].Interface())
}

func Validate(param interface{}) error {
	err := validator.New().Struct(param)
	return trace.Wrap(err)

	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		var errs []string
		for _, fieldError := range validationErrors {
			errs = append(errs,
				fmt.Sprintf("%s=%v is tag=%s struct_field=%s str=%s", fieldError.Field(), fieldError.Value(), fieldError.Tag(), fieldError.StructField(), fieldError.Param(), fieldError))
		}
		return trace.Errorf(strings.Join(errs, ", "))
	}
	return trace.Wrap(err)

}
