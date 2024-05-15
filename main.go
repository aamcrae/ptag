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
	"runtime"
)

var verbose = flag.Bool("verbose", false, "Verbose tracing")
var maxPreload = flag.Int("preload", 10, "Maximum images to concurrently load")

func main() {
	flag.Parse()

	var f []string
	// No args, do all image files
	if len(flag.Args()) == 0 {
		f = expand([]string{"*.jpg", "*.jpeg", "*.tif", "*.tiff"})
	} else {
		f = expand(flag.Args())
	}
	if *verbose {
		fmt.Printf("files: %v\n", f)
	}
	var plist []*Pict
	for _, file := range f {
		p := NewPict(file)
		plist = append(plist, p)
	}
	preload := runtime.NumCPU()
	if preload > *maxPreload {
		preload = *maxPreload
	}
	if *verbose {
		fmt.Printf("%d files in total, preload = %d\n", len(plist), preload)
	}

	preloaded := 0
	for i, p := range plist {
		for preloaded < i+preload {
			if preloaded < len(plist) {
				plist[preloaded].startLoad(200, 300)
			}
			preloaded++
		}
		p.wait()
		if p.state == I_ERROR {
			fmt.Printf("%s: loading error: %v", p.name, p.err)
		} else {
			fmt.Printf("%s: img: %d x %d EXIV: %v\n", p.name, p.data.img.Bounds().Max.X, p.data.img.Bounds().Max.Y, p.data.exiv)
		}
		flush := i - preload
		if flush >= 0 {
			plist[flush].unload()
		}
	}
}