package testbufferqueue

import (
	"fmt"
	"os"
	"strconv"

	"github.com/slee981/pi-video-camera/utils/bufferqueue"
)

func TestBufferQueue() {
	items := 50
	if len(os.Args) == 2 {
		items, _ = strconv.Atoi(os.Args[1])
	}

	bq := bufferqueue.NewBufferQueue(10)
	for i := 0; i < items; i++ {
		bq.Push(i)
	}
	fmt.Println("queue of length ", bq.Length(), " with max length ", bq.MaxLength())
	bq.ToString()
}
