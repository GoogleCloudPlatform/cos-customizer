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

package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

// mapVar implements flag.Value for a map flag variable. Example:
// "-my-map a=A,b=B,c=C" results in {"a": "A", "b": "B", "c": "C"}
type mapVar struct {
	m map[string]string
}

// newMapVar returns an empty mapVar.
func newMapVar() *mapVar {
	return &mapVar{make(map[string]string)}
}

// String implements flag.Value.String.
func (mv *mapVar) String() string {
	mapJSON, _ := json.Marshal(mv.m)
	return string(mapJSON)
}

// Set implements flag.Value.Set. It parses the given string and adds the encoded map values to the mapVar.
func (mv *mapVar) Set(s string) error {
	pairs := strings.Split(s, ",")
	for _, pair := range pairs {
		split := strings.SplitN(pair, "=", 2)
		if len(split) != 2 {
			return fmt.Errorf("item %q is improperly formatted; does it have an '=' character?", pair)
		}
		mv.m[split[0]] = split[1]
	}
	return nil
}

// listVar implements flag.Value for a list flag variable. Example:
// "-my-list a,b,c,d" results in {"a", "b", "c", "d"}
type listVar struct {
	l []string
}

// String implements flag.Value.String.
func (lv *listVar) String() string {
	listJSON, _ := json.Marshal(lv.l)
	return string(listJSON)
}

// Set implements flag.Value.Set. It parses the given string and adds the encoded list values to the listVar.
func (lv *listVar) Set(s string) error {
	list := strings.Split(s, ",")
	lv.l = append(lv.l, list...)
	return nil
}
