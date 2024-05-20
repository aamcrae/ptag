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
	"os/exec"
	"strings"
)

type exivEmbedded struct {
	file string
	exif map[int]string
	Exif
}

func newExivEmbedded(file string) (Exif, error) {
	e := &exivEmbedded{file: file, exif: map[int]string{}}
	cmd := exec.Command("exiv2", "-q", "-P", "EkIXv", "-K", "Xmp.xmp.Rating",
		"-K", "Iptc.Application2.Caption",
		"-K", "Exif.Image.Orientation",
		"-K", "Iptc.Application2.Headline",
		"-K", "Iptc.Application2.ObjectName",
		file)
	outp, err := cmd.Output()
	if *verbose {
		fmt.Printf("Running: %s\noutput: %s\n", strings.Join(cmd.Args, " "), outp)
	}
	if err != nil {
		// No exif in file.
		return e, nil
	}
	e.exif = readExif(e.file, string(outp))
	return e, nil
}

func (e *exivEmbedded) Set(tag int, value string) error {
	etag, ok := exivToSet[tag]
	if !ok {
		return fmt.Errorf("Unknown EXIF tag: %d", tag)
	}
	cmd := exec.Command("exiv2", "-q")
	cmd.Args = append(cmd.Args, fmt.Sprintf("-Mset %s %s", etag, value))
	cmd.Args = append(cmd.Args, e.file)
	if *verbose {
		fmt.Printf("Running: %s\n", strings.Join(cmd.Args, " "))
	}
	if err := cmd.Run(); err != nil {
		return err
	}
	// Update local copy.
	e.exif[tag] = value
	return nil
}

func (e *exivEmbedded) Get(tag int) (string, bool) {
	val, ok := e.exif[tag]
	return val, ok
}

func (e *exivEmbedded) Delete(tag int) error {
	if _, ok := e.exif[tag]; !ok {
		// No tag saved
		return nil
	}
	etag, ok := exivToSet[tag]
	if !ok {
		return fmt.Errorf("Unknown EXIF tag: %d", tag)
	}
	cmd := exec.Command("exiv2", "-q", fmt.Sprintf("-Mdel %s", etag), e.file)
	if *verbose {
		fmt.Printf("Running: %s\n", strings.Join(cmd.Args, " "))
	}
	if err := cmd.Run(); err != nil {
		return err
	}
	delete(e.exif, tag)
	return nil
}
