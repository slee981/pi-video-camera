package uploader

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/gofrs/uuid"
)

type AzureStorageContainer struct {
	key           string
	endpoint      string
	accountName   string
	containerName string
}

type AzureUploader struct {
	container     AzureStorageContainer
	localDir      string
	fileExtension string

	mu *sync.Mutex
}

// returns a new uploader object
func NewUploader(
	key string,
	acctName string,
	containerName string,
	dir string,
	fileExtension string,
) *AzureUploader {
	azStorage := NewStorageContainer(key, acctName, containerName)
	return &AzureUploader{
		container:     *azStorage,
		localDir:      dir,
		fileExtension: fileExtension,
		mu:            new(sync.Mutex),
	}
}

func NewStorageContainer(
	key string,
	acctName string,
	containerName string,

) *AzureStorageContainer {
	endpoint := fmt.Sprintf("https://%s.blob.core.windows.net/%s", acctName, containerName)
	return &AzureStorageContainer{
		key:           key,
		endpoint:      endpoint,
		accountName:   acctName,
		containerName: containerName,
	}
}

// external method

func (u *AzureUploader) Sync() (bool, error) {
	files, err := u.getFiles()
	if err != nil {
		fmt.Println("Error has occured:", err)
		return false, err
	}
	ok, err := u.uploadFiles(files)

	return ok, err
}

func (u *AzureUploader) Upload(f string) (bool, error) {
	file := fmt.Sprintf("%s/%s", u.localDir, f)

	return u.uploadFile(file)
}

// internal methods

func (u *AzureUploader) getFiles() ([]string, error) {
	var files []string

	err := filepath.Walk(u.localDir, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() && strings.Contains(path, u.fileExtension) {
			files = append(files, path)
		}
		return nil
	})

	return files, err
}

func (u *AzureUploader) getUploadedFiles() ([]string, error) {
	// TODO: implement me
	// read log file to see what files have been uploaded
	return []string{""}, nil
}

func (u *AzureUploader) uploadFiles(files []string) (bool, error) {
	for _, f := range files {
		ok, err := u.uploadFile(f)
		if !ok {
			fmt.Println("unable to upload ", f)
			return false, err
		}
	}
	return true, nil
}

func (u *AzureUploader) uploadFile(f string) (bool, error) {
	// going to upload, don't let others do the same
	u.mu.Lock()
	defer u.mu.Unlock()

	d, err := readFile(f)
	if err != nil {
		return false, err
	}

	ok := false
	tries := 0
	maxTries := 3
	for !ok && tries < maxTries {
		// try to upload 3 times before stopping
		// assume no internet and will try later
		fmt.Println("trying upload ", tries)
		ok, err = u.uploadBytesToBlob(d)
		tries++
	}
	if err != nil {
		return false, err
	}
	fmt.Println("uploaded ", f)
	return true, nil
}

func readFile(filePath string) ([]byte, error) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (u *AzureUploader) uploadBytesToBlob(b []byte) (bool, error) {

	// create filename
	uploadFile, _ := url.Parse(fmt.Sprint(u.container.endpoint, "/", u.getBlobName()))
	cred, err := azblob.NewSharedKeyCredential(u.container.accountName, u.container.key)
	if err != nil {
		return false, err
	}

	blockBlobUrl := azblob.NewBlockBlobURL(*uploadFile, azblob.NewPipeline(cred, azblob.PipelineOptions{}))
	fmt.Println("trying to upload ", blockBlobUrl.String())

	ctx := context.Background()
	headers := azblob.UploadToBlockBlobOptions{
		BlobHTTPHeaders: azblob.BlobHTTPHeaders{
			ContentType: "video/mpeg",
		},
	}

	_, err = azblob.UploadBufferToBlockBlob(ctx, b, blockBlobUrl, headers)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (u *AzureUploader) getBlobName() string {
	t := time.Now().Format("20060102T150405")
	uuid, _ := uuid.NewV4()

	return fmt.Sprintf("%s-%v.%s", t, uuid, u.fileExtension)
}
