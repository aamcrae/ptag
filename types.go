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
	"sync"

	"github.com/jezek/xgbutil"
	"github.com/jezek/xgbutil/xgraphics"
	"github.com/jezek/xgbutil/xrect"
	"github.com/jezek/xgbutil/xwindow"
)

type nothing struct{}

type Exiv map[int]string

// Temporary data, present only when the image
// is loaded.
type Data struct {
	rect      xrect.Rect       // Geometry of image
	clearList []xrect.Rect     // Areas to clear
	exiv      Exiv             // Current EXIF data
	img       *xgraphics.Image // Converted image
}

type Pict struct {
	state  int    // Current state
	path   string // Filename of picture
	name   string // short name
	index  int
	width  int
	height int
	err    error          // Error during loading
	lock   sync.WaitGroup // lock for loading
	X      *xgbutil.XUtil // X server connection
	data   *Data          // Image data, nil if unloaded
}

type runner struct {
	X       *xgbutil.XUtil  // X server connection
	win     *xwindow.Window // Display window
	geom    xrect.Rect      // Bounding box of window
	picts   []*Pict         // Pictures
	index   int             // Current picture
	preload int
}
