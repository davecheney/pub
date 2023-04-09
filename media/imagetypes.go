package media

// image.Decode expects image decoders to be registered in the global image package.
// This file registers the decoders we need via the side-effect of importing the
// packages that contain them.

import (
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	_ "golang.org/x/image/webp"
)
