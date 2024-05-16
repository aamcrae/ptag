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
	"github.com/jezek/xgbutil/keybind"
	"github.com/jezek/xgbutil/xevent"
	"github.com/jezek/xgbutil/xwindow"
)

func (r *runner) Start(f []string) {
	X, err := xgbutil.NewConn()
	if err != nil {
		fmt.Fprintf(os.Stderr, "X11: %v", err)
		return
	}
	// Create a pict structure for every image
	for i, file := range f {
		p := NewPict(file, X, i)
		r.picts = append(r.picts, p)
		if i < r.preload {
			p.startLoad(r.width, r.height)
		}
	}
	// Create and map a window
	r.win, err = xwindow.Generate(X)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Win Generate: %v", err)
		return
	}
	keybind.Initialize(X)
	r.win.CreateChecked(X.RootWin(), 20, 50, r.width, r.height, xproto.CwBackPixel, 0)
	r.win.WMGracefulClose(func(w *xwindow.Window) {
		xevent.Detach(w.X, w.Id)
		keybind.Detach(w.X, w.Id)
		w.Destroy()
		xevent.Quit(w.X)
	})
	r.win.Listen(xproto.EventMaskKeyPress, xproto.EventMaskKeyRelease)
	xevent.KeyPressFun(
		func(X *xgbutil.XUtil, e xevent.KeyPressEvent) {
			keyStr := keybind.LookupString(X, e.State, e.Detail)
			if *verbose {
				fmt.Printf("Key: %s\n", keyStr)
			}
			switch keyStr {
			case "q":
				r.quit()
			case "n":
				r.next()
			case "p":
				r.previous()
			}
		}).Connect(X, r.win.Id)
	r.win.Map()
	// Add some callbacks for keys
	// Display the first picture
	r.show()
	xevent.Main(X)
}

func (r *runner) show() {
	p := r.picts[r.index]
	err := p.wait()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: load err: %v", p.name, err)
		return
	}
	p.show(r.win)
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
	r.cache(r.index+r.preload, r.index-r.preload)
}

func (r *runner) next() {
	if r.index < len(r.picts)-1 {
		r.index++
		r.show()
		r.cache(r.index-r.preload-1, r.index+r.preload-1)
	}
}

func (r *runner) cache(clean, load int) {
	if clean >= 0 && clean < len(r.picts) {
		r.picts[clean].unload()
	}
	if load >= 0 && load < len(r.picts) {
		r.picts[load].startLoad(r.width, r.height)
	}
}
