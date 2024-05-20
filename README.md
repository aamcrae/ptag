# ptag
ptag is a high speed image viewer, with the ability to adjust EXIF ratings and captions.
No direct image manipulation is done - only EXIF data is modifed.

The EXIF data can be stored in different formats or files e.g embedded
in the image file itself, or in sidecar files.

ptag will preload images ready for viewing so that image display is fast.

The [fyne](https://fyne.io/) toolkit is used for window management and display,
and the [vips](https://github.com/davidbyttow/govips) library for image handling.
The [exiv2](https://exiv2.org/) tool is used to read and save the EXIF data.

Run ```ptag --help``` to get the usage and keyboard shortcuts supported.
