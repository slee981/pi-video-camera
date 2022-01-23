// What it does:
//
// This example uses the Tensorflow (https://www.tensorflow.org/) deep learning framework
// to classify whatever is in front of the camera.
//
// Download the Tensorflow "Inception" model and descriptions file from:
// https://storage.googleapis.com/download.tensorflow.org/models/inception5h.zip
//
// Extract the tensorflow_inception_graph.pb model file from the .zip file.
//
// Also extract the imagenet_comp_graph_label_strings.txt file with the descriptions.
//
// How to run:
//
// 		go run ./cmd/tf-classifier/main.go 0 ~/Downloads/tensorflow_inception_graph.pb ~/Downloads/imagenet_comp_graph_label_strings.txt opencv cpu
//

package main

import (
	"bufio"
	"fmt"
	"image"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"example.com/pi-video-recorder/utils/bufferqueue"
	"gocv.io/x/gocv"
)

var (
	VIDEO_NUMBER  int = 1   // counter
	PRED_TO_MATCH int = 535 // idx from labels

	RECORD_TIME               int = 15 // seconds
	RECORD_TIME_AFTER_TRIGGER     = RECORD_TIME / 2

	FPS              float64 = 15
	FRAMES_PER_VIDEO int     = int(FPS) * RECORD_TIME

	SAVE_FILE_TEMPLATE string = "imgs/recording_{VIDEO_NUMBER}.avi"
)

var (
	deviceID int    = 0
	model    string = "/home/stephen/Downloads/tf/tensorflow_inception_graph.pb"
	// descr       string = "/home/stephen/Downloads/tf/imagenet_comp_graph_label_strings.txt"
	backendPref string = "opencv"
	targetPref  string = "cpu"
)

func main() {

	// prediction channel
	wg := new(sync.WaitGroup)
	pc := make(chan int, 1)
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// buffer queue for handling video stream
	bq := bufferqueue.NewBufferQueue(FRAMES_PER_VIDEO)

	// read model description
	// descriptions, err := readDescriptions(descr)
	// if err != nil {
	// 	fmt.Printf("Error reading descriptions file: %v\n", descr)
	// 	fmt.Println(err)
	// 	return
	// }

	/* setup model */

	backend := gocv.ParseNetBackend(backendPref)
	target := gocv.ParseNetTarget(targetPref)
	webcam, err := gocv.OpenVideoCapture(deviceID)
	if err != nil {
		fmt.Printf("Error opening video capture device: %v\n", deviceID)
		return
	}
	defer webcam.Close()

	img := gocv.NewMat()
	defer img.Close()

	// open DNN classifier
	net := gocv.ReadNet(model, "")
	if net.Empty() {
		fmt.Printf("Error reading network model : %v\n", model)
		return
	}
	defer net.Close()
	net.SetPreferableBackend(gocv.NetBackendType(backend))
	net.SetPreferableTarget(gocv.NetTargetType(target))

	// initialize image and prediciton
	if ok := webcam.Read(&img); !ok {
		fmt.Printf("Cannot read dev %v\n", deviceID)
		return
	}
	fmt.Println("initializing to camera")
	wg.Add(1)
	go predict(net, img, pc, wg)

	/* do main loop */

	fmt.Println("recording...")
	run := true
	ct := 0
	for run {
		if ok := webcam.Read(&img); !ok {
			fmt.Printf("Device closed: %v\n", deviceID)
			return
		}
		if img.Empty() {
			continue
		}

		// push frame to queue
		bq.Push(img)

		// make prediction when ready
		//
		// do this in a go routine that
		// communicates back over a channel when complete
		select {
		case pred := <-pc:
			// desc := descriptions[pred]
			// fmt.Println("loop", ct, ": found ", desc, " at idx ", pred)
			if pred == PRED_TO_MATCH {
				// 1- write video only when we found what we wanted
				// 2- make sure we're not in the middle of a previous write
				//
				// NOTE: after a "writeVideo" call, write will sleep for some time
				// before actually writing the file to ensure the triggering event
				// is caught in the middle of the video.
				switch {
				case bq.IsWritable():
					// last write already succeeded
					wg.Add(1)
					go writeVideo(bq, wg)
				default:
					// we matched again before the last write is complete
				}
			}

			// send new prediction
			wg.Add(1)
			go predict(net, img, pc, wg)
		case <-done:
			fmt.Println("received interrupt. shutting down gracefully.")
			run = false
		default:
			// no prediction result back yet
			continue
		}
		ct++
	}

	wg.Wait()
}

// readDescriptions reads the descriptions from a file
// and returns a slice of its lines.
func readDescriptions(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

func predict(net gocv.Net, img gocv.Mat, c chan int, wg *sync.WaitGroup) {
	defer wg.Done()

	// convert image Mat to 224x224 blob that the classifier can analyze
	blob := gocv.BlobFromImage(img, 1.0, image.Pt(224, 224), gocv.NewScalar(0, 0, 0, 0), true, false)
	defer blob.Close()

	// feed the blob into the classifier
	net.SetInput(blob, "input")

	// run a forward pass thru the network
	prob := net.Forward("softmax2")
	defer prob.Close()

	// reshape the results into a 1x1000 matrix
	probMat := prob.Reshape(1, 1)
	defer probMat.Close()

	// determine the most probable classification
	_, _, _, maxLoc := gocv.MinMaxLoc(probMat)

	c <- maxLoc.X
}

func writeVideo(bq *bufferqueue.BufferQueue, wg *sync.WaitGroup) {
	defer wg.Done()

	// block other write calls
	bq.Lock()
	bq.LockWrite()
	defer bq.UnlockWrite()
	bq.Unlock()

	// sleep in the background until we're ready to capture video
	fmt.Println("triggered save. sleeping... ")
	time.Sleep(time.Second * time.Duration(RECORD_TIME_AFTER_TRIGGER))
	fmt.Println("done sleeping. doing save")

	// lock buffer queue and get dim of first image
	bq.Lock()
	defer bq.Unlock()
	img := bq.First().GetData()

	// create filename
	saveFname := genSaveFname()
	fmt.Println("saving to: ", saveFname)

	// create writer
	writer, err := gocv.VideoWriterFile(saveFname, "MJPG", FPS, img.Cols(), img.Rows(), true)
	if err != nil {
		fmt.Printf("error opening video writer device: %v\n", saveFname)
		return
	}
	defer writer.Close()

	// write each image in the list and communicate back
	doWrite(bq, writer)
}

func doWrite(bq *bufferqueue.BufferQueue, writer *gocv.VideoWriter) {
	for n := bq.First(); n != nil; n = n.Next() {
		writer.Write(n.GetData())
	}
	fmt.Println("saved.")
}

func genSaveFname() string {
	replaceTag := "{VIDEO_NUMBER}"
	r := strings.NewReplacer(replaceTag, strconv.Itoa(VIDEO_NUMBER))

	VIDEO_NUMBER++
	return r.Replace(SAVE_FILE_TEMPLATE)
}
