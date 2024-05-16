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
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path"

	"github.com/jezek/xgbutil"
	"github.com/jezek/xgbutil/xgraphics"
	"github.com/jezek/xgbutil/xwindow"
)

// Image state
const (
	I_UNLOADED = iota
	I_LOADING
	I_LOADED
	I_ERROR
)

func NewPict(file string, X *xgbutil.XUtil, index int) *Pict {
	_, f := path.Split(file)
	return &Pict{state: I_UNLOADED, path: file, name: f, X: X, index: index}
}

// wait waits for image loading to complete, and then
// checks the result.
func (p *Pict) wait() error {
	p.ready.Wait()
	if p.state == I_ERROR {
		return p.err
	}
	return nil
}

// startLoad sets up loading the image.
// The actual reading is delegated to a goroutine.
func (p *Pict) startLoad(w, h int) {
	p.wait() // Ensure not already loading
	if p.state == I_LOADED && w == p.width && h == p.height {
		return
	}
	// The image is either not loaded, or needs resizing.
	p.unload()
	p.width = w
	p.height = h
	if *verbose {
		fmt.Printf("%s (index %d): loading...\n", p.name, p.index)
	}
	p.ready.Add(1)
	p.state = I_LOADING
	go p.load()
}

// load reads and processes the image ready for display.
// wait() must be called before the
// image can be accessed.
func (p *Pict) load() {
	defer p.ready.Done()
	p.clean()
	f, err := os.Open(p.path)
	if err != nil {
		p.state = I_ERROR
		p.err = err
		return
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		p.state = I_ERROR
		p.err = err
		return
	}
	p.data = new(Data)
	p.data.img = xgraphics.NewConvert(p.X, img)
	// Determine the scaling necessary to fit within the window
	r := p.data.img.Bounds().Max
	xRatio := float64(p.width) / float64(r.X)
	yRatio := float64(p.height) / float64(r.Y)
	// If the image is larger than the window, scale it down
	if xRatio < 1 || yRatio < 1 {
		var w, h int
		if xRatio < yRatio {
			// Scale to width
			w = int(xRatio * float64(r.X))
			h = int(xRatio * float64(r.Y))
			p.x = 0
			p.y = (p.height - h) / 2
		} else {
			// Scale to height
			w = int(yRatio * float64(r.X))
			h = int(yRatio * float64(r.Y))
			p.x = (p.width - w) / 2
			p.y = 0
		}
		if *verbose {
			fmt.Printf("%s: scale to %d, %d, start %d, %d\n", p.name, w, h, p.x, p.y)
		}
		p.data.img = p.data.img.Scale(w, h)
	}
	// Create a pixmap and draw the image onto it.
	err = p.data.img.CreatePixmap()
	if err != nil {
		p.clean()
		p.state = I_ERROR
		p.err = err
		return
	}
	p.data.img.XDraw()
	p.data.exiv, err = getExiv(p.path)
	if err != nil {
		// We do allow an error when reading the EXIF.
		// This usually means there is no EXIF headers in the file
		p.data.exiv = make(Exiv)
	}
	p.state = I_LOADED
}

func (p *Pict) show(win *xwindow.Window) {
	if *verbose {
		fmt.Printf("showing image %s (index %d) at %d, %d\n", p.name, p.index, p.x, p.y)
	}
	p.data.img.XExpPaint(win.Id, p.x, p.y)
}

// unload clears out the image data and sets the picture to unloaded
func (p *Pict) unload() {
	if p.state != I_UNLOADED {
		if *verbose {
			fmt.Printf("Unloading %s, index %d\n", p.name, p.index)
		}
		p.wait() // Ensure not currently loading
		p.clean()
	}
}

func (p *Pict) clean() {
	if p.data != nil && p.data.img != nil {
		p.data.img.Destroy()
	}
	p.state = I_UNLOADED
	p.data = nil
}
