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

// Package sdk provides api's with idiomatic approach
// to satisfy all interfaces of github.com/mkungla/happy package.
package create

import (
	"github.com/mkungla/happy"
	"github.com/mkungla/happy/app"
	"github.com/mkungla/happy/cli"
)

func App(options ...happy.Option) happy.Application {
	return app.New(options...)
}

func Command(name string, argsn uint) (happy.Command, error) {
	return cli.NewCommand(name, argsn)
}