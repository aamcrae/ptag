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

// exiv is an intermediate handler for the EXIF data.
// Separate EXIF handlers can be defined that allow EXIF data to be
// stored in different files or formats.
// Only some EXIF tags are used.

import (
	"fmt"
	"os"
	"strings"
)

// maps the internal EXIF enum to the EXIF tag string
var exivToSet = map[int]string{
	EXIV_RATING:      "Xmp.xmp.Rating",
	EXIV_HEADLINE:    "Iptc.Application2.Headline",
	EXIV_ORIENTATION: "Exif.Image.Orientation",
}

// maps the EXIF tag string to the internal enum
var exivFromName = map[string]int{
	"Xmp.xmp.Rating":               EXIV_RATING,
	"Iptc.Application2.Caption":    EXIV_HEADLINE,
	"Iptc.Application2.Headline":   EXIV_HEADLINE,
	"Iptc.Application2.ObjectName": EXIV_HEADLINE,
	"Exif.Image.Orientation":       EXIV_ORIENTATION,
}

// GetExif will create and return the EXIF object for this file
var GetExif func(string, []byte) (Exif, error)

// Select which EXIF handler should be used.
func initExif() {
	if *sidecar {
		GetExif = newExivSidecar // Simple EXIF sidecar file
	} else {
		GetExif = newExivEmbedded // Embedded EXIF in image file.
	}
}

// readExif parses lines of the form "<exif-tag> <value>"
// and returns a map containing the exif data.
// The exiv2 utility outputs data in this format.
func readExif(src, lines string) map[int]string {
	ex := make(map[int]string)
	for _, l := range strings.Split(lines, "\n") {
		if len(l) == 0 {
			continue
		}
		fields := strings.Fields(l)
		if len(fields) < 2 {
			continue
		}
		exiv, ok := exivFromName[fields[0]]
		if ok {
			// Concatenate values
			value := strings.Join(fields[1:], " ")
			switch exiv {
			case EXIV_HEADLINE:
				// Create a single string from the separate caption words
				ex[EXIV_HEADLINE] = value
			case EXIV_RATING:
				// Validate rating (should "0" - "5")
				switch value {
				default:
					fmt.Fprintf(os.Stderr, "%s: illegal value for rating (%s)", src, value)
				case "0", "1", "2", "3", "4", "5":
					ex[exiv] = value
				}
			case EXIV_ORIENTATION:
				// Validate orientation (should "1" - "8")
				switch value {
				default:
					fmt.Fprintf(os.Stderr, "%s: illegal value for orientation (%s)", src, value)
				case "1", "2", "3", "4", "5", "6", "7", "8":
					ex[exiv] = value
				}
			}
		} else {
			fmt.Fprintf(os.Stderr, "%s: Unknown exiv tag: %s\n", src, fields[0])
		}
	}
	return ex
}
