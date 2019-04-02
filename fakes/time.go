// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package fakes

import "time"

// Time is a fake implementation of the time package.
type Time struct {
	current time.Time
}

// NewTime gets a Time instance initialized with the given time.
func NewTime(t time.Time) *Time {
	return &Time{t}
}

// Now gets the current time.
func (t *Time) Now() time.Time {
	return t.current
}

// Sleep increments the current time by the given duration.
func (t *Time) Sleep(s time.Duration) {
	t.current = t.current.Add(s)
}
