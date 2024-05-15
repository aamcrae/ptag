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
)

const (
	I_UNLOADED = iota
	I_LOADING
	I_LOADED
	I_ERROR
)

func NewPict(f string) *Pict {
	return &Pict{state: I_UNLOADED, name: f}
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
		fmt.Printf("%s: loading...\n", p.name)
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
	p.data = nil
	f, err := os.Open(p.name)
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
	p.data.img = img
	p.data.exiv, err = getExiv(p.name)
	if err != nil {
		// We do allow an on reading the EXIF, which
		// usually means there is no EXIF data on the file
		p.data.exiv = make(Exiv)
	}
	p.state = I_LOADED
}

// unload clears out the image data and sets the picture to unloaded
func (p *Pict) unload() {
	p.wait() // Ensure not already loading
	p.state = I_UNLOADED
	p.data = nil
}
