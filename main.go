package main

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/drone/go-convert/convert/harness/downgrader"
	"github.com/jamie-harness/go-convert/convert/jenkinsjson"
)

func uploadFile(w http.ResponseWriter, r *http.Request) {
	// Maximum upload of 10 MB files
	r.ParseMultipartForm(10 << 20)

	// Get handler for filename, size and headers
	file, handler, err := r.FormFile("jenkinsjsonfile")
	if err != nil {
		fmt.Println("Error Retrieving the File")
		fmt.Println(err)
		return
	}

	defer file.Close()
	// fmt.Printf("Uploaded File: %+v\n", handler.Filename)
	// fmt.Printf("File Size: %+v\n", handler.Size)
	// fmt.Printf("MIME Header: %+v\n", handler.Header)

	// Create file
	dst, err := os.Create(handler.Filename)
	defer dst.Close()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Copy the uploaded file to the created file on the filesystem
	if _, err := io.Copy(dst, file); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	converter := jenkinsjson.New(
		jenkinsjson.WithDockerhub(""),
		jenkinsjson.WithKubernetes("", ""),
	)
	converted, err := converter.ConvertFile(handler.Filename)
	if err != nil {
		fmt.Println("error: ", err)
	}
	// fmt.Println(converted)
	d := downgrader.New(
		downgrader.WithCodebase("", ""),
		downgrader.WithDockerhub(""),
		downgrader.WithKubernetes("", ""),
		downgrader.WithName("default"),
		downgrader.WithOrganization("default"),
		downgrader.WithProject("default"),
	)
	converted, err = d.Downgrade(converted)
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
	http.HandleFunc("/convert-to-harness", uploadHandler)
	http.ListenAndServe(":8088", nil)
}
