package config

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"

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
	TestFunc gravity.TestFunc `json:"-"`
	Param    interface{}
}

type TestSet map[string]Entry

func (t TestSet) add(key string, e Entry) {
	if _, there := t[key]; !there {
		t[key] = e
		return
	}

	for i := 2; ; i++ {
		k := fmt.Sprintf("%s%d", key, i)
		if _, there := t[k]; there {
			continue
		}
		t[k] = e
		return
	}
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
func (c *Config) Parse(args []string) (fns TestSet, err error) {
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
			errs = append(errs, trace.NotFound("no such function: %q", key))
			continue
		}

		e, err := makeFunction(entry.fn, data, entry.defaults)
		if err != nil {
			errs = append(errs, trace.Errorf("%s : %v", key, err))
			continue
		}

		fns.add(key, *e)
	}

	if len(errs) != 0 {
		return nil, trace.NewAggregate(errs...)
	}

	return fns, nil
}

var withArgs = regexp.MustCompile(`^(\S+)=(.+)$`)

func makeFunction(fn ConfigFn, data string, defaults interface{}) (*Entry, error) {
	// make a pointer to a new object of underlying type of `defaults`
	paramPtr := reflect.New(reflect.TypeOf(defaults))
	// point it to a copy of the data from defaults
	// TODO consider some sort of deepcopy or eliminating the option to pass default
	// values. Presently a ref in the parameter defaults is asking for trouble.
	paramPtr.Elem().Set(reflect.ValueOf(defaults))

	err := json.Unmarshal([]byte(data), paramPtr.Interface())
	if err != nil {
		return nil, trace.BadParameter("JSON decode %q failed: %v", data, err)
	}

	err = checkAndSetDefaults(paramPtr.Interface())
	if err != nil {
		return nil, trace.Wrap(err)
	}

	param := reflect.Indirect(paramPtr).Interface()

	var testFn gravity.TestFunc
	testFn, err = fn(param)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	return &Entry{testFn, param}, nil
}

type defaulter interface {
	CheckAndSetDefaults() error
}

// checkAndSetDefaults validates parameters according to struct field tags and
// custom logic specified by implementing the Validator interface.
func checkAndSetDefaults(param interface{}) error {
	if err := validator.New().Struct(param); err != nil {
		return trace.Wrap(err)
	}

	if d, ok := param.(defaulter); ok {
		return trace.Wrap(d.CheckAndSetDefaults())
	}
	return nil
}
