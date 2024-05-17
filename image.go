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
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
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
	return &Pict{state: I_UNLOADED, path: file, name: f, win: win, index: index}
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
	p.size = sz
	// The image is not loaded.
	p.unload()
	if *verbose {
		fmt.Printf("%s (index %d): loading...\n", p.name, p.index)
	}
	p.lock.Add(1)
	p.state = I_LOADING
	go p.load()
}

// load reads and processes the image ready for display.
// wait() must be called before the image can be accessed to
// ensure that the state is valid.
func (p *Pict) load() {
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
	now := time.Now()
	p.data = new(Data)
	p.data.img = canvas.NewImageFromImage(img)
	p.data.img.ScaleMode = canvas.ImageScaleFastest
	if *prescale {
		p.data.img.FillMode = canvas.ImageFillContain
		p.data.img.Resize(p.size)
	} else {
		p.data.img.FillMode = canvas.ImageFillContain
	}
	fmt.Printf("image processing time = %d us\n", time.Now().Sub(now).Microseconds())
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

func (p *Pict) show(win fyne.Window) {
	if *verbose {
		fmt.Printf("showing image %s (index %d)\n", p.name, p.index)
	}
	// Write the image.
	now := time.Now()
	win.SetContent(p.data.img)
	fmt.Printf("image show time = %d us\n", time.Now().Sub(now).Microseconds())
	win.SetTitle(p.title)
	if *verbose {
		fmt.Printf("show: %v\n", win.Content().Size())
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
	}
	p.state = I_UNLOADED
	p.data = nil
}
