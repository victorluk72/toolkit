package toolkit

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

//-----------BEGINING OF RANDOM STRING GENERATION SECTION-----------------

// This constant contains all possible characters, that we can use for randomly generated string
// we can use it for example, for creating file names for Linux system.
const randomStringSource = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_+"

// Tools is the custom type used to instantiate this module.
// Any variables of this type will have access to all methods with reciever *Tools
// This technic used to share methods from the modules with other programs
type Tools struct {
	MaxFileSize      int
	AllowedFileTypes []string //You can provide what type of file you allow (imgs/pdf/docs etc)
}

// RandomStringGenerator generates random string of certain length
// it uses constant randomStringSource as a source for characters
// it accepts one parameter - lenght of string we want to generate and
// returns the random string
func (t *Tools) RandomStringGenerator(n int) string {

	s := make([]rune, n)
	r := []rune(randomStringSource)

	//Some function details are here
	//rand.Reader is a global, shared instance of a cryptographically secure random number generator

	for i := range s {

		//p returns the number of the given bit length that is prime with high probability.
		//Prime will return error for any error returned by rand
		p, _ := rand.Prime(rand.Reader, len(r))

		x := p.Uint64()
		y := uint64(len(r))

		s[i] = r[x%y]

	}

	return string(s)

}

//-----------END OF RANDOM STRING GENERATION SECTION-----------------

//-----------BEGINING OF IMAGE UPLOAD SECTION-------------------------
// This function used to upload files from browser to server

// UploadedFile used to provide information about file that was uploaded from
// local browser to server
type UploadedFile struct {
	NewFileName      string //Shows new generated file name (if we do rename it)
	OriginalFileName string //Shows original file name
	FileSize         int64
}

// UploadFiles takes following paramenters:
//   -details of HTTP request (where file is posted from local comp to server)
//   -user will do POST HTTP request in order to upload file to server
//   -directory we want to upload our files to
//   -do we want to rename file or keep the original name. It can have one or more bools (or empty)for multiple files
// Function returns info about files we uploaded as slice of type UploadedFiles
func (t *Tools) UploadFiles(r *http.Request, uploadDir string, remame ...bool) ([]*UploadedFile, error) {

	// by default rename will be true and we will rename each file to random string
	renameFile := true

	// if we have any boolean value (true or false) from function we will use these values instead of default
	if len(remame) > 0 {
		renameFile = remame[0]
	}

	// this is my variable for uploaded files details. We will return it as a first returned parameter
	var uploadedFiles []*UploadedFile

	//check if my max file is set, if not make it a Gigabite
	if t.MaxFileSize == 0 {
		t.MaxFileSize = 1024 * 1024 * 1024
	}

	// Create directory if it dosn't exists
	// use function we created in this package
	err := t.CreateDirIfNotExist(uploadDir)
	if err != nil {

		return nil, errors.New("error creating directory")

	}

	// when we send Post request with files we want to check if there any error occured
	// take the max file size from Tools struct
	err = r.ParseMultipartForm(int64(t.MaxFileSize))
	if err != nil {

		return nil, errors.New("the uploaded file is too big")

	}

	//look at the request and see if any files are stored there
	// the first part of look give me the file headers from my form (from HTTP POST)
	for _, fileHeaders := range r.MultipartForm.File {

		for _, header := range fileHeaders {

			//this is inline function that return slice of uploaded file info
			uploadedFiles, err = func(uploadedFiles []*UploadedFile) ([]*UploadedFile, error) {

				//this is a variable for single file, we store the info about single file
				var uploadedFile UploadedFile

				//this opens file from request
				insideFile, err := header.Open()
				if err != nil {
					return nil, err
				}
				defer insideFile.Close()

				// find out what kind of file it is
				// we want to look at first 512 bites of the file to check what file it is
				buff512 := make([]byte, 512)
				//read my buff (which is first 512 charachters)
				_, err = insideFile.Read(buff512)
				if err != nil {
					return nil, errors.New("error reading file type")
				}

				// check if file type is permitted

				//by default my allowed file type is false, se we only permit the files we checked
				allowed := false

				//this determine file type from my 512 charachters of the file
				fileType := http.DetectContentType(buff512)

				// these are file types we allow to upload to server
				// Now you can specify file type in your Tools variable
				//allowedTypes := []string{"image/jpeg", "image/png", "image/gif"}

				// check if my file types exist in "allowedTypes" list
				// if yes, allowed = true
				if len(t.AllowedFileTypes) > 0 {

					for _, x := range t.AllowedFileTypes {
						if strings.EqualFold(fileType, x) {
							allowed = true
							fmt.Println("My file type is:", fileType)
						}
					}
				} else {
					allowed = true

				}

				if !allowed {
					return nil, errors.New("file type is not allowed")
				}

				// return to begining of the file (above we read first 512 bytes, now we want to start over)
				_, err = insideFile.Seek(0, 0) //this gets you to the beginign of the first byte of the file
				if err != nil {
					return nil, err
				}

				// at this point we checked if uploaded file has proper size and type is valid
				// now, if rename = true we generate new name and save file with new name,
				// otherwise create file with the same name
				if renameFile {
					// here we renaming file to random string and keep the same extention
					// as in original file (by running filepath.Ext(filename))
					uploadedFile.NewFileName = fmt.Sprintf("%s%s", t.RandomStringGenerator(12), filepath.Ext(header.Filename))

				} else {
					//this is for the case when user chose not to rename file,
					//so the new file name = old file name
					uploadedFile.NewFileName = header.Filename
				}

				uploadedFile.OriginalFileName = header.Filename

				//MILESTONE: here we save file to server disk

				// this is variable for our new file
				var outfile *os.File
				defer outfile.Close()

				if outfile, err := os.Create(filepath.Join(uploadDir, uploadedFile.NewFileName)); err != nil {
					return nil, err
				} else {
					//this line copies file from the one we get from Post requers to server directory
					fileSize, err := io.Copy(outfile, insideFile)
					if err != nil {
						return nil, errors.New("error while coping files")
					}
					uploadedFile.FileSize = fileSize
				}

				// final step - we know file names and sizes, lets build out variable for uploaded files details
				uploadedFiles = append(uploadedFiles, &uploadedFile)

				return uploadedFiles, nil

			}(uploadedFiles)
			if err != nil {
				return uploadedFiles, err
			}

		}

	}

	// final step, if nothing went wring we will return info about uploaded file and no error
	return uploadedFiles, nil
}

// UploadOneFile upload single file and return info about that file
// it utilizes function we created above for uploading file
func (t *Tools) UploadOneFile(r *http.Request, uploadDir string, remame ...bool) (*UploadedFile, error) {

	// by default rename will be true and we will rename each file to random string
	renameFile := true

	// if we have some boolean value from function we will use these values instead of default
	if len(remame) > 0 {
		renameFile = remame[0]
	}

	files, err := t.UploadFiles(r, uploadDir, renameFile)
	if err != nil {
		return nil, err
	}

	return files[0], nil

}

// CreateDirIfNotExist check if directory exist, if not it creates it
func (t *Tools) CreateDirIfNotExist(dirPath string) error {

	// this constant represent permission mode
	const permMode = 0755

	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		err := os.MkdirAll(dirPath, permMode)
		fmt.Printf("Directory %s has been created", dirPath)

		if err != nil {
			return err
		}
	}

	return nil

}

//-----------END OF FILE UPLOAD SECTION------------------------------

//-----------BEGINING OF SLUG CREATION SECTION-------------------------
// This function turn string to slug (browser URL safe text)
// Ex: "Hello, world!"" to "hellow-world"

// Slugify is a simple function turning string to slug
func (t *Tools) Slugify(s string) (string, error) {

	// check if empty string was send as function parameter
	if s == "" {
		return "", errors.New("empty string is not permitted")
	}

	// this will allow to accept only small letters from a-z and any digits, that all

	var re = regexp.MustCompile(`[^a-z\d]+`)

	//Trim removes extra spaces, then ToLower change everything to lower case
	slug := strings.Trim(re.ReplaceAllString(strings.ToLower(s), "-"), "-")

	// check if at least one charachtes was valid
	if len(slug) == 0 {
		return "", errors.New("after removing all characters, slug is zero length")
	}

	return slug, nil
}

//-----------END OF SLUG CREATION SECTION------------------------------

//*******************************************************************
//-----------BEGINING OF IMAGE PLOAD SECTION-------------------------
// This function used to upload files from browser to server

//-----------END OF FILE UPLOAD SECTION------------------------------
