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
)

type nothing struct{}

type Exiv map[int]string

// Temporary data, present only when the image
// is loaded.
type Data struct {
	exiv Exiv             // Current EXIF data
	img  *xgraphics.Image // Converted image
}

type Pict struct {
	state         int            // Current state
	name          string         // Filename of picture
	err           error          // Error during loading
	ready         sync.WaitGroup // lock for loading
	X             *xgbutil.XUtil // X server connection
	width, height int            // Window size
	data          *Data          // Image data, nil if unloaded
}
