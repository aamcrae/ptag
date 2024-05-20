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
var fullscreen = flag.Bool("fullscreen", false, "Fullscreen display")
var fit = flag.Bool("fit", false, "Scale images to fit window")
var maxPreload = flag.Int("preload", 10, "Maximum images to concurrently load")
var width = flag.Int("width", 1200, "Window width") // These are fyne sizes, not pixels
var height = flag.Int("height", 1000, "Window height")
var sidecar = flag.Bool("sidecar", false, "Use sidecar file for EXIF")

func main() {
	flag.Usage = usage
	flag.Parse()

	var f []string
	// No args, do all image files
	if len(flag.Args()) == 0 {
		f = expand([]string{"*.jpg", "*.jpeg", "*.tif", "*.tiff"})
	} else {
		f = expand(flag.Args())
	}
	// Limit the max preload count to the number of CPUs
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
	initExif()
	a, err := newPtag(*width, *height, preload)
	if err != nil {
		fmt.Fprintf(os.Stderr, "init: %v", err)
		return
	}
	a.start(f)
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, `
Shortcut keys are:
  'N' <right-arrow> <space>        Next image
  'P' <left-arrow> <back-space>    Previous image
  <Home>                           First image
  <End>                            Last image
  <down-arrow>                     Jump forward 10 images
  <up-arrow>                       Jump back 10 images
  -, 0, 1, 2, 3, 4, 5              Set the EXIF rating to this value [- delete]
  'F'                              Toggle full-screen
  'R'                              Rotate right 90 degrees
  'M'                              Mirror flip the image
  'Q'                              Quit
`)
}
