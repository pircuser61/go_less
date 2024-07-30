package config

import "runtime"

func GetTmpDir() string {
	if runtime.GOOS == "windows" {
		return "D:/tmp/" //
	}
	return "/tmp/"
}

func GetFileIn() string {
	return GetTmpDir() + "indata.txt"
}

func GetFileOut() string {
	return GetTmpDir() + "outdata.txt"
}
