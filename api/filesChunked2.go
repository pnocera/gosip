package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"

	"github.com/google/uuid"
)

// AddChunked uploads a file in chunks (streaming), is a good fit for large files. Supported starting from SharePoint 2016.
func (files *Files) AddChunked2(name string, stream io.Reader, options *AddChunkedOptions) (FileResp, error) {
	web := NewSP(files.client).Web().Conf(files.config)
	var file *File
	uploadID := uuid.New().String()

	cancelUpload2 := func(file *File, uploadID string) error {
		if err := file.cancelUpload2(uploadID); err != nil {
			return err
		}
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
	for {
		size, err := stream.Read(slot)
		if err == io.EOF {
			log.Printf("reached end of file, breaking\n")
			break
		}
		chunk := slot[:size]

		// Upload in a call if file size is less than chunk size
		if size < options.ChunkSize && progress.BlockNumber == 0 {
			log.Printf("chunksize is greater than file size, uploading one block %d\n", size)
			return files.Add(name, chunk, options.Overwrite)
		}

		// Finishing uploading chunked file
		if size < options.ChunkSize && progress.BlockNumber > 0 {
			log.Printf("finishing %d\n", size)
			progress.Stage = "finishing"
			if !options.Progress(progress) {
				return nil, cancelUpload2(file, uploadID)
			}
			if file == nil {
				return nil, fmt.Errorf("can't get file object")
			}
			return file.finishUpload2(uploadID, progress.FileOffset, chunk)
		}

		// Initial chunked upload
		if progress.BlockNumber == 0 {
			log.Printf("starting\n")
			progress.Stage = "starting"
			if !options.Progress(progress) {
				return nil, fmt.Errorf("file upload was canceled") // cancelUpload(file, uploadID)
			}
			fileResp, err := files.Add(name, nil, options.Overwrite)
			if err != nil {
				return nil, err
			}
			file = web.GetFile(fileResp.Data().ServerRelativeURL)
			offset, err := file.startUpload2(uploadID, chunk)
			if err != nil {
				return nil, err
			}
			progress.FileOffset = offset
			log.Printf("started with offset %d\n", offset)
		} else { // or continue chunk upload
			log.Printf("continuing\n")
			progress.Stage = "continue"
			if !options.Progress(progress) {
				return nil, cancelUpload2(file, uploadID)
			}
			if file == nil {
				return nil, fmt.Errorf("can't get file object")
			}
			offset, err := file.continueUpload2(uploadID, progress.FileOffset, chunk)
			if err != nil {
				return nil, err
			}
			progress.FileOffset = offset
			log.Printf("continued with offset %d\n", offset)
		}

		progress.BlockNumber++
	}
	log.Printf("finishing\n")
	progress.Stage = "finishing"
	if !options.Progress(progress) {
		return nil, cancelUpload2(file, uploadID)
	}
	if file == nil {
		return nil, fmt.Errorf("can't get file object")
	}
	return file.finishUpload2(uploadID, progress.FileOffset, nil)
}

// startUpload starts uploading a document using chunk API
func (file *File) startUpload2(uploadID string, chunk []byte) (int, error) {
	client := NewHTTPClient(file.client)
	endpoint := fmt.Sprintf("%s/StartUpload(uploadId=guid'%s')", file.endpoint, uploadID)
	data, err := client.Post(endpoint, bytes.NewBuffer(chunk), file.config)
	log.Printf(string(data) + "\n")
	if err != nil {
		return 0, err
	}
	data = NormalizeODataItem(data)
	// if res, err := strconv.Atoi(fmt.Sprintf("%s", data)); err == nil {
	// 	return res, nil
	// }
	res := &struct {
		Value int `json:"value"`
	}{}
	if err := json.Unmarshal(data, &res); err != nil {
		return 0, err
	}
	return res.Value, nil
}

// continueUpload continues uploading a document using chunk API
func (file *File) continueUpload2(uploadID string, fileOffset int, chunk []byte) (int, error) {
	client := NewHTTPClient(file.client)
	endpoint := fmt.Sprintf("%s/ContinueUpload(uploadId=guid'%s',fileOffset=%d)", file.endpoint, uploadID, fileOffset)
	data, err := client.Post(endpoint, bytes.NewBuffer(chunk), file.config)
	if err != nil {
		return 0, err
	}
	log.Printf(string(data) + "\n")
	data = NormalizeODataItem(data)
	// if res, err := strconv.Atoi(fmt.Sprintf("%s", data)); err == nil {
	// 	return res, nil
	// }
	res := &struct {
		Value int `json:"value"`
	}{}
	if err := json.Unmarshal(data, &res); err != nil {
		return 0, err
	}
	return res.Value, nil
}

// cancelUpload cancels document upload using chunk API
func (file *File) cancelUpload2(uploadID string) error {
	client := NewHTTPClient(file.client)
	endpoint := fmt.Sprintf("%s/CancelUpload(uploadId=guid'%s')", file.endpoint, uploadID)
	log.Printf(endpoint + "\n")
	_, err := client.Post(endpoint, nil, file.config)
	return err
}

// finishUpload finishes uploading a document using chunk API
func (file *File) finishUpload2(uploadID string, fileOffset int, chunk []byte) (FileResp, error) {
	client := NewHTTPClient(file.client)
	endpoint := fmt.Sprintf("%s/FinishUpload(uploadId=guid'%s',fileOffset=%d)", file.endpoint, uploadID, fileOffset)
	log.Printf(endpoint + "\n")
	return client.Post(endpoint, bytes.NewBuffer(chunk), file.config)
}
