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
	"flag"
	"fmt"
	"os"
	"runtime"
)

var verbose = flag.Bool("verbose", false, "Verbose tracing")
var maxPreload = flag.Int("preload", 10, "Maximum images to concurrently load")
var width = flag.Int("width", 1500, "Window width")
var height = flag.Int("height", 1200, "Window height")

func main() {
	flag.Parse()

	var f []string
	// No args, do all image files
	if len(flag.Args()) == 0 {
		f = expand([]string{"*.jpg", "*.jpeg", "*.tif", "*.tiff"})
	} else {
		f = expand(flag.Args())
	}
	preload := runtime.NumCPU()
	if preload > *maxPreload {
		preload = *maxPreload
	}
	if len(f) == 0 {
		fmt.Printf("No files to display")
		return
	}
	if *verbose {
		fmt.Printf("%d files in total, preload = %d\n", len(f), preload)
	}
	r, err := newRunner(*width, *height, preload)
	if err != nil {
		fmt.Fprintf(os.Stderr, "init: %v", err)
		return
	}
	r.start(f)
}
