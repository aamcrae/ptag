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

// Implement a simple EXIF sidecar.
// The format of the file is:
// <exif-tag> <value>
// This is the same format that the exiv2 utility outputs, allowing
// this handler to use the same parser.

import (
	"fmt"
	"os"
)

type exivSidecar struct {
	file string // sidecar file
	exif map[int]string
	Exif
}

func newExivSidecar(file string) (Exif, error) {
	// Add ".exif" to filename
	f := file + ".exif"
	e := &exivSidecar{file: f, exif: map[int]string{}}
	b, err := os.ReadFile(f)
	if err == nil {
		e.exif = readExif(f, string(b))
	}
	return e, nil
}

func (e *exivSidecar) Set(tag int, value string) error {
	_, ok := exivToSet[tag]
	if !ok {
		return fmt.Errorf("Unknown EXIF tag: %d", tag)
	}
	e.exif[tag] = value
	return e.write()
}

func (e *exivSidecar) Get(tag int) (string, bool) {
	val, ok := e.exif[tag]
	return val, ok
}

func (e *exivSidecar) Delete(tag int) error {
	if _, ok := e.exif[tag]; !ok {
		// No tag saved
		return nil
	}
	_, ok := exivToSet[tag]
	if !ok {
		return fmt.Errorf("Unknown EXIF tag: %d", tag)
	}
	delete(e.exif, tag)
	return e.write()
}

func (e *exivSidecar) write() error {
	f, err := os.Create(e.file)
	if err != nil {
		return err
	}
	defer f.Close()
	for k, v := range e.exif {
		fmt.Fprintf(f, "%s %s\n", exivToSet[k], v)
	}
	return nil
}
