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
	"path"

	"fyne.io/fyne/v2"

	"github.com/davidbyttow/govips/v2/vips"
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
func NewPict(file string, win fyne.Window, index int) *Pict {
	_, f := path.Split(file)
	return &Pict{state: I_UNLOADED, path: file, name: f, index: index}
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
func (p *Pict) startLoad(sz fyne.Size) {
	p.wait() // Ensure not already loading
	// If loaded already, and scaled to match the current window size, don't reload
	if p.state == I_LOADED {
		return
	}
	// The image is not loaded.
	p.unload()
	if *verbose {
		fmt.Printf("%s (index %d): loading...\n", p.name, p.index)
	}
	p.lock.Add(1)
	p.state = I_LOADING
	go p.load(sz)
}

// load reads and processes the image ready for display.
// wait() must be called before the image can be accessed to
// ensure that the load is complete.
func (p *Pict) load(sz fyne.Size) {
	defer p.lock.Done()
	p.clean()
	// Read the image from the file.
	vimg, err := vips.NewImageFromFile(p.path)
	if err != nil {
		p.state = I_ERROR
		p.err = err
		return
	}
	// Scale the image to fit the requested size
	xRatio := sz.Width / float32(vimg.Width())
	yRatio := sz.Height / float32(vimg.Height())
	d := new(Data)
	// If the image is larger than the window, scale it down
	var x, y int
	if xRatio < 1 || yRatio < 1 {
		// Maintain the same aspect, so use the same scaling factor for
		// both width and height. This may mean that there is blank space
		// on either the right/left or top/bottom.
		var err error
		if xRatio < yRatio {
			err = vimg.Resize(float64(xRatio), vips.KernelAuto)
			// Possible blank space at top and bottom
			h := int(sz.Height)
			y = (h - vimg.Height()) / 2
			if y > 0 {
				d.cleared = []image.Rectangle{image.Rect(0, 0, vimg.Width(), y), image.Rect(0, y+vimg.Height(), vimg.Width(), h)}
			} else {
				y = 0
			}

		} else {
			err = vimg.Resize(float64(yRatio), vips.KernelAuto)
			// Possible blank space at right and left
			w := int(sz.Width)
			x = (w - vimg.Width()) / 2
			if x > 0 {
				d.cleared = []image.Rectangle{image.Rect(0, 0, x, vimg.Height()), image.Rect(x+vimg.Width(), 0, w, vimg.Height())}
			} else {
				x = 0
			}
		}
		if err != nil {
			p.state = I_ERROR
			p.err = err
			return
		}
	}
	d.img, err = vimg.ToImage(vips.NewDefaultExportParams())
	if err != nil {
		p.state = I_ERROR
		p.err = err
		return
	}
	d.location = image.Rect(x, y, x+d.img.Bounds().Max.X, y+d.img.Bounds().Max.Y)
	if *verbose {
		fmt.Printf("%s (%d): Loaded, resized to %d, %d, pos %d, %d\n", p.name, p.index, d.img.Bounds().Max.X, d.img.Bounds().Max.Y, d.location.Min.X, d.location.Min.Y)
		for _, cl := range d.cleared {
			fmt.Printf("Clearing %d, %d to %d, %d\n", cl.Min.X, cl.Min.Y, cl.Max.X, cl.Max.Y)
		}
	}
	p.data = d
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

// show writes the image to the window.
func (p *Pict) show(win fyne.Canvas) {
	if *verbose {
		fmt.Printf("showing image %s (index %d)\n", p.name, p.index)
	}
	// Write the image.
	//win.SetContent(p.img)
	if *verbose {
		fmt.Printf("show: %v\n", win.Content().Size())
		for k, v := range p.exiv {
			fmt.Printf("%s = %s\n", exivToSet[k], v)
		}
	}
}

// setTitle sets the window title for this image.
func (p *Pict) setTitle(title string) {
	p.title = title
}

// setRating sets a rating (0-5) on this image.
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

// unload clears out the image data and sets the picture to unloaded.
func (p *Pict) unload() {
	if p.state != I_UNLOADED {
		if *verbose {
			fmt.Printf("Unloading %s, index %d\n", p.name, p.index)
		}
		p.wait() // If loading, wait for the load to complete before clearing.
		p.clean()
	}
}

// clean unloads the image to free the memory.
func (p *Pict) clean() {
	p.state = I_UNLOADED
	p.data = nil
}
