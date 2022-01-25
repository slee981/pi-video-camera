package main

import (
    "fmt"

	"github.com/slee981/pi-video-camera/utils/uploader"
)

func main() {

    // get uploader to run cloud sync
	uploader := uploader.NewUploader(
		"ws0tcluXpxnWAlkbTSpCnmSY6aXX+iogSd+dHaL7mpBqdLz5Xu2Z6FIHc8Phjvs5S7BlihVGmShe0vs8epGOkw==",
		"pivideos",
		"recordings",
		"/home/stephen/Documents/CodeWorkspace/Go/pi-video-recorder",
		"avi",
	)

    // fname := "utils/uploader/test/test.txt"
    fname := "imgs/recording_1.avi"
    res, err := uploader.Upload(fname)
    fmt.Println("received result", res, "and error", err)
}

