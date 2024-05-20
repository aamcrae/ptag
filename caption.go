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
	"fyne.io/fyne/v2/driver/desktop"
)

func (c *CaptionEntry) MouseIn(*desktop.MouseEvent) {
	c.runner.win.Canvas().Focus(c)
	c.runner.Updated() // Flag that the caption may have changed
}

func (c *CaptionEntry) MouseOut() {
	c.runner.win.Canvas().Unfocus()
	c.runner.Sync() // Write the caption to the EXIF data
}

func (c *CaptionEntry) MouseMoved(*desktop.MouseEvent) {
}
