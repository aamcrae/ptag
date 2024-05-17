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
	"time"
)

// Event types
const (
	E_STEP = iota
	E_JUMP
	E_QUIT
	E_RESIZE
	E_RATING
)

func initEvent() (<-chan event, chan<- event) {
	in := make(chan event, 100)
	out := make(chan event, 100)
	go eventHandler(in, out)
	return in, out
}

func eventHandler(in chan event, out chan event) {
	t := time.NewTicker(time.Millisecond * 100)
	var resizeActive bool
	var w, h int
	var ticksResize int
	for {
		select {
		case ev := <-out:
			switch ev.event {
			case E_RESIZE:
				// Cache the resize update
				resizeActive = true
				ticksResize = 0
				w = ev.w
				h = ev.h
			default:
				if resizeActive {
					in <- event{E_RESIZE, w, h}
					resizeActive = false
				}
				in <- ev
			}
		case <-t.C:
			if resizeActive {
				ticksResize++
				if ticksResize >= 5 {
					in <- event{E_RESIZE, w, h}
					resizeActive = false
				}
			}
		}
	}
}
