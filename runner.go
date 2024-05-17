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
	"os"

	"github.com/jezek/xgb/xproto"

	"github.com/jezek/xgbutil"
	"github.com/jezek/xgbutil/ewmh"
	"github.com/jezek/xgbutil/keybind"
	"github.com/jezek/xgbutil/xevent"
	"github.com/jezek/xgbutil/xwindow"
)

func newRunner(width, height, preload int) (*runner, error) {
	X, err := xgbutil.NewConn()
	if err != nil {
		return nil, err
	}
	keybind.Initialize(X)
	// Create a window
	win, err := xwindow.Generate(X)
	if err != nil {
		return nil, err
	}
	// Get the size of the root window.
	rootGeom := xwindow.RootGeometry(X)
	// If the width or height haven't been preset,
	// create a window taking up 1/2 the space
	if width == 0 || height == 0 {
		width = rootGeom.Width() / 2
		height = rootGeom.Height() / 2
	}
	win.CreateChecked(X.RootWin(), rootGeom.Width()/4, rootGeom.Height()/4, width, height, xproto.CwBackPixel, 0)
	win.WMGracefulClose(func(w *xwindow.Window) {
		xevent.Detach(w.X, w.Id)
		keybind.Detach(w.X, w.Id)
		w.Destroy()
		xevent.Quit(w.X)
	})
	ewmh.WmNameSet(X, win.Id, "Initialising...")
	geom, err := win.Geometry()
	if err != nil {
		return nil, err
	}
	if *verbose {
		if err != nil {
			fmt.Printf("Unable to read geometry: %v\n", err)
		} else {
			fmt.Printf("Window at (%d, %d), size [%d x %d]\n", geom.X(), geom.Y(), geom.Width(), geom.Height())
		}
	}
	// Normalise the window geometry to (0, 0)
	geom.XSet(0)
	geom.YSet(0)
	win.Listen(xproto.EventMaskStructureNotify, xproto.EventMaskSubstructureNotify,
		xproto.EventMaskKeyPress, xproto.EventMaskKeyRelease)
	return &runner{X: X, win: win, preload: preload, geom: geom, loaded: map[int]nothing{}}, nil
}

func (r *runner) start(f []string) {
	inEvent, outEvent := initEvent()
	// Create a pict structure for every image
	for i, file := range f {
		p := NewPict(file, r.X, i)
		p.setTitle(fmt.Sprintf("%s (%d/%d)", p.name, i+1, len(f)))
		r.picts = append(r.picts, p)
	}
	r.addCache(0)
	xevent.ConfigureNotifyFun(
		func(X *xgbutil.XUtil, e xevent.ConfigureNotifyEvent) {
			outEvent <- event{E_RESIZE, int(e.Width), int(e.Height)}
		}).Connect(r.X, r.win.Id)
	xevent.KeyPressFun(
		func(X *xgbutil.XUtil, e xevent.KeyPressEvent) {
			modStr := keybind.ModifierString(e.State)
			keyStr := keybind.LookupString(X, e.State, e.Detail)
			if len(modStr) != 0 {
				keyStr = fmt.Sprintf("%s-%s", modStr, keyStr)
			}
			if *verbose {
				fmt.Printf("Key: %s\n", keyStr)
			}
			switch keyStr {
			case "q":
				outEvent <- event{E_QUIT, 0, 0}
			case "n", " ", "Right", "KP_Right":
				outEvent <- event{E_STEP, 1, 0}
			case "p", "Left", "KP_Left":
				outEvent <- event{E_STEP, -1, 0}
			case "KP_Up", "Up":
				outEvent <- event{E_STEP, -10, 0}
			case "KP_Down", "Down":
				outEvent <- event{E_STEP, 10, 0}
			case "Home", "KP_Home":
				outEvent <- event{E_JUMP, 0, 0}
			case "End", "KP_End":
				outEvent <- event{E_JUMP, len(r.picts) - 1, 0}
			case "0":
				outEvent <- event{E_RATING, 0, 0}
			case "1":
				outEvent <- event{E_RATING, 1, 0}
			case "2":
				outEvent <- event{E_RATING, 2, 0}
			case "3":
				outEvent <- event{E_RATING, 3, 0}
			case "4":
				outEvent <- event{E_RATING, 0, 0}
			case "5":
				outEvent <- event{E_RATING, 5, 0}
			}
		}).Connect(r.X, r.win.Id)
	r.win.Map()
	// Display the first picture
	r.show()
	// Preload next pictures
	r.cacheUpdate()
	xc1, xc2, xcExit := xevent.MainPing(r.X)
	for {
		select {
		case <-xc1:
		case <-xc2:
		case <-xcExit:
			return
		case ev := <-inEvent:
			switch ev.event {
			case E_RESIZE:
				r.resize(ev.w, ev.h)
			case E_RATING:
				r.rate(ev.w)
			case E_STEP:
				r.setIndex(r.index + ev.w)
			case E_JUMP:
				r.setIndex(ev.w)
			case E_QUIT:
				r.quit()
			}
		}
	}
}

func (r *runner) show() {
	p := r.picts[r.index]
	defer ewmh.WmNameSet(r.X, r.win.Id, p.title)
	err := p.wait()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: load err: %v", p.name, err)
		return
	}
	p.show(r.win)
}

// Resize notification.
func (r *runner) resize(w, h int) {
	if w == r.geom.Width() && h == r.geom.Height() {
		return
	}
	if *verbose {
		fmt.Printf("Resize to %d x %d (current %d x %d)\n", w, h, r.geom.Width(), r.geom.Height())
	}
	r.geom.WidthSet(w)
	r.geom.HeightSet(h)
	r.flushCache()
	// current picture should be redrawn.
	r.addCache(r.index)
	r.show()
	r.cacheUpdate()
}

func (r *runner) quit() {
	xevent.Quit(r.X)
}

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

// Update the cached set of images
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
			_, ok := r.loaded[index]
			if !ok {
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
		_, ok := nc[k]
		if !ok {
			r.removeCache(k)
		}
	}
	// Begin loading new items - we do this after unloading the expired entries
	for _, index := range newEntries {
		r.addCache(index)
	}
}

func (r *runner) flushCache() {
	for k, _ := range r.loaded {
		r.removeCache(k)
	}
}

func (r *runner) removeCache(index int) {
	_, ok := r.loaded[index]
	if ok {
		delete(r.loaded, index)
		r.picts[index].unload()
	}
}

func (r *runner) addCache(index int) {
	_, ok := r.loaded[index]
	if !ok {
		r.loaded[index] = nothing{}
		r.picts[index].startLoad(r.geom)
	}
}
