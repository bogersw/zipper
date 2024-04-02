package zipper

import (
	"archive/zip"
	"compress/flate"
	"errors"
	"io"
	"os"
)

type ZipFile struct {
	fileHandle *os.File
	writer     *zip.Writer
	FileName   string
}

func (z ZipFile) Create(force bool) error {

	// Check if file exists
	if _, err := os.Stat(z.FileName); err == nil {
		// It exists, keep it, unless force is true
		if !force {
			return nil
		}
	}
	// Create or truncate the new zip file
	newZipFile, err := os.Create(z.FileName)
	if err != nil {
		return err
	}
	defer newZipFile.Close()
	return nil
}

func (z *ZipFile) Open() error {

	// Check if file exists
	if _, err := os.Stat(z.FileName); err != nil {
		// It doesn't exist
		return errors.New("zip file does not exist")
	}

	// Open the zip file for writing
	zipFile, err := os.OpenFile(z.FileName, os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return errors.New("error opening zip file")
	}
	z.fileHandle = zipFile

	// Create a writer for the zip file
	zipWriter := zip.NewWriter(zipFile)
	zipWriter.RegisterCompressor(zip.Deflate, func(out io.Writer) (io.WriteCloser, error) {
		return flate.NewWriter(out, flate.BestCompression)
	})
	z.writer = zipWriter

	return nil
}

func (z *ZipFile) Close() error {

	// Close file and writer. Check for errors that file and / or writer
	// are already closed.
	if err := z.writer.Close(); err != nil && !errors.Is(err, os.ErrClosed) {
		return err
	}
	if err := z.fileHandle.Close(); err != nil && !errors.Is(err, os.ErrClosed) {
		return err
	}
	return nil
}

func (z ZipFile) AddFile(fileName string) error {

	// Open the file which should be added
	fileToAdd, err := os.OpenFile(fileName, os.O_RDONLY, 0644)
	if err != nil {
		return errors.New("error opening file to add")
	}
	defer fileToAdd.Close()

	// Get FileInfo on the file to add: we need this for the header
	fileInfo, err := fileToAdd.Stat()
	if err != nil {
		return errors.New("error getting file info for the file to add")
	}

	// Set the header of the file. FileHeader describes a file within a ZIP file.
	header, err := zip.FileInfoHeader(fileInfo)
	if err != nil {
		return errors.New("error determining header for the file to add")
	}
	// Specify the compression method
	header.Method = zip.Deflate
	// Create the header: we get a io.Writer back for copying the data
	writer, err := z.writer.CreateHeader(header)
	if err != nil {
		return errors.New("error creating header for the file to add")
	}

	// Now acrually copy the file to the zip file
	if _, err = io.Copy(writer, fileToAdd); err != nil {
		return err
	}

	return nil
}

func (z ZipFile) GetFileList() ([]string, error) {

	// Reading a zip file is only possible if the zip file is not open for writing
	if z.fileHandle != nil {
		return nil, errors.New("error getting file list, zip file is open for writing")
	}

	// Open the zip file for reading
	zipFile, err := zip.OpenReader(z.FileName)
	if err != nil {
		return nil, errors.New("error opening zip file for reading")
	}
	defer zipFile.Close()

	// Create a list to store the names of the files that are already added
	list := []string{}

	// Iterate over the files in the zip archive and add file names
	// to the list. Skip directories.
	for _, file := range zipFile.File {
		if file.FileInfo().IsDir() {
			continue
		}
		list = append(list, file.Name)
	}

	return list, nil
}
