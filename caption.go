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

// Extend the Entry widget to capture mouse events
// so that the caption window can be automatically focused.

import (
	"fmt"
	"fyne.io/fyne/v2/driver/desktop"
)

func (c *CaptionEntry) MouseIn(*desktop.MouseEvent) {
	c.app.win.Canvas().Focus(c)
	c.mouseIn = true
	if *verbose {
		fmt.Printf("MouseIn\n")
	}
	c.app.Updated() // Flag that the caption may have changed
}

func (c *CaptionEntry) MouseOut() {
	if *verbose {
		fmt.Printf("MouseOut\n")
	}
	c.mouseIn = false
	c.app.win.Canvas().Unfocus()
	c.app.Sync() // Write the caption to the EXIF data
}

func (c *CaptionEntry) MouseMoved(*desktop.MouseEvent) {
}

func (c *CaptionEntry) OnChange(v string) {
	if *verbose {
		fmt.Printf("OnChange: <%s>\n", v)
	}
	c.app.Updated()
	if !c.mouseIn {
		// Change may have been due to a paste.
		c.app.win.Canvas().Unfocus()
		c.app.Sync()
	}
}
