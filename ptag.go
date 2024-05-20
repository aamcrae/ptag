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
	"os"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"

	"github.com/davidbyttow/govips/v2/vips"
)

// newPtag creates a new Ptag app
func newPtag(width, height, preload int) (*Ptag, error) {
	a := app.New()
	// Create a top level window to hold the elements of the app.
	win := a.NewWindow("ptag")
	win.SetMaster()
	if *fullscreen {
		win.SetFullScreen(true)
	} else {
		win.Resize(fyne.NewSize(float32(width), float32(height)))
	}
	return &Ptag{app: a, win: win, preload: preload, loaded: map[int]nothing{}}, nil
}

// start initialises the app and starts it.
func (a *Ptag) start(f []string) {
	// Make vips less noisy.
	vips.LoggingSettings(nil, vips.LogLevelError)
	vips.Startup(nil)
	// Create some containers for the layout.
	a.build()
	// Create a Pict object for every image
	for i, file := range f {
		p := NewPict(file, i)
		p.SetTitle(fmt.Sprintf("%s (%d/%d)", p.Name(), i+1, len(f)))
		a.picts = append(a.picts, p)
	}
	// Show the main window.
	a.win.Show()
	go a.resizeWatcher()
	// Main runloop.
	a.app.Run()
}

// Show the current image.
func (a *Ptag) show() {
	p := a.picts[a.index]
	defer a.win.SetTitle(p.Title())
	if err := p.Draw(a.iDraw); err != nil {
		fmt.Fprintf(os.Stderr, "%s: draw: %v", p.Name(), err)
		return
	}
	a.iCanvas.Refresh()
	if *verbose {
		fmt.Printf("%s (%d): Showing image, size %g, %g\n", p.Name(), a.index, a.iCanvas.Size().Width, a.iCanvas.Size().Height)
	}
	// If Draw worked, no error will be returned from Caption()
	capt, _ := p.Caption()
	if len(capt) != 0 {
		if *verbose {
			fmt.Printf("Initialising caption text to <%s>\n", capt)
		}
		a.caption.SetText(capt)
	} else {
		a.caption.SetText("")
		a.caption.SetPlaceHolder("Caption")
	}
	a.displayRating()
}

// build creates the elements that comprise the main window.
func (a *Ptag) build() {
	a.rating = canvas.NewText("Rating: -", color.Black)
	a.caption = &CaptionEntry{app: a}
	a.caption.ExtendBaseWidget(a.caption)
	a.caption.SetPlaceHolder("Caption")
	// Initially set up a dummy canvas in order for the
	// window to be shown and the sizes determined.
	// The resize watcher will detect when the window is
	// shown so that the first image can be loaded.
	a.iCanvas = canvas.NewRectangle(color.Black)
	a.top = container.NewBorder(nil, nil, a.rating, nil, a.caption)
	a.win.SetContent(container.NewBorder(a.top, nil, nil, nil, a.iCanvas))
	// Add key handlers
	if deskCanvas, ok := a.win.Canvas().(desktop.Canvas); ok {
		deskCanvas.SetOnKeyDown(func(key *fyne.KeyEvent) {
			if *verbose {
				fmt.Printf("Key: %s\n", key.Name)
			}
			switch key.Name {
			case fyne.KeyUp, fyne.KeyPageUp:
				a.setIndex(a.index - 10)
			case fyne.KeyDown, fyne.KeyPageDown:
				a.setIndex(a.index + 10)
			case "N", fyne.KeyRight, fyne.KeySpace:
				a.setIndex(a.index + 1)
			case "P", fyne.KeyLeft, fyne.KeyBackspace:
				a.setIndex(a.index - 1)
			case fyne.KeyHome:
				a.setIndex(0)
			case fyne.KeyEnd:
				a.setIndex(len(a.picts) - 1)
			case "M":
				a.mirror()
			case "R":
				a.rotate()
			case "F":
				a.fullScreen()
			case "Q":
				a.quit()
			case fyne.KeyMinus:
				a.rate(-1)
			case fyne.Key0:
				a.rate(0)
			case fyne.Key1:
				a.rate(1)
			case fyne.Key2:
				a.rate(2)
			case fyne.Key3:
				a.rate(3)
			case fyne.Key4:
				a.rate(4)
			case fyne.Key5:
				a.rate(5)
			}
		})
	}
}

// Updated flags that the EXIF data may have changed.
func (a *Ptag) Updated() {
	a.updated = true
}

// Sync writes the caption if it has changed.
func (a *Ptag) Sync() {
	if a.updated {
		a.updated = false
		p := a.picts[a.index]
		c, err := p.Caption()
		if err == nil {
			if c != a.caption.Text {
				if *verbose {
					fmt.Printf("%s (%d): update caption to <%s>\n", p.Name(), a.index, a.caption.Text)
				}
				p.SetCaption(a.caption.Text)
			}
		}
	}
}

// Window has been resized, so rescale all the images and redisplay the current one.
func (a *Ptag) resize() {
	sz := a.iCanvas.Size()
	scale := a.win.Canvas().Scale()
	if *verbose {
		fmt.Printf("resize to %g, %g, scale %g\n", sz.Width, sz.Height, scale)
	}
	// Create a new raster canvas for displaying the image. A raster canvas
	// maps the image pixels 1-1 to the canvas pixels.
	// This canvas is then used as the target for the image drawing.
	a.iDraw = image.NewRGBA(image.Rect(0, 0, int(sz.Width*scale), int(sz.Height*scale)))
	a.iCanvas = canvas.NewRasterFromImage(a.iDraw)
	a.win.SetContent(container.NewBorder(a.top, nil, nil, nil, a.iCanvas))
	// The first image to be displayed shows the window.
	if !a.active {
		a.active = true
		// Preload other images
		defer a.cacheUpdate()
	} else {
		a.flushCache()
	}
	a.redisplay()
}

// fullScreen toggles full screen mode
func (a *Ptag) fullScreen() {
	a.win.SetFullScreen(!a.win.FullScreen())
}

// quit exits the app
func (a *Ptag) quit() {
	a.Sync()
	vips.Shutdown()
	a.app.Quit()
}

// rotate the image 90 degrees clockwise
func (a *Ptag) rotate() {
	a.adjustOrientation(map[string]string{
		"":  "6", // No existing orientation
		"1": "6",
		"2": "5",
		"3": "8",
		"4": "7",
		"5": "4",
		"6": "3",
		"7": "2",
		"8": "1"})
}

// mirror the image
func (a *Ptag) mirror() {
	a.adjustOrientation(map[string]string{
		"":  "2", // No existing orientation
		"1": "2",
		"2": "1",
		"3": "4",
		"4": "3",
		"5": "6",
		"6": "5",
		"7": "8",
		"8": "7"})
}

// adjustOrientation selects a new orientation value using
// the current orientation as the key.
func (a *Ptag) adjustOrientation(adj map[string]string) {
	p := a.picts[a.index]
	if current, err := p.Orientation(); err != nil {
		fmt.Fprintf(os.Stderr, "%s: current orientation: %v", p.Name(), err)
	} else {
		newO, ok := adj[current]
		if !ok {
			fmt.Fprintf(os.Stderr, "%s: unknown orientation: %s", p.Name(), current)
		} else {
			p.SetOrientation(newO)
			a.redisplay()
			if *verbose {
				fmt.Printf("%s: old orientation %s, new orientation: %s\n", p.Name(), current, newO)
			}
		}
	}
}

// rate sets the rating on the current picture.
func (a *Ptag) rate(rating int) {
	p := a.picts[a.index]
	if err := p.SetRating(rating); err != nil {
		fmt.Fprintf(os.Stderr, "%s: Failed to set rating: %v", p.Name(), err)
	} else {
		a.displayRating()
	}
}

// displayRating updates the rating display
func (a *Ptag) displayRating() {
	p := a.picts[a.index]
	rating, err := p.Rating()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: rating: %v", p.Name(), err)
	}
	// Display rating.
	if *verbose {
		fmt.Printf("Displaying rating as %d\n", rating)
	}
	if rating < 0 {
		a.rating.Text = fmt.Sprintf("Rating: -")
	} else {
		a.rating.Text = fmt.Sprintf("Rating: %d", rating)
	}
	a.rating.Refresh()
}

// redisplay the current image, usually because something has changed
// such as orientation or size.
func (a *Ptag) redisplay() {
	a.removeCache(a.index)
	a.setIndex(a.index)
}

// setIndex selects the image to display.
func (a *Ptag) setIndex(newIndex int) {
	a.Sync()
	if newIndex < 0 {
		newIndex = 0
	}
	if newIndex >= len(a.picts) {
		newIndex = len(a.picts) - 1
	}
	a.index = newIndex
	a.addCache(a.index)
	a.show()
	a.cacheUpdate()
}

// cacheUpdate updates the cached set of images
func (a *Ptag) cacheUpdate() {
	// Map of items to cache
	nc := map[int]nothing{}
	// List of new items
	var newEntries []int
	count := a.preload + 1
	if count > len(a.picts) {
		count = len(a.picts)
	}
	f := func(index int) {
		if index >= 0 && index < len(a.picts) && count > 0 {
			if _, ok := a.loaded[index]; !ok {
				// new entry
				newEntries = append(newEntries, index)
			}
			// Flag item for caching
			nc[index] = nothing{}
			count--
		}
	}
	// Bias the preload going forwards
	start := a.index + a.preload/4
	before := start
	after := start + 1
	for count > 0 {
		f(before)
		f(after)
		before--
		after++
	}
	// Unload any items not in the new cache
	for k, _ := range a.loaded {
		if _, ok := nc[k]; !ok {
			a.removeCache(k)
		}
	}
	// Begin loading new items - we do this after unloading the expired entries
	for _, index := range newEntries {
		a.addCache(index)
	}
}

// flushCache removes all the items from the cache.
func (a *Ptag) flushCache() {
	for k, _ := range a.loaded {
		a.removeCache(k)
	}
}

// removeCache removes one image from the cache.
func (a *Ptag) removeCache(index int) {
	if _, ok := a.loaded[index]; ok {
		delete(a.loaded, index)
		a.picts[index].Unload()
	}
}

// addCache adds this image to the cache and initiates loading it.
func (a *Ptag) addCache(index int) {
	if _, ok := a.loaded[index]; !ok {
		a.loaded[index] = nothing{}
		a.picts[index].StartLoad(a.iDraw.Bounds().Max.X, a.iDraw.Bounds().Max.Y)
	}
}

// resizeWatcher tracks the actual size of the image canvas,
// and will force the images to be resized and redisplayed once
// the window has been resized.
func (a *Ptag) resizeWatcher() {
	sl := time.Millisecond * 50
	changed := 0
	lastC := a.iCanvas.Size()
	current := lastC
	for {
		time.Sleep(sl)
		if a.iCanvas.Size() != lastC {
			lastC = a.iCanvas.Size()
			// 250 ms delay before actioning resize
			changed = 5
		}
		if changed != 0 {
			changed--
			if changed == 0 {
				if *verbose {
					fmt.Printf("Canvas resize to %g, %g from %g, %g\n", a.iCanvas.Size().Width, a.iCanvas.Size().Height, current.Width, current.Height)
				}
				a.resize()
				current = lastC
			}
		}
	}
}
