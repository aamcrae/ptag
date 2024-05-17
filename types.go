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

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
)

type nothing struct{}

type Exiv map[int]string

// Temporary data, present only when the image
// is loaded.
type Data struct {
	img *canvas.Image // Converted image
}

type Pict struct {
	state int    // Current state
	path  string // Filename of picture
	name  string // short name
	title string // window title
	index int
	win   fyne.Window
	size  fyne.Size
	err   error          // Error during loading
	lock  sync.WaitGroup // lock for loading
	exiv  Exiv           // Current EXIF data
	data  *Data          // Image data, nil if unloaded
}

type runner struct {
	app     fyne.App
	win     fyne.Window
	picts   []*Pict // Pictures
	size    fyne.Size
	index   int // Current picture
	preload int
	loaded  map[int]nothing
	visible bool
}

type event struct {
	event int
	w     int
	h     int
}
