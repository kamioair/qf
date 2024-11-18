package qio

import (
	"errors"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

// PathExists
//
//	@Description: 判断文件或文件夹是否存在
//	@param path 路径
//	@return bool
func PathExists(path string) bool {
	var exist = true
	if _, err := os.Stat(path); os.IsNotExist(err) {
		exist = false
	}
	return exist
}

// GetCurrentFilePath
//
//	@Description: 获取当前文件路径
//	@return string
//	@return error
func GetCurrentFilePath() (string, error) {
	if runtime.GOOS == "windows" {
		file, err := exec.LookPath(os.Args[0])
		if err != nil {
			return "", err
		}
		return filepath.Abs(file)
	}
	return filepath.Abs(os.Args[0])
}

// GetCurrentDirectory
//
//	@Description: 获取程序的当前工作目录
//	@return string
func GetCurrentDirectory() string {
	path, err := GetCurrentFilePath()
	if err != nil {
		return ""
	}
	return GetDirectory(path)
}

// GetCurrentRoot
//
//	@Description: 获取程序的当前的根目录
//	@return string
func GetCurrentRoot() string {
	dir := GetCurrentDirectory()
	if dir == "" {
		return ""
	}
	dir = strings.Replace(dir, "\\", "/", -1)
	sp := strings.Split(dir, "/")
	if len(sp) > 0 {
		if runtime.GOOS == "windows" {
			return sp[0]
		}
	}
	return "/"
}

// GetDirectory
//
//	@Description: 获取路径下面的目录
//	@param path 文件路径
//	@return string 目录
func GetDirectory(path string) string {
	return filepath.Dir(path)
}

// GetFiles
//
//	@Description: 获取指定目录下面的所有文件
//	@param path
//	@return []string
//	@return error
func GetFiles(path string) ([]string, error) {
	return GetFilesByPattern(path, "*")
}

// GetFolders
//
//	@Description: 获取指定目录下面的所有文件夹
//	@param path
//	@return []string
//	@return error
func GetFolders(path string) ([]string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	dirs := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			dirs = append(dirs, entry.Name())
		}
	}
	return dirs, nil
}

// GetFilesByPattern
//
//	@Description: 获取指定目录下面的所有文件
//	@param path
//	@param searchPattern
//	@return []string
//	@return error
func GetFilesByPattern(path string, searchPattern string) ([]string, error) {
	var files []string

	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		files = append(files, path)

		return nil
	})

	if err != nil {
		return nil, err
	}

	return files, nil
}

// GetFullPath
//
//	@Description: 获取完整路径
//	@param path 相对路径
//	@return string
func GetFullPath(path string) string {
	// 将\\转为/
	full, _ := filepath.Abs(formatPath(path))
	return full
}

// GetFileName
//
//	@Description: 获取文件名
//	@param path
//	@return string
func GetFileName(path string) string {
	return filepath.Base(formatPath(path))
}

// GetFileExt
//
//	@Description: 获取文件的后缀名
//	@param path
//	@return string
func GetFileExt(path string) string {
	return filepath.Ext(formatPath(path))
}

// GetFileNameWithoutExt
//
//	@Description: 获取没有后缀名的文件名
//	@param path
//	@return string
func GetFileNameWithoutExt(path string) string {
	fileName := filepath.Base(formatPath(path))
	fileExt := filepath.Ext(fileName)
	return fileName[0 : len(fileName)-len(fileExt)]
}

// CreateDirectory
//
//	@Description: 创建文件夹
//	@param path 文件路径或文件夹路径
//	@return string 文件夹路径
//	@return error 是否成功
func CreateDirectory(path string) (string, error) {
	path = GetFullPath(path)
	if IsFile(path) {
		path = GetDirectory(path)
	}
	if PathExists(path) == false {
		err := os.MkdirAll(path, 0777)
		if err != nil {
			return path, err
		}
	}
	return path, nil
}

// CreateFile
//
//	@Description: 创建空文件
//	@param path
//	@return error
func CreateFile(path string) error {
	fullPath := GetFullPath(path)
	// 判断文件夹是否操作
	dir := GetDirectory(path)
	if PathExists(dir) == false {
		_, err := CreateDirectory(dir)
		if err != nil {
			return err
		}
	}
	// 生成文件
	file, err := os.Create(fullPath)
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {

		}
	}(file)
	return err
}

// DeleteFile
//
//	@Description: 删除文件
//	@param filename
//	@return error
func DeleteFile(filename string) error {
	return os.Remove(filename)
}

// IsDirectory
//
//	@Description: 判断所给路径是否为文件夹
//	@param path
//	@return bool
func IsDirectory(path string) bool {
	return !IsFile(path)
}

// IsFile
//
//	@Description: 判断所给路径是否为文件
//	@param path
//	@return bool
func IsFile(path string) bool {
	matched, err := regexp.MatchString("^.+\\.[a-zA-Z]+$", path)
	if err != nil {
		return false
	}
	return matched
}

// ReadAllString
//
//	@Description: 读取文件内容到字符串
//	@param filename
//	@return string
//	@return error
func ReadAllString(filename string) (string, error) {
	if !PathExists(filename) {
		return "", errors.New("file '" + filename + "' is not exist")
	}
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return string(bytes), err
}

// ReadAllBytes
//
//	@Description: 读取文件内容到数组
//	@param filename
//	@return []byte
//	@return error
func ReadAllBytes(filename string) ([]byte, error) {
	if !PathExists(filename) {
		return nil, errors.New("file '" + filename + "' is not exist")
	}
	return ioutil.ReadFile(filename)
}

// WriteAllBytes
//
//	@Description: 写入字节数组，如果文件不存在则创建
//	@param filename
//	@param bytes
//	@param isAppend
//	@return error
func WriteAllBytes(filename string, bytes []byte, isAppend bool) error {
	f, err := readyToWrite(filename, isAppend)
	if err != nil || f == nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(bytes)
	return err
}

// WriteString
//
//	@Description: 写入字符串，如果文件不存在则创建
//	@param filename
//	@param str
//	@param isAppend
//	@return error
func WriteString(filename string, str string, isAppend bool) error {
	f, err := readyToWrite(filename, isAppend)
	if err != nil || f == nil {
		return err
	}
	defer f.Close()
	_, err = io.WriteString(f, str) //写入文件(字符串)
	return err
}

func readyToWrite(filename string, isAppend bool) (f *os.File, e error) {
	filename = GetFullPath(filename)
	// 创建文件夹
	_, _ = CreateDirectory(filepath.Dir(filename))
	// 如果文件存在
	if PathExists(filename) {
		if isAppend {
			return os.OpenFile(filename, os.O_APPEND|os.O_WRONLY, os.ModePerm)
		}
		err := DeleteFile(filename)
		if err != nil {
			return nil, err
		}
	}
	return os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.ModePerm)
}

func formatPath(path string) string {
	path = filepath.Clean(path)
	path = strings.Replace(path, "\\", "/", -1)
	path = strings.Replace(path, "//", "/", -1)
	return path
}