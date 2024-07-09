package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
    "strings"

	"github.com/drone/go-convert/convert/harness/downgrader"
	"github.com/jamie-harness/go-convert/convert/jenkinsjson"
)

type JenkinsJSON struct {
	// TraceId     string `json:"traceId"`
	// SpanId      string `json:"spanId"`
	Name string `json:"name"`
	// ParentSpanId string `json:"parentSpanId"`
	// SpanName    string `json:"spanName"`
	// Children    []interface{} `json:"children"` // Use appropriate type if known
}

func cleanFileName(fileName string) string {
    // Remove unwanted characters and spaces
    re := regexp.MustCompile(`[^a-zA-Z0-9]`)
    cleanedName := re.ReplaceAllString(fileName, "")
    
    // Remove the file extension before cleaning
    extension := filepath.Ext(cleanedName)
    cleanedNameWithoutExt := strings.TrimSuffix(cleanedName, extension)
    
    return cleanedNameWithoutExt + extension
}

func uploadFile(w http.ResponseWriter, r *http.Request) {
	// Maximum upload of 10 MB files
	r.ParseMultipartForm(10 << 20)

	// Get handler for filename, size and headers
	file, _, err := r.FormFile("jenkinsjsonfile")
	if err != nil {
		fmt.Println("Error Retrieving the File")
		fmt.Println(err)
		return
	}
	defer file.Close()

	// Read the uploaded file into memory
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var jenkinsData JenkinsJSON
	err = json.Unmarshal(fileBytes, &jenkinsData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Use the `name` field from the JSON as the new filename
	newFilename := jenkinsData.Name 
	cleanedFileName := cleanFileName(newFilename) + ".json"
	// Create directory if it doesn't exist
	uploadDir := "TRACE_JSON"
	err = os.MkdirAll(uploadDir, os.ModePerm)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Create file with the new name
	filePath := filepath.Join(uploadDir, cleanedFileName)
	dst, err := os.Create(filePath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	fmt.Printf("Saving file as: %s\n", filePath)

	// Write the content to the new file
	if _, err := dst.Write(fileBytes); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Proceed with the conversion and downgrading using the new file name
	converter := jenkinsjson.New(
		jenkinsjson.WithDockerhub(DockerhubCredentials),
		jenkinsjson.WithKubernetes(KubernetesAPI, KubernetesToken),
	)
	converted, err := converter.ConvertFile(filePath)
	if err != nil {
		fmt.Println("error: ", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Downgrading using parameterized values from constant.go
	d := downgrader.New(
		downgrader.WithCodebase(CodebaseAPI, CodebaseToken),
		downgrader.WithDockerhub(DockerhubCredentials),
		downgrader.WithKubernetes(KubernetesAPI, KubernetesToken),
		downgrader.WithName(jenkinsData.Name),
		downgrader.WithOrganization(DowngraderOrg),
		downgrader.WithProject(DowngraderProject),
	)
	converted, err = d.Downgrade(converted)
	if err != nil {
		fmt.Println("error: ", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Output the converted pipeline to stdout and send it as an HTTP response
	os.Stdout.Write(converted)
	fmt.Fprintf(w, string(converted[:]))
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		uploadFile(w, r)
	}
}

func main() {
	http.HandleFunc(ConvertEndpoint, uploadHandler)
	http.ListenAndServe(ServerAddress, nil)
}
