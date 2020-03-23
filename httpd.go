// httpd
package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/url"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

var statusesMap = map[string]string {
	"200": "200 OK",
	"404": "404 Not Found",
	"403": "403 Forbidden",
	"405": "405 Method Not Allowed",
	"415": "415 Unsupported Media Type",
}

var responsesHeaders = map[string]string {
	"status":         "HTTP/1.1 %s\r\n",
	"date":           "Date: %s\r\n",
	"content_type":   "Content-Type: %s\r\n",
	"content_length": "Content-Length: %d\r\n",
	"server":         "Server: GoGoSeRvEr\r\n",
	"connection":     "Connection: close\r\n",
}

var indexFile = "index.html"

func main() {
	documentRoot := ""
	cpuCount := 2
	var err error

	fmt.Print("Len args: ")
	fmt.Println(len(os.Args))

	i := 1
	for i < len(os.Args) {
		switch os.Args[i] {
		case "-r":
			i += 1
			documentRoot = os.Args[i]
			fmt.Println("Document root: " + documentRoot)
		case "-c":
			i += 1
			cpuCount, err = strconv.Atoi(os.Args[i])
			if err != nil {
				fmt.Println("-c is a numeric flag")
				os.Exit(-1)
			}
			fmt.Printf("cpu count: %d\n", cpuCount)
		default:
			i += 1
		}
	}

	runtime.GOMAXPROCS(cpuCount)
	address, err := net.ResolveTCPAddr("tcp", "127.0.0.1:8080")
	checkError(err)

	listener, err := net.ListenTCP("tcp", address)
	checkError(err)

	for {
		conn, err := listener.Accept()
		if err == nil && conn != nil {
			go handleClient(conn, documentRoot)
		} else {
			fmt.Println(err.Error())
		}
	}
}

func checkError(err error) {
	if err != nil {
		var ioErr error
		_, ioErr = fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
		if ioErr != nil {
			fmt.Println(ioErr.Error())
		}
		os.Exit(1)
	}
}

func handleClient(conn net.Conn, DocumentRoot string) {
	defer conn.Close()
	var buf [1024 * 8]byte
	_, err := conn.Read(buf[0:])
	if err != nil {
		return
	}

	re11, _ := regexp.Compile(`(GET) (.*) HTTP.*`)
	re12, _ := regexp.Compile(`(HEAD) (.*) HTTP.*`)
	re2, _ := regexp.Compile(`(?m)^Host: (.*)`)

	headerType := re11.FindStringSubmatch(string(buf[:]))
	if headerType == nil { // if request is not GET
		headerType = re12.FindStringSubmatch(string(buf[:]))
		_ = re2.FindStringSubmatch(string(buf[:])) // TODO UNDERSTAND WTF ???
	}
	if headerType != nil { // if request is GET or HEAD
		method := headerType[1]
		request := headerType[2]
		makeResponse(conn, request, method, DocumentRoot)
	} else {
		response := fmt.Sprintf(responsesHeaders["status"], statusesMap["405"])
		_, _ = conn.Write([]byte(response))
		_, _ = conn.Write([]byte("\r\n"))
	}
}
func makeResponse(conn net.Conn, query, method, DocumentRoot string) {
	urlPath, _ := url.Parse(query)

	fileName, mimeType, err := determinateMime(urlPath.Path[1:]) // remove first slash

	if err != nil {
		return
	}
	var dat []byte
	var responseCode string
	var isSuccessCode bool
	if mimeType == "" {
		responseCode = "415"
		isSuccessCode = false
	} else {
		dat, responseCode, err = readFile(DocumentRoot, fileName)
		if err != nil {
			return
		}
		isSuccessCode = true
	}

	statusCode := statusesMap[responseCode]
	status := fmt.Sprintf(responsesHeaders["status"], statusCode)
	date := fmt.Sprintf(responsesHeaders["date"], time.Now().Format(time.RFC850))

	var contentType, contentLength, server, connection string
	if isSuccessCode {
		contentType = fmt.Sprintf(responsesHeaders["content_type"], mimeType)
		contentLength = fmt.Sprintf(responsesHeaders["content_length"], len(dat))
		server = responsesHeaders["server"]
		connection = responsesHeaders["connection"]
	}

	_, _ = conn.Write([]byte(status))
	_, _ = conn.Write([]byte(date))

	if isSuccessCode {
		_, _ = conn.Write([]byte(contentType))
		_, _ = conn.Write([]byte(contentLength))
		_, _ = conn.Write([]byte(server))
		_, _ = conn.Write([]byte(connection))
		_, _ = conn.Write([]byte("\r\n"))
	}

	if statusCode == statusesMap["200"] && method == "GET" {
		_, _ = conn.Write(dat[0:])
	} else {
		return
	}
}
func getMimeTypeByExt(extension string) string {
	result := ""
	extension = strings.ToLower(extension)
	switch extension {
	case ".html":
		result = "text/html"
	case ".txt":
		result = "text/plain"
	case ".jpg", ".jpeg":
		result = "image/jpeg"
	case ".png":
		result = "image/png"
	case ".gif":
		result = "image/gif"
	case ".css":
		result = "text/css"
	case ".js":
		result = "text/javascript"
	case ".swf":
		result = "application/x-shockwave-flash"
	default:
		result = ""
	}
	return result
}

func determinateMime(fileName string) (string, string, error) {
	var mimeType string
	var resultName string

	nameParts := strings.Split(fileName, ".")
	if len(nameParts) > 1 {
		lastExt := nameParts[len(nameParts)-1]
		mimeType = getMimeTypeByExt("." + lastExt)
		resultName = fileName
	} else {
		mimeType = getMimeTypeByExt(".html")
		resultName = indexFile
	}

	return resultName, mimeType, nil
}

func readFile(DocumentRoot, fileName string) ([]byte, string, error) {
	code := "200"
	dat, err := ioutil.ReadFile(DocumentRoot + fileName)
	if os.IsPermission(err) {
		code = "403"
	}
	if os.IsNotExist(err) {
		if strings.Contains(fileName, indexFile) {
			code = "403"
		} else {
			code = "404"
		}
	}
	return dat, code, err
}
