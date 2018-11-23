package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Cache should have a comment
type Cache struct {
	Commands map[string]*CachedFileByCommand `json:"commands"`
}

// CachedFileByCommand should have a comment
type CachedFileByCommand struct {
	Files map[string]*CachedFile `json:"files"`
}

// CachedFile should have a comment
type CachedFile struct {
	Output   string    `json:"output"`
	Modified time.Time `json:"modified"`
	Hash     string    `json:"hash"`
}

// State should have a comment
type State struct {
	Changed   int
	Unchanged int
	New       int
}

// ProgramConfig should have a comment
type ProgramConfig struct {
	Command []string
	Cache   *Cache
	State   *State
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

// CacheFilePath should have a comment
const CacheFilePath = "pc-cache.json"

func loadCache() *Cache {
	var err error
	cache := Cache{}

	cacheFile, err := os.Open(CacheFilePath)
	if err != nil {
		if !strings.HasSuffix(err.Error(), "no such file or directory") {
			panic(err)
		}
		cacheFile, err = os.Create(CacheFilePath)
		checkErr(err)
		var data []byte
		data, err = json.Marshal(cache)
		checkErr(err)
		_, err = cacheFile.Write(data)
		checkErr(err)
		cacheFile, err = os.Open(CacheFilePath)
		checkErr(err)
	}

	checkErr(err)

	contents, err := ioutil.ReadAll(cacheFile)
	checkErr(err)

	err = json.Unmarshal(contents, &cache)
	checkErr(err)

	return &cache
}

func saveCache(cache *Cache) {
	cacheFile, err := os.Create(CacheFilePath)
	checkErr(err)
	data, err := json.Marshal(cache)
	checkErr(err)
	_, err = cacheFile.Write(data)
	checkErr(err)
}

func runCmd(config ProgramConfig, filename string) {
	cacheKey := strings.Join(config.Command, " ")
	command := config.Command
	command = append(command, filename)
	fileStats, err := os.Stat(filename)
	if err != nil {
		fmt.Println(err)
		return
	}
	cmd := exec.Command(command[0], command[1:]...)

	cmdOut, err := cmd.Output()
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(fmt.Sprintf("> %s", strings.Join(command, " ")))
	fmt.Println(string(cmdOut))

	if config.Cache.Commands == nil {
		config.Cache.Commands = map[string]*CachedFileByCommand{}
	}
	if config.Cache.Commands[cacheKey] == nil {
		config.Cache.Commands[cacheKey] = &CachedFileByCommand{}
	}
	if config.Cache.Commands[cacheKey].Files == nil {
		config.Cache.Commands[cacheKey].Files = map[string]*CachedFile{}
	}
	config.Cache.Commands[cacheKey].Files[filename] = &CachedFile{
		Output:   string(cmdOut),
		Modified: fileStats.ModTime(),
		Hash:     string(fileStats.Size()),
	}
}

func main() {
	config := ProgramConfig{
		Command: os.Args[1:],
		Cache:   loadCache(),
	}

	scanner := bufio.NewScanner(os.Stdin)

	for scanner.Scan() {
		filename := scanner.Text()

		runCmd(config, filename)
	}

	saveCache(config.Cache)

	if err := scanner.Err(); err != nil {
		_, err = fmt.Fprintln(os.Stderr, "error:", err)
		if err != nil {
			panic(err)
		}
		os.Exit(1)
	}
}
