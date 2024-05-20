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
	"os/exec"
	"strings"
)

// The list of EXIF fields that we care about
const (
	EXIV_RATING = iota
	EXIV_CAPTION
	EXIV_ORIENTATION
)

// maps the exiv enum to the EXIF tag
var exivToSet = map[int]string{
	EXIV_RATING:      "Xmp.xmp.Rating",
	EXIV_CAPTION:     "Iptc.Application2.Caption",
	EXIV_ORIENTATION: "Exif.Image.Orientation",
}

// maps the EXIF tag to the internal enum
var exivFromName = map[string]int{
	"Xmp.xmp.Rating":               EXIV_RATING,
	"Iptc.Application2.Caption":    EXIV_CAPTION,
	"Iptc.Application2.Headline":   EXIV_CAPTION,
	"Iptc.Application2.ObjectName": EXIV_CAPTION,
	"Exif.Image.Orientation":       EXIV_ORIENTATION,
}

// getExiv will retrieve the required EXIF fields from the file
func getExiv(f string) (Exiv, error) {
	cmd := exec.Command("exiv2", "-q", "-P", "EkIXv", "-K", "Xmp.xmp.Rating",
		"-K", "Iptc.Application2.Caption",
		"-K", "Exif.Image.Orientation",
		"-K", "Iptc.Application2.Headline",
		"-K", "Iptc.Application2.ObjectName",
		f)
	outp, err := cmd.Output()
	if *verbose {
		fmt.Printf("Running: %s\noutput: %s\n", strings.Join(cmd.Args, " "), outp)
	}
	if err != nil {
		// Very likely there is no exif headers in this file.
		return nil, err
	}
	ev := make(Exiv)
	for _, l := range strings.Split(string(outp), "\n") {
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
			case EXIV_CAPTION:
				// Create a single string from the separate caption words
				ev[EXIV_CAPTION] = value
			case EXIV_RATING:
				// Validate rating (should "0" - "5")
				switch value {
				default:
					fmt.Fprintf(os.Stderr, "%s: illegal value for rating (%s)", f, value)
				case "0", "1", "2", "3", "4", "5":
					ev[exiv] = value
				}
			case EXIV_ORIENTATION:
				// Validate orientation (should "1" - "8")
				switch value {
				default:
					fmt.Fprintf(os.Stderr, "%s: illegal value for orientation (%s)", f, value)
				case "1", "2", "3", "4", "5", "6", "7", "8":
					ev[exiv] = value
				}
			}
		} else {
			fmt.Fprintf(os.Stderr, "%s: Unknown exiv tag: %s\n", f, fields[0])
		}
	}
	return ev, nil
}

// setExiv will set the selected EXIF fields in the file
func setExiv(f string, ex Exiv) error {
	if len(ex) == 0 {
		return fmt.Errorf("no EXIV tags to set")
	}
	cmd := exec.Command("exiv2", "-q")
	for k, v := range ex {
		cmd.Args = append(cmd.Args, fmt.Sprintf("-Mset %s %s", exivToSet[k], v))
	}
	cmd.Args = append(cmd.Args, f)
	if *verbose {
		fmt.Printf("Running: %s\n", strings.Join(cmd.Args, " "))
	}
	return cmd.Run()
}

// deleteExiv will delete the selected fields from the file
func deleteExiv(f string, ex Exiv) error {
	if len(ex) == 0 {
		return fmt.Errorf("no EXIV tags to delete")
	}
	cmd := exec.Command("exiv2", "-q")
	for k, _ := range ex {
		cmd.Args = append(cmd.Args, fmt.Sprintf("-Mdelete %s", exivToSet[k]))
	}
	cmd.Args = append(cmd.Args, f)
	if *verbose {
		fmt.Printf("Running: %s\n", strings.Join(cmd.Args, " "))
	}
	return cmd.Run()
}
