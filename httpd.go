package main

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

var statusesMap = map[string]string{
	"200": "200 OK",
	"404": "404 Not Found",
	"403": "403 Forbidden",
	"405": "405 Method Not Allowed",
	"415": "415 Unsupported Media Type",
}

var responsesHeaders = map[string]string{
	"status":         "HTTP/1.1 %s\r\n",
	"date":           "Date: %s\r\n",
	"content_type":   "Content-Type: %s\r\n",
	"content_length": "Content-Length: %d\r\n",
	"server":         "Server: GoGoSeRvEr\r\n",
	"connection":     "Connection: close\r\n",
}

const indexFile = "index.html"

func main() {
	documentRoot := ""
	cpuCount := 2
	host := "0.0.0.0"
	port := "8080"
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
		case "-h":
			i += 1
			host = os.Args[i]
		case "-p":
			i += 1
			port = os.Args[i]
		default:
			i += 1
		}
	}

	log.SetFormatter(&log.JSONFormatter{})
	log.SetFormatter(&log.TextFormatter{
		DisableColors: false,
		FullTimestamp: true,
	})

	runtime.GOMAXPROCS(cpuCount)
	address, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%s", host, port))

	if err != nil {
		log.Fatal(fmt.Sprintf("Unable to connect host: %s; port: %s", host, port), err)
	}

	listener, err := net.ListenTCP("tcp", address)

	if err != nil {
		log.Fatal("Unable to establish TCP connection", err)
	}

	for {
		conn, err := listener.Accept()
		if err == nil && conn != nil {
			go handleClient(conn, documentRoot)
		} else {
			log.Error("can not handle client request", err, conn)
		}
	}
}

func handleClient(conn net.Conn, DocumentRoot string) {
	defer conn.Close()
	var buf [1024 * 8]byte
	_, err := conn.Read(buf[0:])
	if err != nil {
		log.Error(err.Error())
		return
	}

	re11, _ := regexp.Compile(`(GET) (.*) HTTP.*`)
	re12, _ := regexp.Compile(`(HEAD) (.*) HTTP.*`)
	re2, _ := regexp.Compile(`(?m)^Host: (.*)\r`)

	dataHost := re2.FindStringSubmatch(string(buf[:]))
	var requestHost string = ""
	if len(dataHost) > 1 {
		requestHost = dataHost[1]
	}

	headerType := re11.FindStringSubmatch(string(buf[:]))
	if headerType == nil { // if request is not GET
		headerType = re12.FindStringSubmatch(string(buf[:]))
	}
	if headerType != nil { // if request is GET or HEAD
		method := headerType[1]
		request := headerType[2]

		log.Info(fmt.Sprintf("request from %s url %s", requestHost, request))

		makeResponse(conn, request, method, DocumentRoot)
	} else {
		response := fmt.Sprintf(responsesHeaders["status"], statusesMap["405"])
		log.Warn(fmt.Sprintf("method not allowed;\n request: %s", buf[:]))
		_, _ = conn.Write([]byte(response))
		_, _ = conn.Write([]byte("\r\n"))
	}
}
func makeResponse(conn net.Conn, query, method, DocumentRoot string) {
	urlPath, _ := url.Parse(query)

	fileName, mimeType, err := determinateMime(urlPath.Path[1:]) // remove first slash

	if err != nil {
		log.Error(err.Error())
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
			log.Error(err.Error())
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
		log.Warn(fmt.Sprintf("empty response for query %s; method: %s", query, method))
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
	var dat []byte
	code := "200"
	absolutePath, err := filepath.Abs(DocumentRoot + fileName)

	if err != nil {
		log.Error(err.Error())
		return nil, "", err
	}

	if !strings.HasPrefix(absolutePath, DocumentRoot) { // don't return file which outside the project
		code = "403"
		return nil, code, nil
	}

	dat, err = ioutil.ReadFile(DocumentRoot + fileName)
	if os.IsPermission(err) {
		code = "403"
	} else {
		if os.IsNotExist(err) {
			code = "404"
		}
	}
	return dat, code, err
}
