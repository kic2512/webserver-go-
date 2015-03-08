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

var STATUS_MAP = map[string]string{
	"200": "200 OK",
	"404": "404 Not Found",
	"403": "403 Forbidden",
	"405": "405 Method Not Allowed",
}

var RESP_HEADERS = map[string]string{
	"status":         "HTTP/1.1 %s\r\n",
	"date":           "Date: %s\r\n",
	"content_type":   "Content-Type: %s\r\n",
	"content_length": "Content-Length: %d\r\n",
	"server":         "Server: GoGoSeRvEr\r\n",
	"connection":     "Connection: close\r\n",
}

var extentions_set = []string{".html", ".jpg", ".jpeg", ".png", ".gif", ".css", ".js", ".swf", ".txt"}
var index_file = "index.html"

func main() {
	i := 1
	DOCUMENT_ROOT := ""
	ncpu := 1
	var err error
	for i < len(os.Args) {
		switch os.Args[i] {
		case "-r":
			i += 1
			DOCUMENT_ROOT = os.Args[i]
		case "-c":
			i += 1
			ncpu, err = strconv.Atoi(os.Args[i])
			if err != nil {
				fmt.Println("-c is a numeric flag")
				os.Exit(-1)
			}
		default:
			i += 1
		}

	}
	runtime.GOMAXPROCS(ncpu)
	port := ":80"
	address, err := net.ResolveTCPAddr("127.0.0.1", port)
	checkError(err)
	listener, err := net.ListenTCP("tcp", address)
	for {
		conn, err := listener.Accept()
		if err == nil && conn != nil {
			go handleClient(conn, DOCUMENT_ROOT)
		} else {
			fmt.Println(err.Error())
		}
	}

}

func checkError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
		os.Exit(1)
	}
}

func handleClient(conn net.Conn, DOCUMENT_ROOT string) {
	defer conn.Close()
	var buf [1024 * 8]byte
	_, err := conn.Read(buf[0:])
	if err != nil {
		return
	}

	re11, _ := regexp.Compile(`(GET) (.*) HTTP.*`)
	re12, _ := regexp.Compile(`(HEAD) (.*) HTTP.*`)
	re2, _ := regexp.Compile(`(?m)^Host: (.*)`)

	header_type := re11.FindStringSubmatch(string(buf[:]))
	if header_type == nil { // if request is not GET
		header_type = re12.FindStringSubmatch(string(buf[:]))
		_ = re2.FindStringSubmatch(string(buf[:]))
	}
	if header_type != nil { // if request is GET or HEAD
		method := header_type[1]
		request_type := header_type[2]
		makeResponse(conn, request_type, method, DOCUMENT_ROOT)
	} else {
		response := fmt.Sprintf(RESP_HEADERS["status"], STATUS_MAP["405"])
		_, _ = conn.Write([]byte(response))
		_, _ = conn.Write([]byte("\r\n"))
	}
}
func makeResponse(conn net.Conn, query, method, DOCUMENT_ROOT string) {
	url_path, _ := url.Parse(query)

	file_name, mime_type, err := determinate_mime(url_path.Path[1:]) // remove first slash
	STATUS_CODE := STATUS_MAP["200"]

	if err != nil {
		STATUS_CODE = STATUS_MAP["404"]
	}

	dat, local_code, err := check_n_read_file(DOCUMENT_ROOT, file_name)

	if err != nil {
		STATUS_CODE = STATUS_MAP[local_code]
	}

	status := fmt.Sprintf(RESP_HEADERS["status"], STATUS_CODE)
	content_type := fmt.Sprintf(RESP_HEADERS["content_type"], mime_type)
	date := fmt.Sprintf(RESP_HEADERS["date"], time.Now().Format(time.RFC850))
	content_length := fmt.Sprintf(RESP_HEADERS["content_length"], len(dat))
	server := RESP_HEADERS["server"]
	connection := RESP_HEADERS["connection"]

	var log_map = map[string]string{
		"status":    status,
		"content":   content_type,
		"file_name": file_name,
		"body_len":  string(len(dat)),
	}

	_, _ = conn.Write([]byte(status))
	_, _ = conn.Write([]byte(date))
	_, _ = conn.Write([]byte(content_type))
	_, _ = conn.Write([]byte(content_length))
	_, _ = conn.Write([]byte(server))
	_, _ = conn.Write([]byte(connection))
	_, _ = conn.Write([]byte("\r\n"))

	if STATUS_CODE == STATUS_MAP["200"] && method == "GET" {
		_, _ = conn.Write(dat[0:])
	} else {
		writelog(log_map)
	}
}
func get_mime_type_by_ext(extention string) string {
	result := ""
	switch extention {
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

func idx_after_ext(path string, extentions_set string) int {
	return strings.Index(path, extentions_set) + len(extentions_set)
}
func determinate_mime(file_name string) (string, string, error) {
	mime_type := get_mime_type_by_ext(".html")
	result_name := index_file
	for s := range extentions_set {
		if strings.Contains(file_name, extentions_set[s]) {
			mime_type = get_mime_type_by_ext(extentions_set[s])
			sub_str_end_idx := idx_after_ext(file_name, extentions_set[s])
			result_name = file_name[0:sub_str_end_idx]
			return result_name, mime_type, nil
		}
	}
	if len(file_name) > 1 {
		result_name = file_name + index_file
	}
	return result_name, mime_type, nil
}

func check_n_read_file(DOCUMENT_ROOT, file_name string) ([]byte, string, error) {
	code := "200"
	fmt.Println("Seek:")
	fmt.Println(DOCUMENT_ROOT + file_name)
	dat, err := ioutil.ReadFile(DOCUMENT_ROOT + file_name)
	if os.IsPermission(err) {
		code = "403"
	}
	if os.IsNotExist(err) {
		if strings.Contains(file_name, index_file) {
			code = "403"
		} else {
			code = "404"
		}
	}
	return dat, code, err
}
func writelog(log_map map[string]string) {

	fmt.Println("status code:")
	fmt.Println(log_map["status"])
	fmt.Println("mime:")
	fmt.Println(log_map["content"])
	fmt.Println("file name : ")
	fmt.Print(log_map["file_name"])
	fmt.Print("Q")
	fmt.Println("")
	fmt.Print("response start")
	fmt.Println("")
	fmt.Println(log_map["body_len"])
	fmt.Println("response finish")
}
