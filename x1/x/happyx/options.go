// Copyright 2022 The Happy Authors
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
// limitations under the License.

package happyx

import (
	"github.com/mkungla/happy"
	"github.com/mkungla/happy/x/pkg/vars"
)

func OptionFunc(vfunc happy.VariableParseFunc) happy.OptionSetFunc {
	return func(opts happy.OptionSetter) happy.Error {
		if vfunc == nil {
			return NewError("happyx.OptionFunc got <nil> arg")
		}
		v, err := vfunc()
		if err != nil {
			return err
		}
		return opts.SetOption(v)
	}
}

func Option(k string, v any) happy.OptionSetFunc {
	return OptionFunc(func() (happy.Variable, happy.Error) {
		var err happy.Error
		vv, e := vars.NewVariable(k, v, false)
		if e != nil {
			err = Errorf("option error: %w", e)
			return nil, err
		}

		// var vvv happy.Variable
		return vars.AsVariable[happy.Variable, happy.Value](vv), nil
	})
}

func ReadOnlyOption(k string, v any) happy.OptionSetFunc {
	return OptionFunc(func() (happy.Variable, happy.Error) {
		var err happy.Error
		vv, e := vars.NewVariable(k, v, true)
		if e != nil {
			err = Errorf("option error: %w", e)
			return nil, err
		}

		// var vvv happy.Variable
		return vars.AsVariable[happy.Variable, happy.Value](vv), nil
	})
}

type optreader struct {
	v happy.Variable
}

func (r *optreader) SetOption(v happy.Variable) happy.Error {
	r.v = v
	return nil
}

func (r *optreader) SetOptionKeyValue(key string, val any) happy.Error {
	v, err := vars.NewVariable(key, val, true)
	if err != nil {
		return ErrOption.Wrap(err)
	}
	r.v = vars.AsVariable[happy.Variable, happy.Value](v)
	return nil
}

func (r *optreader) SetOptionValue(key string, val happy.Value) happy.Error {
	// r.v = vars.NewVariable(key, val, true)
	return r.SetOptionKeyValue(key, val)
}

func OptionParseFuncFor(f happy.OptionSetFunc) happy.VariableParseFunc {
	return func() (happy.Variable, happy.Error) {
		r := new(optreader)
		err := f(r)
		return r.v, err
	}
}

func OptionsToVariables(option ...happy.OptionSetFunc) (happy.Variables, happy.Error) {
	opts := vars.AsMap[happy.Variables, happy.Variable, happy.Value](new(vars.Map))
	for _, ofunc := range option {
		o, err := OptionParseFuncFor(ofunc)()
		if err != nil {
			return nil, err
		}
		opts.Store(o.Key(), o.Value())
	}
	return opts, nil
}