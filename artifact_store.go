package main

import (
	"crypto/md5"
	"encoding/hex"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"bytes"
)

var Build string
var Version string
var Hash string
var Dirty string

const UplURL = "/upload"
const DirURL = "/dir"

type Config struct {
        IsServer		*bool
	ServerPort		*int
	ServerAddress		*string
        ServerDir		*string
        ClientFileType		*string
        ClientDescription	*string
	ClientArchitecture	*string
	ClientFileName		*string
	ClientScope		*string
	ClientVersion		*string

}
type Metadata struct {
	Description  string `json:"description"`
	Type         string `json:"type"`
	Architecture string `json:"architecture"`
	Scope        string `json:"scope"`
	CreationDate string `json:"creationdate"`
	CreatedAt    string `json:"created_at"`
	FileName     string `json:"FileName"`
	Version      string `json:"version"`
}
var ServerDir string

func printConfig(c *Config) {
	fmt.Printf("IsServer %t\n", *(*c).IsServer)
	fmt.Printf("ServerPort %d\n", *(*c).ServerPort)
	fmt.Printf("ServerAddress %s\n", *(*c).ServerAddress)
	fmt.Printf("ServerDir %s\n", *(*c).ServerDir)
	fmt.Printf("ClientFileType %s\n", *(*c).ClientFileType)
	fmt.Printf("ClientDescription %s\n", *(*c).ClientDescription)
	fmt.Printf("ClientArchitecture %s\n", *(*c).ClientArchitecture)
	fmt.Printf("ClientFileName %s\n", *(*c).ClientFileName)
	fmt.Printf("ClientScope %s\n", *(*c).ClientScope)
	fmt.Printf("ClientVersion %s\n", *(*c).ClientVersion)

}

func generateHTMLTable(directory string) (string, error) {
	files, err := ioutil.ReadDir(directory)
	if err != nil {
		return "", err
	}

	htmlTable := "<table border='1'><tr><th>Description</th><th>Type</th><th>architecture</th><th>scope</th><th>Version</th><th>original file name</th></tr>"

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".meta") {
			continue
		}

		metaFilePath := filepath.Join(directory, file.Name())
		metaJSON, err := ioutil.ReadFile(metaFilePath)
		if err != nil {
			return "", err
		}

		var metadata Metadata
		err = json.Unmarshal(metaJSON, &metadata)
		if err != nil {
			return "", err
		}

		fileName := strings.TrimSuffix(file.Name(), ".meta")
		fileLink := fmt.Sprintf("<a href='%s'>%s</a>", fileName, metadata.FileName)

		htmlTable += fmt.Sprintf("<tr><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td></tr>", metadata.Description, metadata.Type, metadata.Architecture, metadata.Scope, metadata.Version, fileLink)
	}

	htmlTable += "</table>"

	return htmlTable, nil
}

func confOk(c *Config) bool {
	if *(*c).IsServer {
		return true
	}
	return *(*c).ClientFileType != "" && *(*c).ClientDescription != "" && *(*c).ClientArchitecture != "" && *(*c).ClientFileName != "" && *(*c).ClientScope != "" &&*(*c).ClientVersion != "";
}


func main() {
	var c Config
	fmt.Printf("artifact_store Ver. %s.%s (%s) %s\n", Version, Build, Hash, Dirty)
	if len(os.Args) <2 {
		flag.PrintDefaults()
	}
	c.IsServer =		flag.Bool("server", false, "Run as server")
	c.ServerPort =		flag.Int("port", 8080, "If server, port to bind the service, if client, server port to connect")
	c.ServerAddress =	flag.String("address", "0.0.0.0", "If server, address to bind the service, if client, server to connect")
	c.ServerDir =		flag.String("dir", "./artifacts/", "If server, directory where store the artifacts")
	c.ClientDescription =	flag.String("description", "", "Artifact description")
	c.ClientFileType =	flag.String("type", "", "Artifact type")
	c.ClientArchitecture =	flag.String("architecture", "", "Artifact architecture")
	c.ClientScope =		flag.String("scope", "", "Artifact scope")
	c.ClientFileName =	flag.String("file", "", "Original artifact file name")
	c.ClientVersion=	flag.String("version", "", "All you need to define this thing version")
	flag.Parse()

	if confOk(&c) {
		if *c.IsServer {
			startServer(&c)
		} else {
			err := uploadFile(&c)
			if err != nil {
				fmt.Println("Error uploading file:", err)
				os.Exit(1)
			}
		}
	}

}

func uploadFile(c *Config) error {
	hostname, _ := os.Hostname()
	metadata := Metadata{
		Description:  *(*c).ClientDescription,
		Type:         *(*c).ClientFileType,
		Architecture: *(*c).ClientArchitecture,
		Scope:        *(*c).ClientScope,
		CreationDate: time.Now().Format(time.RFC3339),
		CreatedAt:    hostname,
		FileName:     *(*c).ClientFileName,
		Version:      *(*c).ClientVersion,

	}

	fileContents, err := ioutil.ReadFile(metadata.FileName)
	if err != nil {
		return err
	}
	metadata.FileName = filepath.Base(metadata.FileName)
	fileContentsBase64 := base64.StdEncoding.EncodeToString(fileContents)

	payload := struct {
		Metadata   Metadata `json:"metadata"`
		FileBase64 string   `json:"file_base64"`
	}{
		Metadata:   metadata,
		FileBase64: fileContentsBase64,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}


	url := fmt.Sprintf("http://%s:%d%s", *(*c).ServerAddress, *(*c).ServerPort, UplURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil

}

func produceDir(w http.ResponseWriter, r *http.Request) {
	response, _ := generateHTMLTable(ServerDir)
        w.Write([]byte(response))


}
func startServer(c *Config) {
	http.HandleFunc(UplURL, uploadHandler)
	http.HandleFunc(DirURL, produceDir)
	port := strconv.Itoa(*(*c).ServerPort)
	ServerDir = *(*c).ServerDir
	fs := http.FileServer(http.Dir(ServerDir))
	http.Handle("/", fs)
	fmt.Println("Server listening on port", port)
	err := http.ListenAndServe(":"+ port, nil)
	if err != nil {
		fmt.Println("Error starting server:", err)
	}
}


func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}

	var data struct {
		Metadata   Metadata `json:"metadata"`
		FileBase64 string   `json:"file_base64"`
	}

	if err := json.Unmarshal(body, &data); err != nil {
		http.Error(w, "Error parsing JSON", http.StatusBadRequest)
		return
	}
	fileContents, err := decodeBase64(data.FileBase64)
	if err != nil {
		http.Error(w, "Error decoding base64 file contents", http.StatusBadRequest)
		return
	}

	fileMD5 := calculateMD5(fileContents)

	filePath := filepath.Join(ServerDir, fileMD5)
	metaFilePath := filePath + ".meta"

	file, err := os.Create(filePath)
	if err != nil {
		http.Error(w, "Error creating file", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	_, err = file.Write(fileContents)
	if err != nil {
		http.Error(w, "Error writing file contents", http.StatusInternalServerError)
		return
	}

	metaFile, err := os.Create(metaFilePath)
	if err != nil {
		http.Error(w, "Error creating meta file", http.StatusInternalServerError)
		return
	}
	defer metaFile.Close()

	metaJSON, err := json.Marshal(data.Metadata)
	if err != nil {
		http.Error(w, "Error serializing metadata to JSON", http.StatusInternalServerError)
		return
	}

	_, err = metaFile.Write(metaJSON)
	if err != nil {
		http.Error(w, "Error writing metadata to meta file", http.StatusInternalServerError)
		return
	}

	response := fmt.Sprintf("File %s uploaded successfully.", fileMD5)
	w.Write([]byte(response))
}

func decodeBase64(data string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(data)
}

func calculateMD5(data []byte) string {
	hasher := md5.New()
	hasher.Write(data)
	return hex.EncodeToString(hasher.Sum(nil))
}
