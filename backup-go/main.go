package main

import (
	"github.com/jlaffaye/ftp"
	"io"
	"time"
	"flag"
	"log"
	"os"
	"io/ioutil"
	"archive/zip"
	"path/filepath"
	"encoding/json"
)

type Configuration struct {
	Server_IP string
	Username string
	Password string
}

func getConfig() (*Configuration, error) {
	configuration := Configuration{}
	filename := flag.String("config", "config.json", "location of the config file.")
	flag.Parse()
	data, err := ioutil.ReadFile(*filename)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(data, &configuration)
	if err != nil {
		return nil, err
	}
	return &configuration, nil
}

func createZip(path string) {
	file, err := os.Create(path + ".zip")
	check(err)
	defer file.Close()

	w := zip.NewWriter(file)
	defer w.Close()

	walker := func(path string, info os.FileInfo, err error) error {
		log.Printf("Zipping %#v\n", path)
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		f, err := w.Create(path)
		if err != nil {
			return err
		}

		_, err = io.Copy(f, file)
		if err != nil {
			return err
		}

		return nil
	}

	err = filepath.Walk(path, walker)
}

func deleteFolder(path string) error{
	err := os.Remove(path)
	if err != nil {
		return err
	}
	return nil
}

func main() {
	pathPrefix := "mc_backup-" + time.Now().Format("02-01-2006") + "/"
	config, err := getConfig()
	conn, err := ftp.Dial(config.Server_IP, ftp.DialWithTimeout(5*time.Second))
	check(err)

	if err = conn.Login(config.Username, config.Password); err != nil {
		check(err)
	}

	log.Println("[DEBUG] Connected to the FTP server.")

	os.Mkdir(pathPrefix, 0755)
	walker := conn.Walk("/")

	for walker.Next() {
		if err := walker.Err(); err != nil {
			check(err)
		}
		path := walker.Path()
		path = path[1:]

		if walker.Stat().Type.String() == "folder" {
			log.Printf("[DEBUG] Creating folder: %s\n", path)
			os.MkdirAll(pathPrefix + path, 0700)
		} else {
			r, err := conn.Retr(path)
			check(err)

			b, err := ioutil.ReadAll(r)
			check(err)

			log.Printf("[DEBUG] Downloading file %s\n", path)
			err = ioutil.WriteFile(pathPrefix + path, b, 0755)
			check(err)
			r.Close()
		}
	}

	if err := conn.Quit(); err != nil {
		check(err)
	}

	log.Println("[DONE] Finished downloading files.")
	log.Printf("[DEBUG] Creating a zip for %s", pathPrefix)

	relativePath := pathPrefix[:len(pathPrefix) - 1]
	createZip(relativePath)
	err = deleteFolder(relativePath)
	check(err)
}

func check(err error) {
	if err != nil {
		log.Printf("[ERROR] %v", err)
	}
}
