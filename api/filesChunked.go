package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strconv"

	"github.com/google/uuid"
)

// AddChunkedOptions provides optional settings for AddChunked method
type AddChunkedOptions struct {
	Overwrite bool                                    // should overwrite existing file
	Progress  func(data *FileUploadProgressData) bool // on progress callback, execute custom logic on each chunk, if the Progress is used it should return "true" to continue upload otherwise upload is canceled
	ChunkSize int                                     // chunk size in bytes
}

// FileUploadProgressData describes Progress callback options
type FileUploadProgressData struct {
	UploadID    string
	Stage       string
	ChunkSize   int
	BlockNumber int
	FileOffset  int
}

// AddChunked uploads a file in chunks (streaming), is a good fit for large files. Supported starting from SharePoint 2016.
func (files *Files) AddChunked(name string, stream io.Reader, options *AddChunkedOptions) (FileResp, error) {
	web := NewSP(files.client).Web().Conf(files.config)
	var file *File
	uploadID := uuid.New().String()

	cancelUpload := func(file *File, uploadID string) error {
		if err := file.cancelUpload(uploadID); err != nil {
			log.Printf("error canceling upload: %v", err)
			return err
		}
		log.Printf("upload was canceled")
		return fmt.Errorf("file upload was canceled")
	}

	// Default props
	if options == nil {
		options = &AddChunkedOptions{
			Overwrite: true,
			ChunkSize: 10485760,
		}
	}
	if options.Progress == nil {
		options.Progress = func(data *FileUploadProgressData) bool {
			return true
		}
	}
	if options.ChunkSize == 0 {
		options.ChunkSize = 10485760
	}

	progress := &FileUploadProgressData{
		UploadID:    uploadID,
		Stage:       "starting",
		ChunkSize:   options.ChunkSize,
		BlockNumber: 0,
		FileOffset:  0,
	}

	slot := make([]byte, options.ChunkSize)
	//reader := bufio.NewReader(stream)
	for {

		numBytesRead := 0
		for numBytesRead < options.ChunkSize {
			n, err := stream.Read(slot[numBytesRead:])
			if err != nil {
				if err == io.EOF {
					break
				}
			}
			numBytesRead += n
		}

		// size, err := reader.Read(slot)
		// if err == io.EOF {
		// 	break
		// }
		chunk := slot[:numBytesRead]

		log.Printf("read chunk size: %d", numBytesRead)

		// Upload in a call if file size is less than chunk size
		// pnocera: this is not working with io.Reader when it's not a file but a reader from an io.Pipe, so I commented it out
		if numBytesRead < options.ChunkSize && progress.BlockNumber == 0 {
			return files.Add(name, chunk, options.Overwrite)
		}

		// Finishing uploading chunked file
		if numBytesRead < options.ChunkSize && progress.BlockNumber > 0 {
			progress.Stage = "finishing"
			log.Printf("finishing upload")
			if !options.Progress(progress) {
				return nil, cancelUpload(file, uploadID)
			}
			if file == nil {
				return nil, fmt.Errorf("can't get file object")
			}
			return file.finishUpload(uploadID, progress.FileOffset, chunk)
		}

		// Initial chunked upload
		if progress.BlockNumber == 0 {
			progress.Stage = "starting"
			log.Printf("starting upload")
			if !options.Progress(progress) {
				return nil, fmt.Errorf("file upload was canceled") // cancelUpload(file, uploadID)
			}
			fileResp, err := files.Add(name, nil, options.Overwrite)
			if err != nil {
				return nil, err
			}
			file = web.GetFile(fileResp.Data().ServerRelativeURL)
			offset, err := file.startUpload(uploadID, chunk)
			if err != nil {
				return nil, err
			}
			progress.FileOffset = offset
		} else { // or continue chunk upload
			progress.Stage = "continue"
			log.Printf("continue upload")
			if !options.Progress(progress) {
				return nil, cancelUpload(file, uploadID)
			}
			if file == nil {
				return nil, fmt.Errorf("can't get file object")
			}
			offset, err := file.continueUpload(uploadID, progress.FileOffset, chunk)
			if err != nil {
				return nil, err
			}
			progress.FileOffset = offset
		}

		progress.BlockNumber++
	}

	progress.Stage = "finishing"
	log.Printf("finishing upload on main")
	if !options.Progress(progress) {
		return nil, cancelUpload(file, uploadID)
	}
	if file == nil {
		return nil, fmt.Errorf("can't get file object")
	}
	return file.finishUpload(uploadID, progress.FileOffset, nil)
}

// startUpload starts uploading a document using chunk API
func (file *File) startUpload(uploadID string, chunk []byte) (int, error) {
	client := NewHTTPClient(file.client)
	endpoint := fmt.Sprintf("%s/StartUpload(uploadId=guid'%s')", file.endpoint, uploadID)
	data, err := client.Post(endpoint, bytes.NewBuffer(chunk), file.config)
	if err != nil {
		return 0, err
	}
	data = NormalizeODataItem(data)
	if res, err := strconv.Atoi(fmt.Sprintf("%s", data)); err == nil {
		return res, nil
	}
	res := &struct {
		StartUpload int `json:"StartUpload,string"` // With OData=verbose
		Value       int `json:"value,string"`       // With OData=minimanmetadata or nometadata
	}{}
	if err := json.Unmarshal(data, &res); err != nil {
		return 0, err
	}
	offset := res.StartUpload
	if offset == 0 {
		offset = res.Value
	}
	return offset, nil
}

// continueUpload continues uploading a document using chunk API
func (file *File) continueUpload(uploadID string, fileOffset int, chunk []byte) (int, error) {
	client := NewHTTPClient(file.client)
	endpoint := fmt.Sprintf("%s/ContinueUpload(uploadId=guid'%s',fileOffset=%d)", file.endpoint, uploadID, fileOffset)
	data, err := client.Post(endpoint, bytes.NewBuffer(chunk), file.config)
	if err != nil {
		return 0, err
	}
	data = NormalizeODataItem(data)
	if res, err := strconv.Atoi(fmt.Sprintf("%s", data)); err == nil {
		return res, nil
	}
	res := &struct {
		ContinueUpload int `json:"ContinueUpload,string"` // With OData=verbose
		Value          int `json:"value,string"`          // With OData=minimanmetadata or nometadata
	}{}
	if err := json.Unmarshal(data, &res); err != nil {
		return 0, err
	}
	offset := res.ContinueUpload
	if offset == 0 {
		offset = res.Value
	}
	return offset, nil
}

// cancelUpload cancels document upload using chunk API
func (file *File) cancelUpload(uploadID string) error {
	client := NewHTTPClient(file.client)
	endpoint := fmt.Sprintf("%s/CancelUpload(uploadId=guid'%s')", file.endpoint, uploadID)
	_, err := client.Post(endpoint, nil, file.config)
	return err
}

// finishUpload finishes uploading a document using chunk API
func (file *File) finishUpload(uploadID string, fileOffset int, chunk []byte) (FileResp, error) {
	client := NewHTTPClient(file.client)
	endpoint := fmt.Sprintf("%s/FinishUpload(uploadId=guid'%s',fileOffset=%d)", file.endpoint, uploadID, fileOffset)
	return client.Post(endpoint, bytes.NewBuffer(chunk), file.config)
}
