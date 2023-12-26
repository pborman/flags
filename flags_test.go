// Copyright 2023 Paul Borman
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and

package flags

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/pborman/check"
)

type X string

func (x *X) Set(s string) error { *x = X(s); return nil }
func (x *X) String() string     { return (string)(*x) }
func (x *X) Get() any           { return *x } // Needed by some flag packages

func TestVar(t *testing.T) {
	var x X
	fs := flag.NewFlagSet("v", flag.ExitOnError)
	if err := setvar(fs, &x, "flag", "usage"); err != nil {
		t.Fatal(err)
	}
	found := false
	fs.VisitAll(func(f *flag.Flag) {
		if f.Name == "flag" {
			found = true
		}
	})
	if !found {
		t.Errorf("flag was not set")
	}
}

type badvar1 struct{}

func (*badvar1) Var(string, string, int) {}

type badvar2 struct{}

func (*badvar2) Var(string, string, string, string) {}

type badvar3 struct{}

func (*badvar3) Var(string, string) {}

func TestSetVar(t *testing.T) {
	var x X
	if s := check.Error(setvar(&struct{}{}, &x, "X", ""), "Type *struct {} missing Var method"); s != "" {
		t.Error(s)
	}
	p1 := "Type *flags.badvar"
	p2 := " has the wrong signature for Var"
	if s := check.Error(setvar(&badvar1{}, &x, "X", ""), p1+"1"+p2); s != "" {
		t.Error(s)
	}
	if s := check.Error(setvar(&badvar1{}, &x, "X", ""), p1+"1"+p2); s != "" {
		t.Error(s)
	}
	if s := check.Error(setvar(&badvar2{}, &x, "X", ""), p1+"2"+p2); s != "" {
		t.Error(s)
	}
	if s := check.Error(setvar(&badvar3{}, &x, "X", ""), p1+"3"+p2); s != "" {
		t.Error(s)
	}
}

func TestLookup(t *testing.T) {
	opt := &struct {
		Ignore bool   `flag:"-"`
		Option string `flag:"--option=A_VERY_LONG_NAME some flag"`
		Lazy   string
	}{
		Option: "value",
		Lazy:   "lazy",
	}
	if o := Lookup(opt, "option"); o.(string) != "value" {
		t.Errorf("--option returned value %q, want %q", o, "value")
	}
	if o := Lookup(opt, "lazy"); o.(string) != "lazy" {
		t.Errorf("--lazy returned value %q, want %q", o, "lazy")
	}
	if o := Lookup("a", "a"); o != nil {
		t.Errorf("string returned %v, want nil", o)
	}
	if o := Lookup(new(string), "a"); o != nil {
		t.Errorf("*string returned %v, want nil", o)
	}
	if o := Lookup(opt, "missgin"); o != nil {
		t.Errorf("missing returned %v, want nil", o)
	}
	opt2 := &struct {
		Invalid string `flag:"invalid tag"`
		Option  string `flag:"--option"`
		Lazy    string
	}{
		Option: "value",
	}
	if o := Lookup(opt2, "option"); o != nil {
		t.Errorf("able to lookup past invalid tag")
	}
}

func checkPanic(p any, want string) string {
	if p == nil {
		return ""
	}
	switch t := p.(type) {
	case string:
		if t == want {
			return ""
		}
	case error:
		if t.Error() == want {
			return ""
		}
	default:
		panic(p)
	}
	return fmt.Sprintf("Got panic %s, want %s", p, want)

}

func TestValidate(t *testing.T) {
	func() {
		// This validate should pass
		opts := &struct {
			Name string `flag:"--the_name"`
		}{}
		Validate(opts)
	}()
	func() {
		defer func() {
			if s := checkPanic(recover(),
				`struct { Name string "flag:\"the_name\"" } is not a pointer to a struct`); s != "" {
				t.Error(s)
			}
		}()
		opts2 := struct {
			Name string `flag:"the_name"`
		}{}
		Validate(opts2)
	}()
}

func TestRegisterSet(t *testing.T) {
	opts := &struct {
		Name string `flag:"--the_name"`
	}{
		Name: "bob",
	}
	s := NewFlagSet("")
	RegisterSet("", opts, s)
	s.(*flag.FlagSet).VisitAll(func(f *flag.Flag) {
		if f.Name != "the_name" {
			t.Errorf("unexpected option found: %q", f.Name)
			return
		}
		if v := f.Value.String(); v != "bob" {
			t.Errorf("%s=%q, want %q", f.Name, v, "bob")
		}
	})
	s.Parse([]string{"--the_name", "fred"})
	s.(*flag.FlagSet).VisitAll(func(f *flag.Flag) {
		if f.Name != "the_name" {
			t.Errorf("unexpected option found: %q", f.Name)
			return
		}
		if v := f.Value.String(); v != "fred" {
			t.Errorf("%s=%q, want %q", f.Name, v, "fred")
		}
	})
}

func TestRegister(t *testing.T) {
	func() {
		defer func() {
			if s := checkPanic(recover(), "string is not a pointer to a struct"); s != "" {
				t.Error(s)
			}
		}()
		Register("a")
	}()
	func() {
		defer func() {
			if s := checkPanic(recover(), "*string is not a pointer to a struct"); s != "" {
				t.Error(s)
			}
		}()
		Register(new(string))
	}()
	func() {
		defer func() {
			if s := checkPanic(recover(), ""); s != "" {
				t.Error(s)
			}
		}()
		register("test", &struct {
			F int `flag:"bad"`
		}{}, NewFlagSet(""))
	}()
}

func TestMultiString(t *testing.T) {
	var opts struct {
		Value []string `flag:"--multi=VALUE help"`
		List  []string `flag:"--list=VALUE help"`
	}
	_, err := SubRegisterAndParse(&opts, []string{"name", "--multi", "value1", "--multi", "value2", "--list", "item1", "--list", "item2"})
	if err != nil {
		t.Fatal(err)
	}
	if len(opts.Value) != 2 {
		t.Errorf("got %d values, want 2", len(opts.Value))
	} else {
		if opts.Value[0] != "value1" {
			t.Errorf("got %s, want value1", opts.Value[0])
		}
		if opts.Value[1] != "value2" {
			t.Errorf("got %s, want value2", opts.Value[1])
		}
	}
	if len(opts.List) != 2 {
		t.Errorf("got %d values, want 2", len(opts.List))
	} else {
		if opts.List[0] != "item1" {
			t.Errorf("got %s, want item1", opts.List[0])
		}
		if opts.List[1] != "item2" {
			t.Errorf("got %s, want item2", opts.List[1])
		}
	}
}

func TestSubRegisterAndParse(t *testing.T) {
	var b bytes.Buffer
	output = &b
	defer func() { output = nil }()
	opts := struct {
		Value string `flag:"--the_name=VALUE help"`
	}{
		Value: "bob",
	}

	for _, tt := range []struct {
		args  []string
		err   string
		value string
		out   []string
	}{{
		args:  []string{"name"},
		value: "bob",
	}, {
		args:  []string{"name", "--the_name=fred"},
		value: "fred",
	}, {
		args:  []string{"name", "--the_name=fred", "a", "b", "c"},
		value: "fred",
		out:   []string{"a", "b", "c"},
	}, {
		value: "bob",
	}} {
		myopts := opts
		args, err := SubRegisterAndParse(&myopts, tt.args)
		if s := check.Error(err, tt.err); s != "" {
			t.Errorf("%s", s)
			continue
		}
		if len(args) == 0 {
			args = nil
		}
		if tt.value != myopts.Value {
			t.Errorf("%q got value %q, want %q", tt.args, myopts.Value, tt.value)
		}
		if !reflect.DeepEqual(tt.out, args) {
			t.Errorf("%q got args %#v, want %#v", tt.args, args, tt.out)
		}
	}
	_, err := SubRegisterAndParse(&struct{ N int16 }{}, []string{"c"})
	if s := check.Error(err, "invalid option type: int16"); s != "" {
		t.Error(s)
	}
	_, err = SubRegisterAndParse(&struct{}{}, []string{"c", "-v"})
	if s := check.Error(err, "flag provided but not defined: -v"); s != "" {
		t.Error(s)
	}
}

func TestParseTag(t *testing.T) {
	for _, tt := range []struct {
		name string
		in   string
		tag  *optTag
		str  string
		err  string
	}{
		{
			name: "nothing",
		},
		{
			name: "dash",
			in:   "-",
		},
		{
			name: "dash-dash",
			in:   "--",
		},
		{
			name: "long arg",
			in:   "--option",
			str:  "{ --option }",
			tag: &optTag{
				name: "option",
			},
		},
		{
			name: "short arg",
			in:   "-o",
			str:  "{ -o }",
			tag: &optTag{
				name: "o",
			},
		},
		{
			name: "long help",
			in:   "--option this is an option",
			str:  `{ --option "this is an option" }`,
			tag: &optTag{
				name: "option",
				help: "this is an option",
			},
		},
		{
			name: "long help1",
			in:   "--option -- this is an option",
			str:  `{ --option "this is an option" }`,
			tag: &optTag{
				name: "option",
				help: "this is an option",
			},
		},
		{
			name: "long help2",
			in:   "--option - this is an option",
			str:  `{ --option "this is an option" }`,
			tag: &optTag{
				name: "option",
				help: "this is an option",
			},
		},
		{
			name: "long help3",
			in:   "--option -- -this is an option",
			str:  `{ --option "-this is an option" }`,
			tag: &optTag{
				name: "option",
				help: "-this is an option",
			},
		},
		{
			name: "long arg with param",
			in:   "--option=PARAM",
			str:  "{ --option =PARAM }",
			tag: &optTag{
				name:  "option",
				param: "PARAM",
			},
		},
		{
			name: "everything",
			in:   "--option=PARAM -- - this is help",
			str:  `{ --option =PARAM "- this is help" }`,
			tag: &optTag{
				name:  "option",
				param: "PARAM",
				help:  "- this is help",
			},
		},
		{
			name: "two longs",
			in:   "--option1 --option2",
			err:  "tag has too many names",
		},
		{
			name: "two shorts",
			in:   "-a -b",
			err:  "tag has too many names",
		},
		{
			name: "two parms",
			in:   "--option=PARAM1 -o=PARAM2",
			err:  "tag has multiple parameter names",
		},
		{
			name: "missing option",
			in:   "no option",
			err:  "tag missing option name",
		},
		{
			name: "long param only",
			in:   "--=PARAM",
			err:  "tag missing option name",
		},
		{
			name: "short param only",
			in:   "-=PARAM",
			err:  "tag missing option name",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			tag, err := parseTag(tt.in)
			switch {
			case err == nil && tt.err != "":
				t.Fatalf("did not get expected error %v", tt.err)
			case err != nil && tt.err == "":
				t.Fatalf("unexpected error %v", err)
			case err == nil:
			case !strings.Contains(err.Error(), tt.err):
				t.Fatalf("got error %v, want %v", err, tt.err)
			}
			if !reflect.DeepEqual(tag, tt.tag) {
				t.Errorf("got %v, want %v", tag, tt.tag)
			}
			if tag != nil {
				str := tag.String()
				if str != tt.str {
					t.Errorf("%s: got string %q, want %q", tt.name, str, tt.str)
				}
			}
		})
	}
}

func TestArgPrefix(t *testing.T) {
	for _, tt := range []struct {
		in  string
		out string
	}{
		{"a", ""},
		{"-a", "-"},
		{"--a", "--"},
		{"", ""},
		{"-", "-"},
		{"--", "--"},
	} {
		if out := argPrefix(tt.in); out != tt.out {
			t.Errorf("argPrefix(%q) got %q want %q", tt.in, out, tt.out)
		}
	}
}

func TestDup(t *testing.T) {
	// Most of Dup is tested via other test methods.  We need to test the errors.
	func() {
		defer func() {
			p := recover()
			if p == nil {
				t.Errorf("Did not panic on string")
			}
		}()
		Dup("a")
	}()
	func() {
		defer func() {
			p := recover()
			if p == nil {
				t.Errorf("Did not panic on *string")
			}
		}()
		Dup(new(string))
	}()
	func() {
		defer func() {
			p := recover()
			if p == nil {
				t.Errorf("Did not panic on bad tag")
			}
		}()
		Dup(&struct {
			Opt bool `flag:"bad tag"`
		}{})
	}()

	// Test to make sure dup does not copy "-" values.
	type opt struct {
		Flag    string
		Private string `flag:"-"`
	}
	in := &opt{
		Flag:    "flag",
		Private: "private",
	}
	want := &opt{
		Flag: "flag",
	}
	got := Dup(in).(*opt)
	if got.Flag != want.Flag {
		t.Errorf("Got flag %q, want %q", got.Flag, want.Flag)
	}
	if got.Private != want.Private {
		t.Errorf("Got private %q, want %q", got.Private, want.Private)
	}
}

func TestParse(t *testing.T) {
	args, cl := os.Args, flag.CommandLine
	defer func() {
		os.Args, flag.CommandLine = args, cl
	}()
	CommandLine = NewFlagSet("")
	opts := &struct {
		Name string `geopt:"--name a name"`
	}{}
	Register(opts)
	os.Args = []string{"test", "--name", "bob", "arg"}
	pargs, err := Parse()
	if err != nil {
		t.Errorf("Got unexpected error: %v", err)
	}
	if opts.Name != "bob" {
		t.Errorf("Got name %q, want %q", opts.Name, "bob")
	}
	if len(pargs) != 1 || pargs[0] != "arg" {
		t.Errorf("Got args %q, want %q", pargs, []string{"arg"})
	}

	CommandLine.SetOutput(&bytes.Buffer{})
	os.Args = []string{"test", "--foo"}
	_, err = Parse()
	if err == nil {
		t.Errorf("Did not get an error on an invalid flag")
	}
}

func TestHelp(t *testing.T) {
	opts := &struct {
		Alpha   string  `flag:"--alpha=LEVEL set the alpha level"`
		Beta    int     `flag:"--beta=N      set beta to N"`
		Float   float64 `flag:"-f=RATE       set frame rate to RATE"`
		Fancy   bool    `flag:"--the_real_fancy_and_long_option yes or no"`
		Verbose bool    `flag:"-v            be verbose"`
		List    []string
	}{
		Alpha: "foo",
	}
	usage := `
Usage: xyzzy [--alpha=LEVEL] [--beta=N] [-f=RATE] [--list=VALUE] [--the_real_fancy_and_long_option] [-v] ...
`[1:]
	want := `
  --alpha=LEVEL    set the alpha level [foo]
  --beta=N         set beta to N
   -f=RATE         set frame rate to RATE
  --list=VALUE
  --the_real_fancy_and_long_option
                   yes or no
   -v              be verbose
`[1:]
	var out bytes.Buffer
	Help(&out, "xyzzy", "...", opts)
	got := out.String()
	if got != usage+want {
		t.Errorf("got:\n%s\nwant:\n%s", got, usage+want)
	}
	out.Reset()
	Help(&out, "", "", opts)
	got = out.String()
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
	out.Reset()
	Help(&out, "command", "args...", nil)
	got = out.String()
	want = "Usage: command args...\n"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}

	out.Reset()
	Help(&out, "command", "", nil)
	got = out.String()
	want = "Usage: command\n"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestAll(t *testing.T) {
	type options struct {
		List1    list
		List2    []string
		Duration time.Duration
		String   string
		Int      int
		Int64    int64
		Uint     uint
		Uint64   uint64
		Float    float64
		Bool     bool
	}
	opts := &options{}
	vnopts, fs := RegisterNew("all", opts)
	want := &options{
		List1:    []string{"a", "b"},
		List2:    []string{"c", "d"},
		Duration: 1200 * time.Millisecond,
		String:   "str",
		Int:      -42,
		Int64:    17,
		Uint:     7,
		Uint64:   13,
		Float:    1.425,
		Bool:     true,
	}
	got := vnopts.(*options)

	if err := fs.Parse([]string{"--list1", "a", "--list1", "b", "--list2", "c", "--list2", "d", "--duration", "1.2s", "--string", "str", "--int", "-42", "--int64", "17", "--uint", "7", "--uint64", "13", "--float", "1.425", "--bool"}); err != nil {
		t.Fatalf("Parsing: %v", err)
	}
	if !reflect.DeepEqual(got.List1, want.List1) {
		t.Errorf("List1: got %v, want %v", got.List1, want.List1)
	}
	if !reflect.DeepEqual(got.List2, want.List2) {
		t.Errorf("List2: got %v, want %v", got.List2, want.List2)
	}
	if !reflect.DeepEqual(got.Duration, want.Duration) {
		t.Errorf("Duration: got %v, want %v", got.Duration, want.Duration)
	}
	if !reflect.DeepEqual(got.String, want.String) {
		t.Errorf("String: got %v, want %v", got.String, want.String)
	}
	if !reflect.DeepEqual(got.Int, want.Int) {
		t.Errorf("Int: got %v, want %v", got.Int, want.Int)
	}
	if !reflect.DeepEqual(got.Int64, want.Int64) {
		t.Errorf("Int64: got %v, want %v", got.Int64, want.Int64)
	}
	if !reflect.DeepEqual(got.Uint, want.Uint) {
		t.Errorf("Uint: got %v, want %v", got.Uint, want.Uint)
	}
	if !reflect.DeepEqual(got.Uint64, want.Uint64) {
		t.Errorf("Uint64: got %v, want %v", got.Uint64, want.Uint64)
	}
	if !reflect.DeepEqual(got.Float, want.Float) {
		t.Errorf("Float: got %v, want %v", got.Float, want.Float)
	}
	if !reflect.DeepEqual(got.Bool, want.Bool) {
		t.Errorf("Bool: got %v, want %v", got.Bool, want.Bool)
	}
}

func TestError(t *testing.T) {
	for _, tt := range []struct {
		i     any
		panic string
	}{{
		i:     "foo",
		panic: "string is not a pointer to a struct",
	}, {
		i:     new(int),
		panic: "*int is not a pointer to a struct",
	}, {
		i: &struct {
			Int16 int16
		}{},
		panic: "invalid option type: int16",
	}} {
		t.Run(fmt.Sprintf("%v", tt.i), func(t *testing.T) {
			defer func() {
				if s := checkPanic(recover(), tt.panic); s != "" {
					t.Error(s)
				}

			}()
			Validate(tt.i)
		})
	}
}

func TestRegisterAndParse(t *testing.T) {
	var opts = &struct {
		Name string `flag:"--name"`
	}{}
	for _, tt := range []struct {
		in    []string
		name  string
		want  string
		index int // index into in for the args
		error string
	}{{
		name: "empty",
		in:   []string{},
	}, {
		name: "no flags",
		in:   []string{"a", "b"},
	}, {
		name:  "have flag",
		in:    []string{"--name", "bob", "a", "b"},
		index: 2,
		want:  "bob",
	}} {
		t.Run(fmt.Sprint(tt.in), func(t *testing.T) {
			defer func() {
				if s := checkPanic(recover(), tt.error); s != "" {
					t.Error(s)
				}
			}()
			opts.Name = ""
			CommandLine = NewFlagSet("")
			os.Args = append([]string{"command"}, tt.in...)
			args, err := RegisterAndParse(opts)
			if err != nil {
				t.Fatal(err)
			}
			if opts.Name != tt.want {
				t.Errorf("Got name %q, want %q", opts.Name, tt.want)
			}
			if !reflect.DeepEqual(args, tt.in[tt.index:]) {
				t.Errorf("Got %q, want %q", args, tt.in[tt.index])
			}
		})
	}
}

func TestUsageLine(t *testing.T) {
	opts := &struct {
		Name string
	}{}
	got := UsageLine("cmd", "param", opts)
	want := "cmd [--name=VALUE] param"
	if got != want {
		t.Errorf("Got %q, want %q", got, want)
	}
}

func TestGetInfo(t *testing.T) {
	u, i := getInfo(new(string), 10)
	if u != nil || i != 0 {
		t.Errorf("bad types got %v,%d want %v,%d", u, i, nil, 0)
	}
	opts := &struct {
		Missing int `flag:"-"`
		BadFlag int `flag:"-v -y"`
	}{}
	u, i = getInfo(opts, 10)
	if u != nil || i != 0 {
		t.Errorf("bad tag got %v,%d want %v,%d", u, i, nil, 0)
	}
}

func TestExtra(t *testing.T) {
	var opts = &struct {
		Name    string `flag:"--name"`
		Private string `flag:"-"`
		Int16   int16
	}{}
	defer func() {
		if s := checkPanic(recover(), "invalid option type: int16"); s != "" {
			t.Error(s)
		}
	}()
	RegisterNew("extra", opts)
}
