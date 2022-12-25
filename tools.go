package toolkit

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

//-----------BEGINING OF RANDOM STRING GENERATION SECTION-----------------

// This constant contains all possible characters, that we can use for randomly generated string
// we can use it for example, for creating file names for Linux system.
const randomStringSource = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_+"

// Tools is the custom type used to instantiate this module.
// Any variables of this type will have access to all methods with reciever *Tools [func(t *Tools){}]
// This technic used to share methods from the modules with other programs
type Tools struct {
	MaxFileSize        int
	AllowedFileTypes   []string //You can provide what type of file you allow (imgs/pdf/docs etc)
	MaxJSONSize        int      // Provide maximum size of JSON payload
	AllowUnknownFields bool     // allow or not unknown fields in JSON payload
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
func (t *Tools) UploadFiles(r *http.Request, uploadDir string, rename ...bool) ([]*UploadedFile, error) {

	// by default rename will be true and we will rename each file to random string
	renameFile := true

	// if we have any boolean value (true or false) from function we will use these values instead of default
	if len(rename) > 0 {
		renameFile = rename[0]
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
func (t *Tools) UploadOneFile(r *http.Request, uploadDir string, rename ...bool) (*UploadedFile, error) {

	// by default rename will be true and we will rename each file to random string
	renameFile := true

	// if we have some boolean value from function we will use these values instead of default
	if len(rename) > 0 {
		renameFile = rename[0]
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

//-----------BEGINING OF DOWNLOAD FILE SECTION-------------------------
// This function used to download static files from server to your local folder

// DownloadStaticFile downloads file from server to local drive
// As a parameters it takes http r, w, path, file and name you want to rename file to
func (t *Tools) DownloadStaticFile(w http.ResponseWriter, r *http.Request, p, file, displayName string) {

	filePath := path.Join(p, file)

	// this sets header and forces file to be downloaded instead of displayed
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", displayName))

	//download the file
	http.ServeFile(w, r, filePath)

}

//-----------END OF DOWNLOAD FILE SECTION------------------------------

//-----------BEGINING OF JSON READ SECTION-------------------------
// This function reads the data from JSON and convert it to Go structure

// Stuct for response we send back to server. Two first fileds are mandatory, Data is optional
type JSONResponse struct {
	Error   bool        `json:"error"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// ReadJSON allows to read incoming JSON payloads (as a body of incoming request)
// and converts to provided Go data structure
func (t *Tools) ReadJSON(w http.ResponseWriter, r *http.Request, data interface{}) error {

	// 1.0 Limit max size of the incoming JSON
	maxBytesJson := 1024 * 1024 // around 1 MGB of payload size

	if t.MaxJSONSize != 0 {
		maxBytesJson = t.MaxJSONSize
	}

	// 1.1 Read request body and check if it less than size we identified
	// MaxBytesReader prevents clients from accidentally or maliciously sending a large request
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytesJson))

	// 1.2 decode payload from request Body
	dec := json.NewDecoder(r.Body)

	// this part check if paramer "AllowUnknownFields" from our Tools
	// is false, then it forces our decoded result reject unknown fields
	// we won't process JSON fields that we don't know about
	if !t.AllowUnknownFields {
		dec.DisallowUnknownFields()
	}

	// 1.3 Take our decoded values and turn them to "data" (the struct we provided as a function parameter)
	// ^Decode reads the next JSON-encoded value from its input and stores it in the value
	// !->This is main comand that turns decoded JSON to Golang data struct
	err := dec.Decode(data)
	if err != nil {
		// improve error handling
		// ^A SyntaxError is a description of a JSON syntax error
		// ^An UnmarshalTypeError describes a JSON value that was not appropriate for a value of a specific Go type
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError
		var invalidUnmarshalError *json.InvalidUnmarshalError

		switch {
		case errors.As(err, &syntaxError):
			return fmt.Errorf("body contains badly-formed JSON (at character %d)", syntaxError.Offset)

		case errors.Is(err, io.ErrUnexpectedEOF):
			return errors.New("body contains badly formed JSON")

		case errors.As(err, &unmarshalTypeError):
			if unmarshalTypeError.Field != "" {
				return fmt.Errorf("body contains incorrect JSON type for field %q", unmarshalTypeError.Field)
			}
			return fmt.Errorf("body contains incorrect JSON type for field %d", unmarshalTypeError.Offset)

		// check if there is no body at all
		case errors.Is(err, io.EOF):
			return errors.New("JSON body must not be empty")

		// check for unknown field in the body
		case strings.HasPrefix(err.Error(), "json: unknown field"):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field")
			return fmt.Errorf("body contains unknown key %s", fieldName)

		// check if request body is too large
		case err.Error() == "http: request body too large":
			return fmt.Errorf("body must not be larger then %d bytes", maxBytesJson)

		case errors.As(err, &invalidUnmarshalError):
			return fmt.Errorf("error unmarshaling JSON: %s", err.Error())

		default:
			return err
		}

	}

	// 1.4 Check if we only recieving one file
	err = dec.Decode(&struct{}{})
	if err != io.EOF {
		return errors.New("body should contain only one JSON value")
	}

	return nil

}

//-----------END OF JSON READ SECTION------------------------------

//-----------BEGINING OF JSON WRITE SECTION-------------------------
// This function used to write JSON back to response

// WriteJSON takes some data (as function parameter, and writes it to http response along with status code)
// header is variadic parameter for http headers. We might send headers or might not
func (t *Tools) WriteJSON(w http.ResponseWriter, status int, data interface{}, headers ...http.Header) error {

	// we take our data (some struct and turn it (marshal) to JSON)
	out, err := json.Marshal(data)
	if err != nil {
		return err
	}

	// we need to check if have any "headers" paramenter in our function parameters and send it as a header
	// back in our resonse
	if len(headers) > 0 {
		for key, value := range headers[0] {
			w.Header()[key] = value //key is what the header type is, and value is what it set to
		}
	}

	// now set content type of our response to "applicatiom/json"
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	// !->This is main comand that writes our data to http respnse as a JSON
	_, err = w.Write(out)
	if err != nil {
		return err
	}
	return nil
}

//-----------END OF JSON WRITE SECTION------------------------------

//-----------BEGINING OF JSON ERROR SECTION-------------------------
// This function write the error to JSON resonse

// ErrorJSON takes an error and optionally status code, then generates JSON resonse
// and sends as http response
func (t *Tools) ErrorJSON(w http.ResponseWriter, err error, status ...int) error {

	//this is our defualt status code
	statusCode := http.StatusBadRequest // status 400

	// check if stasu code exosts as function parameter send it as a status code
	if len(status) > 0 {
		statusCode = status[0]
	}

	//variable payload of type JSONResponse (our custom type we created at line 327)
	var payload JSONResponse

	// build the payload that we send back
	payload.Error = true
	payload.Message = err.Error() //the message of payload is our error

	return t.WriteJSON(w, statusCode, payload)

}

//-----------END OF JSON ERROR SECTION------------------------------

// -----------BEGINING OF PUSH JSON TO REMOTE SECTION-------------------------

// PushJSONToRemote takes URL, some Data and optional client and pushes data to remote sever
// If no client is specified we use standard http.Client
// Function returns the resonse (as *http.Response type) and status code (e.g 200)
func (t *Tools) PushJSONToRemote(uri string, data interface{}, client ...*http.Client) (*http.Response, int, error) {

	// create JSON from our "data" function parameter (must be struct)
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, 0, errors.New("error building JSON")

	}

	// check for custom http client
	httpClient := &http.Client{}

	// check if client is part of function parameter
	// if it's exists asign our variable to it
	if len(client) > 0 {
		httpClient = client[0]
	}

	// build the request (with headers)
	request, err := http.NewRequest("POST", uri, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, 0, errors.New("error sending JSON to uri")

	}
	// set headers
	request.Header.Set("Content-Type", "application/json")

	// sent it to remote server (as POST), it returns response back
	// ^Do sends an HTTP request and returns an HTTP response, following
	// policy (such as redirects, cookies, auth) as configured on the client.
	response, err := httpClient.Do(request)
	if err != nil {
		return nil, 0, errors.New("error gettng response from remote server")

	}
	defer response.Body.Close()

	// retrieve/return the resoponse
	return response, response.StatusCode, nil

}
