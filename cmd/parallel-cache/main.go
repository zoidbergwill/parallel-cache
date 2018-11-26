package main

import (
	"bufio"
	"crypto/md5" // #nosec G501
	"encoding/hex"
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
	LastSeen time.Time `json:"last_seen"`
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
	file, err := os.Open(filename) // #nosec G304
	if err != nil {
		fmt.Println(err)
		return
	}
	fileContents, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Println(err)
		return
	}
	hasher := md5.New() // #nosec G401
	_, err = hasher.Write(fileContents)
	if err != nil {
		fmt.Println(err)
		return
	}
	fileHash := hex.EncodeToString(hasher.Sum(nil))
	fileStats, err := os.Stat(filename)
	if err != nil {
		fmt.Println(err)
		return
	}
	modified := fileStats.ModTime()
	if config.Cache.Commands == nil {
		config.Cache.Commands = map[string]*CachedFileByCommand{}
	}
	if config.Cache.Commands[cacheKey] == nil {
		config.Cache.Commands[cacheKey] = &CachedFileByCommand{}
	}
	new := false
	if config.Cache.Commands[cacheKey].Files == nil {
		new = true
		config.Cache.Commands[cacheKey].Files = map[string]*CachedFile{}
	}

	cacheValue := config.Cache.Commands[cacheKey].Files[filename]
	if cacheValue.Hash == fileHash && cacheValue.Modified == modified {
		config.State.Unchanged++
		config.Cache.Commands[cacheKey].Files[filename].LastSeen = time.Now()
	} else {
		if new {
			config.State.New++
		} else {
			config.State.Changed++
		}
		command := config.Command
		command = append(command, filename)
		cmd := exec.Command(command[0], command[1:]...) // #nosec G204
		cmdOut, err := cmd.Output()
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println(fmt.Sprintf("> %s", strings.Join(command, " ")))
		fmt.Println(string(cmdOut))

		config.Cache.Commands[cacheKey].Files[filename] = &CachedFile{
			Output:   string(cmdOut),
			LastSeen: time.Now(),
			Modified: modified,
			Hash:     fileHash,
		}
	}
}

func main() {
	config := ProgramConfig{
		Command: os.Args[1:],
		Cache:   loadCache(),
		State:   &State{},
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
	total := config.State.New + config.State.Changed + config.State.Unchanged
	fmt.Printf("Found %d files. New: %d. Changed: %d. Unchanged: %d.\n", total, config.State.New, config.State.Changed, config.State.Unchanged)
}
