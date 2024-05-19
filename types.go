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
	"image"
	"image/draw"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
)

type nothing struct{}

type CaptionEntry struct {
	runner *runner
	widget.Entry
}

// Exiv holds a map of selected EXIF elements that
// we are interested in (rating and caption)
type Exiv map[int]string

// Cached image data.
type Data struct {
	location image.Rectangle   // Location and size of displayed image
	cleared  []image.Rectangle // Margins to be cleared
	img      image.Image       // Image to be displayed
}

// Pict represents one image.
type Pict struct {
	state int            // Current state
	path  string         // Filename of picture
	name  string         // short name
	title string         // window title
	index int            // Index within list of images
	err   error          // Error during loading
	lock  sync.WaitGroup // lock for loading
	exiv  Exiv           // Current EXIF data
	data  *Data          // Cached mage data, nil if unloaded
}

// Main execution runner. Holds the state of the application.
type runner struct {
	app     fyne.App          // Main application
	win     fyne.Window       // Main window
	rating  *canvas.Text      // widget holding rating stars
	caption *CaptionEntry     // Caption entry widget
	top     *fyne.Container   // top box containing stars and caption elements
	iDraw   draw.Image        // Image backing the canvas being displayed
	iCanvas fyne.CanvasObject // Canvas holding the displayed image
	picts   []*Pict           // List of images
	index   int               // Current picture index
	preload int               // Number of images to preload
	loaded  map[int]nothing   // Set of images that are cached
	active  bool              // True if window now active
	updated bool              // Set if the EXIF data may have changed
}
