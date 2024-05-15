// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func expand(g []string) []string {
	var f []string
	// Map to check for existing file (skipped).
	m := make(map[string]nothing)
	for _, fp := range g {
		files, err := filepath.Glob(fp)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v (ignored)\n", fp, err)
			continue
		}
		for _, fn := range files {
			_, ok := m[fn]
			if !ok {
				// remember filename
				m[fn] = nothing{}
				f = append(f, fn)
			} else {
				if *verbose {
					fmt.Printf("%s: duplicate, skipped\n", fn)
				}
			}
		}
	}
	return f
}
