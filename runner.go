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
	return &runner{X: X, win: win, preload: preload, geom: geom}, nil
}

func (r *runner) start(f []string) {
	inEvent, outEvent := initEvent()
	// Create a pict structure for every image
	for i, file := range f {
		p := NewPict(file, r.X, i)
		r.picts = append(r.picts, p)
		if i < r.preload {
			p.startLoad(r.geom)
		}
	}
	xevent.ConfigureNotifyFun(
		func(X *xgbutil.XUtil, e xevent.ConfigureNotifyEvent) {
			outEvent <- event{E_RESIZE, int(e.Width), int(e.Height)}
		}).Connect(r.X, r.win.Id)
	xevent.KeyPressFun(
		func(X *xgbutil.XUtil, e xevent.KeyPressEvent) {
			keyStr := keybind.LookupString(X, e.State, e.Detail)
			if *verbose {
				fmt.Printf("Key: %s\n", keyStr)
			}
			switch keyStr {
			case "q":
				outEvent <- event{E_QUIT, 0, 0}
			case "n":
				outEvent <- event{E_NEXT, 0, 0}
			case "p":
				outEvent <- event{E_PREVIOUS, 0, 0}
			}
		}).Connect(r.X, r.win.Id)
	r.win.Map()
	// Display the first picture
	r.show()
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
			case E_NEXT:
				r.next()
			case E_PREVIOUS:
				r.previous()
			case E_QUIT:
				r.quit()
			}
		}
	}
}

func (r *runner) show() {
	p := r.picts[r.index]
	defer ewmh.WmNameSet(r.X, r.win.Id, p.name)
	err := p.wait()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: load err: %v", p.name, err)
		return
	}
	p.show(r.win)
}

// Resize notification.
func (r *runner) resize(w, h int) {
	if *verbose {
		fmt.Printf("Resize to %d x %d (current %d x %d)\n", w, h, r.geom.Width(), r.geom.Height())
	}
	if w == r.geom.Width() && h == r.geom.Height() {
		return
	}
	r.geom.WidthSet(w)
	r.geom.HeightSet(h)
	r.picts[r.index].startLoad(r.geom)
	r.show()
}

func (r *runner) quit() {
	xevent.Quit(r.X)
}

func (r *runner) previous() {
	if r.index == 0 {
		return
	}
	r.index--
	r.show()
	r.updateCache(r.index+r.preload, r.index-r.preload)
}

func (r *runner) next() {
	if r.index < len(r.picts)-1 {
		r.index++
		r.show()
		r.updateCache(r.index-r.preload-1, r.index+r.preload-1)
	}
}

// Update the cached images, unloading one and start to load another.
func (r *runner) updateCache(clean, load int) {
	if clean >= 0 && clean < len(r.picts) {
		r.picts[clean].unload()
	}
	if load >= 0 && load < len(r.picts) {
		r.picts[load].startLoad(r.geom)
	}
}
