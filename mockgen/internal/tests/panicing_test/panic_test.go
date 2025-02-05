//go:build panictest
// +build panictest

// Copyright 2020 Google LLC
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

package paniccode

import (
	"testing"

	"github.com/pableeee/implgen/gomock"
)

func TestDanger_Panics_Explicit(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := NewMockFoo(ctrl)
	mock.EXPECT().Bar().Return("Bar")
	mock.EXPECT().Bar().Return("Baz")
	Danger(mock)
}

func TestDanger_Panics_Implicit(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := NewMockFoo(ctrl)
	mock.EXPECT().Bar().Return("Bar")
	mock.EXPECT().Bar().Return("Baz")
	Danger(mock)
}
