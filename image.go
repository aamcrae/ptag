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
	"image/color"
	"image/draw"
	"path"

	"github.com/davidbyttow/govips/v2/vips"
)

// Image state. This should only be changed during the
// loading stage when the lock is active.
const (
	I_UNLOADED = iota
	I_LOADING
	I_LOADED
	I_ERROR
)

// Create a new Pict, representing an image from the file.
func NewPict(file string, index int) *Pict {
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

// startLoad sets up to load and process the image.
// The actual reading is delegated to a background goroutine.
// After calling startLoad, the wait function must be called before
// the image data is accessed.
func (p *Pict) StartLoad(w, h int) {
	p.wait() // Ensure loading is not already in progress
	// If loaded already, don't reload
	if p.state == I_LOADED {
		return
	}
	// The image is not loaded.
	p.Unload()
	if *verbose {
		fmt.Printf("%s (index %d): loading...\n", p.name, p.index)
	}
	p.lock.Add(1)
	p.state = I_LOADING
	go p.load(w, h)
}

// load reads and if necessary resizes the image ready for display.
// wait() must be called before the image can be accessed to
// ensure that the load is complete.
func (p *Pict) load(w, h int) {
	defer p.lock.Done()
	p.clean()
	// Read the EXIF data if it doesn't already exist
	// This is read first so that the orientation can be used
	// to flip the image if necessary.
	if p.exiv == nil {
		var err error
		p.exiv, err = getExiv(p.path)
		if err != nil {
			// We do allow an error when reading the EXIF.
			// This usually means there is no EXIF headers in the file
			p.exiv = make(Exiv)
			if *verbose {
				fmt.Printf("%s (%d): No exiv data!\n", p.name, p.index, p.exiv)
			}
		} else {
			if *verbose {
				fmt.Printf("%s (%d): exiv loaded: %v\n", p.name, p.index, p.exiv)
			}
		}
	}

	// Read the image from the file.
	vimg, err := vips.NewImageFromFile(p.path)
	if err != nil {
		p.state = I_ERROR
		p.err = err
		return
	}
	// Apply EXIF orientation (if any)
	adjustMap := map[string]struct {
		rotate vips.Angle
		flip   bool
	}{
		"1": {vips.Angle0, false},
		"2": {vips.Angle0, true},
		"3": {vips.Angle180, false},
		"4": {vips.Angle180, true},
		"5": {vips.Angle90, true},
		"6": {vips.Angle90, false},
		"7": {vips.Angle270, true},
		"8": {vips.Angle270, false},
	}
	// Get EXIF orientation, if any
	orient, ok := p.exiv[EXIV_ORIENTATION]
	if !ok {
		orient = "1" // No adjustment required
	}
	adjust, ok := adjustMap[orient]
	if ok {
		// Rotate before flip
		if adjust.rotate != vips.Angle0 {
			vimg.Rotate(adjust.rotate)
			if *verbose {
				fmt.Printf("%s (%d): rotating %v\n", p.name, p.index, adjust.rotate)
			}
		}
		if adjust.flip {
			vimg.Flip(vips.DirectionHorizontal)
			if *verbose {
				fmt.Printf("%s (%d): flipping\n", p.name, p.index)
			}
		}
	}
	iW := vimg.Width()
	iH := vimg.Height()
	// Scale the image to fit the requested size
	xRatio := float32(w) / float32(iW)
	yRatio := float32(h) / float32(iH)
	d := new(Data)
	// If the image is larger than the window, scale it down
	var x, y int
	if (*fit && (xRatio != 1 || yRatio != 1)) || (!*fit && (xRatio < 1 || yRatio < 1)) {
		// Maintain the same aspect, so use the same scaling factor for
		// both width and height. This may mean that there is blank space
		// on either the right/left or top/bottom.
		var err error
		if xRatio < yRatio {
			err = vimg.Resize(float64(xRatio), vips.KernelAuto)
			// Possible blank space at top and bottom,
			// calculate space in pixels
			y = (h - vimg.Height()) / 2

		} else {
			err = vimg.Resize(float64(yRatio), vips.KernelAuto)
			// Possible blank space at right and left
			x = (w - vimg.Width()) / 2
		}
		if err != nil {
			p.state = I_ERROR
			p.err = err
			return
		}
	} else {
		// The image is smaller than the canvas, so center it.
		x = (w - vimg.Width()) / 2
		y = (h - vimg.Height()) / 2
	}
	d.img, err = vimg.ToImage(vips.NewDefaultExportParams())
	if err != nil {
		p.state = I_ERROR
		p.err = err
		return
	}
	// If there are any surrounding margins, create a list of areas to be cleared.
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	d.location = image.Rect(x, y, x+d.img.Bounds().Max.X, y+d.img.Bounds().Max.Y)
	if y != 0 {
		d.cleared = append(d.cleared, image.Rect(0, 0, w, y))
	}
	if d.location.Max.Y != h {
		d.cleared = append(d.cleared, image.Rect(0, d.location.Max.Y, w, h))
	}
	if x != 0 {
		d.cleared = append(d.cleared, image.Rect(0, y, x, d.location.Max.Y))
	}
	if d.location.Max.X != w {
		d.cleared = append(d.cleared, image.Rect(d.location.Max.X, y, w, d.location.Max.Y))
	}
	if *verbose {
		fmt.Printf("%s (%d): Canvas %d x %d, Loaded size %d x %d, resized to %d, %d, pos %d, %d\n", p.name, p.index, w, h, iW, iH, d.img.Bounds().Max.X, d.img.Bounds().Max.Y, d.location.Min.X, d.location.Min.Y)
		for _, cl := range d.cleared {
			fmt.Printf("Clearing %d, %d to %d, %d\n", cl.Min.X, cl.Min.Y, cl.Max.X, cl.Max.Y)
		}
	}
	// Save the cached image data.
	p.data = d
	p.state = I_LOADED
}

// draw writes the image to the backing image of the canvas,
// and clears any surrounding area.
func (p *Pict) Draw(dst draw.Image) error {
	if err := p.wait(); err != nil {
		return err
	}
	d := p.data
	draw.Draw(dst, d.location, d.img, image.ZP, draw.Src)
	// Clear the margins.
	black := image.NewUniform(color.Black)
	for _, cl := range p.data.cleared {
		draw.Draw(dst, cl, black, image.ZP, draw.Src)
	}
	return nil
}

// Title returns the current title
func (p *Pict) Title() string {
	return p.title
}

// Name returns the current base filename
func (p *Pict) Name() string {
	return p.name
}

// setTitle sets the window title for this image.
func (p *Pict) SetTitle(title string) {
	p.title = title
}

// Rating returns the current rating, -1 if none
func (p *Pict) Rating() (int, error) {
	if err := p.wait(); err != nil {
		return 0, err
	}
	if r, ok := p.exiv[EXIV_RATING]; ok {
		var rating int
		n, err := fmt.Sscanf(r, "%d", &rating)
		if err != nil {
			return -1, err
		}
		if n != 1 || rating < 0 || rating > 5 {
			return -1, fmt.Errorf("%s: illegal rating")
		}
		return rating, nil
	} else {
		return -1, nil
	}
}

// SetRating sets a rating (0-5) on this image.
// -1 will delete the rating
func (p *Pict) SetRating(rating int) error {
	if err := p.wait(); err != nil {
		return err
	}
	if *verbose {
		fmt.Printf("Set rating of %s to %d\n", p.name, rating)
	}
	if rating < 0 {
		if err := deleteExiv(p.path, Exiv{EXIV_RATING: ""}); err != nil {
			return err
		}
		delete(p.exiv, EXIV_RATING)
		return nil
	}
	if rating > 5 {
		return fmt.Errorf("%d: illegal rating", rating)
	}
	sr := fmt.Sprintf("%d", rating)
	err := setExiv(p.path, Exiv{EXIV_RATING: sr})
	if err == nil {
		// Update the current values
		p.exiv[EXIV_RATING] = sr
	}
	return nil
}

// Orientation returns the current orientation, "" if none
func (p *Pict) Orientation() (string, error) {
	if err := p.wait(); err != nil {
		return "", err
	}
	if r, ok := p.exiv[EXIV_ORIENTATION]; ok {
		return r, nil
	} else {
		return "", nil
	}
}

// SetOrientation sets an orientation ("1" - "8") on this image.
// "" will delete the rating
func (p *Pict) SetOrientation(orientation string) error {
	if err := p.wait(); err != nil {
		return err
	}
	if *verbose {
		fmt.Printf("Set orientation of %s to %s\n", p.name, orientation)
	}
	if orientation == "" {
		if err := deleteExiv(p.path, Exiv{EXIV_ORIENTATION: ""}); err != nil {
			return err
		}
		delete(p.exiv, EXIV_ORIENTATION)
		return nil
	}
	err := setExiv(p.path, Exiv{EXIV_ORIENTATION: orientation})
	if err == nil {
		// Update the current values
		p.exiv[EXIV_ORIENTATION] = orientation
	}
	return nil
}

// Caption returns the current caption (if any)
func (p *Pict) Caption() (string, error) {
	if err := p.wait(); err != nil {
		return "", err
	}
	if r, ok := p.exiv[EXIV_CAPTION]; ok {
		return r, nil
	} else {
		return "", nil
	}
}

// SetCaption sets a caption on the EXIF.
// An empty caption will delete the caption
func (p *Pict) SetCaption(caption string) error {
	if err := p.wait(); err != nil {
		return err
	}
	if *verbose {
		fmt.Printf("Set caption of %s to %s\n", p.name, caption)
	}
	if len(caption) == 0 {
		if err := deleteExiv(p.path, Exiv{EXIV_CAPTION: ""}); err != nil {
			return err
		}
		delete(p.exiv, EXIV_CAPTION)
		return nil
	}
	err := setExiv(p.path, Exiv{EXIV_CAPTION: caption})
	if err == nil {
		// Update the current values
		p.exiv[EXIV_CAPTION] = caption
	}
	return nil
}

// unload clears out the image data and sets the picture to unloaded.
func (p *Pict) Unload() {
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
