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
	"github.com/jezek/xgbutil/xrect"
	"github.com/jezek/xgbutil/xwindow"
)

// Image state. This should only be changed during the locked
// loading stage.
const (
	I_UNLOADED = iota
	I_LOADING
	I_LOADED
	I_ERROR
)

// Create a new Pict, representing an image.
func NewPict(file string, X *xgbutil.XUtil, index int) *Pict {
	_, f := path.Split(file)
	return &Pict{state: I_UNLOADED, path: file, name: f, X: X, index: index}
}

// wait waits for image loading to complete, and then
// checks the result, returning any error found during the image load.
func (p *Pict) wait() error {
	p.lock.Wait()
	if p.state == I_ERROR {
		return p.err
	}
	return nil
}

// startLoad sets up to load the image.
// The actual reading is delegated to a background goroutine.
func (p *Pict) startLoad(rect xrect.Rect) {
	p.wait() // Ensure not already loading
	// If loaded already, and scaled to match the current window size, don't reload
	if p.state == I_LOADED && rect.Width() == p.width && rect.Height() == p.height {
		return
	}
	// The image is either not loaded, or needs resizing.
	p.width = rect.Width()
	p.height = rect.Height()
	p.unload()
	if *verbose {
		fmt.Printf("%s (index %d): loading...\n", p.name, p.index)
	}
	p.lock.Add(1)
	p.state = I_LOADING
	go p.load(rect)
}

// load reads and processes the image ready for display.
// wait() must be called before the image can be accessed to
// ensure that the state is valid.
func (p *Pict) load(winGeom xrect.Rect) {
	defer p.lock.Done()
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
	var w, h, x, y int
	// If the image is larger than the window, scale it down
	if xRatio < 1 || yRatio < 1 {
		// Maintain the same aspect, so use the same scaling factor for
		// both width and height. This may mean that there is blank space
		// on either the right/left or top/bottom.
		if xRatio < yRatio {
			// Scale to width
			w = int(xRatio * float64(r.X))
			h = int(xRatio * float64(r.Y))
			x = 0
			y = (p.height - h) / 2
		} else {
			// Scale to height
			w = int(yRatio * float64(r.X))
			h = int(yRatio * float64(r.Y))
			x = (p.width - w) / 2
			y = 0
		}
		p.data.img = p.data.img.Scale(w, h)
	}
	p.data.rect = xrect.New(x, y, w, h)
	// Build a list of rectangles to be cleared (if any)
	p.data.clearList = xrect.Subtract(winGeom, p.data.rect)
	if *verbose {
		fmt.Printf("%s (%d): scale to %d, %d, start %d, %d\n", p.name, p.index, w, h, x, y)
		for _, cr := range p.data.clearList {
			fmt.Printf("clear: (%d, %d) [%d, %d]\n", cr.X(), cr.Y(), cr.Width(), cr.Height())
		}
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
	// Image processing is done. Now read the EXIF data if it
	// doesn't already exist
	if p.exiv == nil {
		p.exiv, err = getExiv(p.path)
		if err != nil {
			// We do allow an error when reading the EXIF.
			// This usually means there is no EXIF headers in the file
			p.exiv = make(Exiv)
		}
		if *verbose {
			fmt.Printf("%s (%d): exiv loaded: %v\n", p.name, p.index, p.exiv)
		}
	}
	p.state = I_LOADED
}

func (p *Pict) show(win *xwindow.Window) {
	if *verbose {
		fmt.Printf("showing image %s (index %d) at %d, %d\n", p.name, p.index, p.data.rect.X(), p.data.rect.Y())
	}
	// Write the image and clear the areas around it.
	p.data.img.XExpPaint(win.Id, p.data.rect.X(), p.data.rect.Y())
	for _, cr := range p.data.clearList {
		win.Clear(cr.X(), cr.Y(), cr.Width(), cr.Height())
	}
	if *verbose {
		for k, v := range p.exiv {
			fmt.Printf("%s = %s\n", exivToSet[k], v)
		}
	}
}

func (p *Pict) setTitle(title string) {
	p.title = title
}

func (p *Pict) setRating(rating int) error {
	if *verbose {
		fmt.Printf("Set rating of %s to %d\n", p.name, rating)
	}
	sr := fmt.Sprintf("%d", rating)
	err := setExiv(p.path, Exiv{EXIV_RATING: sr})
	if err == nil {
		// Update the current values
		p.exiv[EXIV_RATING] = sr
	}
	return nil
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
