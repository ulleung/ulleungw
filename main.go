package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/buger/jsonparser"
	"github.com/dustin/go-humanize"
	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/transform"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type Asset struct {
	browser_download_url string
}
type Release struct {
	message string
	tag_name string
	assets Asset
	id int
}
func main() {
	WRAPPER_VERSION := "1.0.0"

	args := os.Args[1:]

	doTranspile := true
	doIncludeDokList := true

	ulleungPath := ulleungPath()

	transpilerList := reverse(getFilesUnderPath(ulleungPath+"/transpiler"))
	compilerList := reverse(getFilesUnderPath(ulleungPath +"/compiler"))

	ulnFile := ""
	transpilerVersion := getTranspiler(transpilerList)
	compilerVersion := getCompiler(compilerList)

	runCommand := false

	var dokFiles []string

	currentFlag := "file"

	for _, arg := range args {
		if arg[0] == '-'{
			if arg[1] == '-' { // -- flag control
				currentFlag = arg[2:]
			} else { // - flag control
				boolFlag := arg[1:]
				switch boolFlag {
				case "t" : doTranspile = true
				case "c" : doTranspile = false
				case "no-doklist" : doIncludeDokList = false
				case "v" : fallthrough
				case "version" :
					fmt.Println("울릉 래퍼 버전 " + WRAPPER_VERSION)
					fmt.Println("설치된 트랜스파일러 목록 -")
					for _, fileName := range transpilerList {
						fmt.Println(fileName)
					}
					fmt.Println("설치된 컴파일러 목록 -")
					for _, fileName := range compilerList {
						fmt.Println(fileName)
					}
				case "h" : fallthrough
				case "help" : printHelp(WRAPPER_VERSION)
				case "get-stable" :
					jsonString := getHttp("https://api.github.com/repos/ulleung/ulleungt/releases/latest")
					_, err := jsonparser.GetString([]byte(jsonString), "message")
					if err == nil {
						fmt.Println("안정된 버전이 없습니다.")
						return
					}
					tag_name, _ := jsonparser.GetString([]byte(jsonString),  "tag_name")
					assetUrl, _ := jsonparser.GetString([]byte(jsonString),  "assets", "[0]", "browser_download_url")
					fmt.Println(tag_name + "을 설치합니다...")
					dwnErr := downloadFile(ulleungPath+"/transpiler/ulleungt-"+tag_name+".jar", assetUrl)
					if dwnErr != nil {
						fmt.Println("설치 실패")
					}
				case "get-recent" : fallthrough
				case "get-latest" :
					jsonString := getHttp("https://api.github.com/repos/ulleung/ulleungt/releases?per_page=1")
					tag_name, _ := jsonparser.GetString([]byte(jsonString), "[0]", "tag_name")
					assetUrl, _ := jsonparser.GetString([]byte(jsonString), "[0]", "assets", "[0]", "browser_download_url")
					fmt.Println(tag_name + "을 설치합니다...")
					err := downloadFile(ulleungPath+"/transpiler/ulleungt-"+tag_name+".jar", assetUrl)
					if err != nil {
						fmt.Println("설치 실패")
					}
				case "get-list" :
					jsonString := getHttp("https://api.github.com/repos/ulleung/ulleungt/releases?per_page=10")
					fmt.Println("버전 이름 | 버전 ID")
					jsonparser.ArrayEach([]byte(jsonString), func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
						tag_name, _ := jsonparser.GetString(value, "tag_name")
						id, _ := jsonparser.GetInt(value, "id")
						fmt.Println(tag_name + " | " + strconv.Itoa(int(id)))
					})
				}
			}
		} else { // content control
			if currentFlag == "file" {
				runCommand = true
				ulnFile = getAbsolutePath(arg)
			} else if currentFlag == "use-version" {
				if doTranspile {
					transpilerVersion = "ulleungt-"+arg+".jar"
				} else {
					compilerVersion = "ulleungc-"+arg+".jar"
				}
			} else if currentFlag == "dok" {
				dokFiles = append(dokFiles, getAbsolutePath(arg))
			} else if currentFlag == "get-specify" {
				jsonString := getHttp("https://api.github.com/repos/ulleung/ulleungt/releases/" + arg)
				_, err := jsonparser.GetString([]byte(jsonString), "message")
				if err == nil {
					fmt.Println("존재하지 않는 버전입니다.")
					return
				}
				tag_name, _ := jsonparser.GetString([]byte(jsonString),  "tag_name")
				assetUrl, _ := jsonparser.GetString([]byte(jsonString),  "assets", "[0]", "browser_download_url")
				fmt.Println(tag_name + "을 설치합니다...")
				dwnErr := downloadFile(ulleungPath+"/transpiler/ulleungt-"+tag_name+".jar", assetUrl)
				if dwnErr != nil {
					fmt.Println("설치 실패")
				}
			}
		}
	}

	if !runCommand {
		return
	}

	runArgs := []string{"-jar"}

	if doTranspile {
		runArgs = append(runArgs, ulleungPath + "/transpiler/" + transpilerVersion)
	} else {
		runArgs = append(runArgs, ulleungPath + "/compiler/" + compilerVersion)
	}

	runArgs = append(runArgs, ulnFile)

	for _, dokFile := range dokFiles {
		runArgs = append(runArgs, dokFile)
	}

	if doIncludeDokList {
		jsonString := readDok(runPath() + "/doklist")

		var dokListFiles []string

		json.Unmarshal([]byte(jsonString), &dokListFiles)

		for _, dokFile := range dokListFiles {
			runArgs = append(runArgs, ulleungPath + "/dok/" + dokFile)
		}
	}
	output, e := exec.Command("java", runArgs...).Output()
	if e != nil {
		panic (e)
	}

	bufs := new(bytes.Buffer)

	wr := transform.NewWriter(bufs, korean.EUCKR.NewDecoder())
	wr.Write(output)
	wr.Close()

	fmt.Println(bufs)
}

type WriteCounter struct {
	Total uint64
}
func (wc *WriteCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.Total += uint64(n)
	wc.printProgress()
	return n, nil
}
func (wc WriteCounter) printProgress() {
	fmt.Printf("\r%s", strings.Repeat(" ", 35))
	fmt.Printf("\r설치 %s 완료", humanize.Bytes(wc.Total))
}
func downloadFile(filepath string, url string) error {
	out, err := os.Create(filepath + ".tmp")
	if err != nil {
		return err
	}
	resp, err := http.Get(url)
	if err != nil {
		out.Close()
		return err
	}
	defer resp.Body.Close()
	counter := &WriteCounter{}
	if _, err = io.Copy(out, io.TeeReader(resp.Body, counter)); err != nil {
		out.Close()
		return err
	}
	fmt.Print("\n")
	out.Close()
	if err = os.Rename(filepath+".tmp", filepath); err != nil {
		return err
	}
	return nil
}
func getTranspiler(transpilerList []string) string {
	if len(transpilerList) == 0 {
		return ""
	}
	return transpilerList[0]
}
func getCompiler(compilerList []string) string {
	if len(compilerList) == 0 {
		return ""
	}
	return compilerList[0]
}
func ulleungPath() string {
	//return "C:/Ulleung" // debug
	path, err := os.Executable()
	if err != nil {
		panic(err)
	}
	return filepath.Dir(path) + "/.."
}
func runPath() string {
	path, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return path
}
func getFilesUnderPath(path string) []string {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		panic (err)
	}
	var returnArray []string
	for _, f := range files {
		returnArray = append(returnArray, f.Name())
	}
	return returnArray
}
func getAbsolutePath(path string) string {
	if isAbsolutePath(path) {
		return path
	}
	return runPath() + "/" + path
}
func isAbsolutePath(path string) bool {
	if path[0] == '/' {
		return true
	} else if path[1] == ':' {
		return true
	}
	return false
}
func reverse(list []string) []string {
	var returnList []string
	for  i := 0; i < len(list); i++ {
		returnList = append(returnList, list[len(list)-i-1])
	}
	return returnList
}
func readDok(fileName string) string {
	file, err := os.Open(fileName)

	if err != nil {
		fmt.Println("Dok 경고: doklist 파일이 존재하지 않습니다.")
		return ""
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	var returnString string = ""
	for scanner.Scan() {
		returnString = returnString + scanner.Text()
	}
	return returnString
}
func getHttp(link string) string {
	resp, err := http.Get(link)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic (err)
	}
	return string(body)
}
func printHelp(WRAPPER_VERSION string) {
	fmt.Println("울릉 래퍼 버전 "+WRAPPER_VERSION)
	fmt.Println("ulleungw -v")
	fmt.Println("- 현재 울릉 래퍼의 버전 및 설치된 트랜스파일러 / 컴파일러 목록 출력")
	fmt.Println("ulleungw -get-list")
	fmt.Println("- 가장 최근의 울릉 트랜스파일러 목록 출력")
	fmt.Println("ulleungw -get-stable")
	fmt.Println("- 가장 최근의 안정 버전 트랜스파일러 설치")
	fmt.Println("ulleungw -get-latest")
	fmt.Println("- 가장 최근의 울릉 트랜스파일러 설치")
	fmt.Println("ulleungw --get-specify [버전 ID]")
	fmt.Println("- 해당 버전 ID의 울릉 트랜스파일러 설치")
	fmt.Println("uleungw 파일 이름.uln [--use-version 버전 이름] [-t | -c] [-no-doklist] [--dok 독 파일 이름.dok]")
	fmt.Println("- 파일 이름.uln 컴파일")
	fmt.Println(" --use-version : 버전 이름 지정하여 실행, 기본값: 가장 최신 안정 버전, 생략 가능")
	fmt.Println(" -t : 트랜스파일러 사용 -c : 컴파일러 사용, 기본값: 트랜스파일러 사용, 생략 가능")
	fmt.Println(" -no-doklist : doklist 파일을 사용하지 않음 명시")
	fmt.Println(" --dok 독 파일 이름.dok : 해당 폴더에 있는 dok 파일 사용")
}