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
	"fyne.io/fyne/v2/widget"

	"github.com/davidbyttow/govips/v2/vips"
)

func newRunner(width, height, preload int) (*runner, error) {
	a := app.New()
	// Create a window to display the images.
	win := a.NewWindow("ptag")
	win.SetMaster()
	if *fullscreen {
		win.SetFullScreen(true)
	} else {
		win.Resize(fyne.NewSize(float32(width), float32(height)))
	}
	return &runner{app: a, win: win, preload: preload, loaded: map[int]nothing{}}, nil
}

func (r *runner) start(f []string) {
	// Make vips less noisy.
	vips.LoggingSettings(nil, vips.LogLevelError)
	vips.Startup(nil)
	// Create some containers for the layout.
	r.build()
	// Create a pict structure for every image
	for i, file := range f {
		p := NewPict(file, i)
		p.setTitle(fmt.Sprintf("%s (%d/%d)", p.name, i+1, len(f)))
		r.picts = append(r.picts, p)
	}
	r.win.Show()
	go r.resizeWatcher()
	// Main runloop.
	r.app.Run()
}

// Show the current image from the cache.
func (r *runner) show() {
	p := r.picts[r.index]
	defer r.win.SetTitle(p.title)
	err := p.wait()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: load err: %v", p.name, err)
		return
	}
	p.draw(r.iDraw)
	r.iCanvas.Refresh()
	if *verbose {
		fmt.Printf("%s (%d): Showing size %g, %g\n", p.name, r.index, r.iCanvas.Size().Width, r.iCanvas.Size().Height)
	}
	if capt, ok := p.exiv[EXIV_CAPTION]; ok {
		if *verbose {
			fmt.Printf("Setting caption text to <%s>\n", capt)
		}
		r.caption.SetText(capt)
	} else {
		r.caption.SetText("")
		r.caption.SetPlaceHolder("Caption")
	}
}

func (r *runner) build() {
	r.rating = canvas.NewRectangle(color.Black)
	r.rating.SetMinSize(fyne.NewSize(100, 0))
	r.caption = widget.NewEntry()
	r.caption.SetPlaceHolder("Caption is here")
	// Initially set up a dummy canvas in order for the
	// window to be shown and the sizes determined.
	// The resize watcher will detect when the window is
	// shown so that the first image can be loaded.
	r.iCanvas = canvas.NewRectangle(color.Black)
	r.top = container.NewBorder(nil, nil, r.rating, nil, r.caption)
	r.win.SetContent(container.NewBorder(r.top, nil, nil, nil, r.iCanvas))
	// Add key handlers
	if deskCanvas, ok := r.win.Canvas().(desktop.Canvas); ok {
		deskCanvas.SetOnKeyDown(func(key *fyne.KeyEvent) {
			if *verbose {
				fmt.Printf("Key: %s\n", key.Name)
			}
			switch key.Name {
			case fyne.KeyUp, fyne.KeyPageUp:
				r.setIndex(r.index - 10)
			case fyne.KeyDown, fyne.KeyPageDown:
				r.setIndex(r.index + 10)
			case "N", fyne.KeyRight, fyne.KeySpace:
				r.setIndex(r.index + 1)
			case "P", fyne.KeyLeft, fyne.KeyBackspace:
				r.setIndex(r.index - 1)
			case fyne.KeyHome:
				r.setIndex(0)
			case fyne.KeyEnd:
				r.setIndex(len(r.picts) - 1)
			case "F":
				r.fullScreen()
			case "Q":
				r.quit()
			case fyne.Key0:
				r.rate(0)
			case fyne.Key1:
				r.rate(1)
			case fyne.Key2:
				r.rate(2)
			case fyne.Key3:
				r.rate(3)
			case fyne.Key4:
				r.rate(4)
			case fyne.Key5:
				r.rate(5)
			}
		})
	}
}

// Window has been resized, so rescale all the images and redisplay the current one.
func (r *runner) resize() {
	sz := r.iCanvas.Size()
	scale := r.win.Canvas().Scale()
	if *verbose {
		fmt.Printf("resize to %g, %g, scale %g\n", sz.Width, sz.Height, scale)
	}
	// Create a new raster canvas for displaying the image
	r.iDraw = image.NewRGBA(image.Rect(0, 0, int(sz.Width*scale), int(sz.Height*scale)))
	r.iCanvas = canvas.NewRasterFromImage(r.iDraw)
	r.win.SetContent(container.NewBorder(r.top, nil, nil, nil, r.iCanvas))
	// The first image to be displayed shows the window.
	if !r.active {
		r.active = true
		// Preload other images
		defer r.cacheUpdate()
	} else {
		r.flushCache()
	}
	r.addCache(r.index)
	r.show()
}

// fullScreen toggles full screen mode
func (r *runner) fullScreen() {
	r.win.SetFullScreen(!r.win.FullScreen())
}

// quit exits the app
func (r *runner) quit() {
	vips.Shutdown()
	r.app.Quit()
}

// rate sets the rating on the current picture
func (r *runner) rate(rating int) {
	p := r.picts[r.index]
	err := p.wait()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: load err: %v", p.name, err)
		return
	}
	err = p.setRating(rating)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: Failed to set rating: %v", r.picts[r.index].name, err)
	}
}

// setIndex selects the image to display.
func (r *runner) setIndex(newIndex int) {
	if newIndex < 0 {
		newIndex = 0
	}
	if newIndex >= len(r.picts) {
		newIndex = len(r.picts) - 1
	}
	r.index = newIndex
	r.addCache(r.index)
	r.show()
	r.cacheUpdate()
}

// cacheUpdate updates the cached set of images
func (r *runner) cacheUpdate() {
	// Map of items to cache
	nc := map[int]nothing{}
	// List of new items
	var newEntries []int
	count := r.preload + 1
	if count > len(r.picts) {
		count = len(r.picts)
	}
	f := func(index int) {
		if index >= 0 && index < len(r.picts) && count > 0 {
			if _, ok := r.loaded[index]; !ok {
				// new entry
				newEntries = append(newEntries, index)
			}
			// Flag item for caching
			nc[index] = nothing{}
			count--
		}
	}
	// Bias the preload going forwards
	start := r.index + r.preload/4
	before := start
	after := start + 1
	for count > 0 {
		f(before)
		f(after)
		before--
		after++
	}
	// Unload any items not in the new cache
	for k, _ := range r.loaded {
		if _, ok := nc[k]; !ok {
			r.removeCache(k)
		}
	}
	// Begin loading new items - we do this after unloading the expired entries
	for _, index := range newEntries {
		r.addCache(index)
	}
}

// flushCache removes all the items from the cache.
func (r *runner) flushCache() {
	for k, _ := range r.loaded {
		r.removeCache(k)
	}
}

// removeCache removes one image from the cache.
func (r *runner) removeCache(index int) {
	if _, ok := r.loaded[index]; ok {
		delete(r.loaded, index)
		r.picts[index].unload()
	}
}

// addCache adds this image to the cache and initiates loading it.
func (r *runner) addCache(index int) {
	if _, ok := r.loaded[index]; !ok {
		r.loaded[index] = nothing{}
		r.picts[index].startLoad(r.iDraw.Bounds().Max.X, r.iDraw.Bounds().Max.Y)
	}
}

// resizeWatcher tracks the actual size of the image canvas,
// and will force the images to be resized and redisplayed once
// the window has been resized.
func (r *runner) resizeWatcher() {
	sl := time.Millisecond * 50
	changed := 0
	lastC := r.iCanvas.Size()
	current := lastC
	for {
		time.Sleep(sl)
		if r.iCanvas.Size() != lastC {
			lastC = r.iCanvas.Size()
			// 250 ms delay before actioning resize
			changed = 5
		}
		if changed != 0 {
			changed--
			if changed == 0 {
				if *verbose {
					fmt.Printf("Canvas resize to %g, %g from %g, %g\n", r.iCanvas.Size().Width, r.iCanvas.Size().Height, current.Width, current.Height)
				}
				r.resize()
				current = lastC
			}
		}
	}
}
