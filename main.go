package main

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"github.com/codegangsta/cli"
	"io"
	"log"
	"os"
	fpath "path/filepath"
	"regexp"
)

func main() {
	app := cli.NewApp()
	app.Name = "MTA Community CLI"
	app.Version = "1.0.0"
	app.Author = "Qais \"qaisjp\" Patankar"
	app.Email = "qaisjp@gmail.com"

	app.Commands = []cli.Command{
		{
			Name:  "check",
			Usage: "check whether the resource is valid for upload",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "file",
					Usage: "file to check",
				},
			},
			Action: checkResource,
		},
	}

	app.Run(os.Args)
}

var logger *log.Logger

func checkResource(c *cli.Context) {
	logger = log.New(os.Stdout, "error: ", 0)
	filepath := c.String("file")
	success := true

	if filepath == "" {
		logger.Fatal("No file passed")
	}

	filepath = fpath.Clean(filepath)

	fileInfo, err := os.Stat(filepath)
	if err != nil {
		logger.Fatal("File does not exist")
	}

	if fileInfo.IsDir() {
		logger.Fatal("Expected file, got directory")
	}

	reader, err := zip.OpenReader(filepath)
	if err != nil {
		logger.Fatal("not a valid zip file")
	}
	defer reader.Close()

	// Do they have a meta file?
	hasMeta := false
	var meta XMLMeta

	for _, file := range reader.File {
		fileInfo := file.FileInfo()
		filename := fileInfo.Name()
		if (filename == "meta.xml") && !fileInfo.IsDir() {
			hasMeta = true

			metaReader, err := file.Open()
			if err != nil {
				logger.Println("could not open meta.xml file")
				success = false
				continue
			}

			var result bool
			result, meta = checkMeta(metaReader)
			if !result {
				success = false
			}

			metaReader.Close()
			continue
		}

		ext := fpath.Ext(filename)
		if (ext == ".exe") || (ext == ".com") || (ext == ".bat") {
			logger.Printf("contains blocked file %s\n", filename)
			success = false
		}
	}

	if !hasMeta {
		logger.Println("missing meta.xml file")
		return
	}

	if success {
		fmt.Printf(`ok {version: "%s", type: "%s", name: "%s"}`, meta.Infos[0].Version, meta.Infos[0].Type, meta.Infos[0].Name)
	}
}

type XMLInfo struct {
	Name        string `xml:"name,attr"`
	Version     string `xml:"version,attr"`
	Description string `xml:"description,attr"`
	Type        string `xml:"type,attr"`
}

type XMLMeta struct {
	Infos []XMLInfo `xml:"info"`
}

func checkMeta(file io.ReadCloser) (success bool, meta XMLMeta) {

	// TODO - THIS IS NOT "STRICT" ENOUGH
	// TRY PREPENDING THE XML FILE WITH "jasjdsd"
	// GOLANG WILL BE ABLE TO READ THE SYNTACTICALLY INCORRECT FILE
	// MTA:SA CANNOT
	decoder := xml.NewDecoder(file)
	err := decoder.Decode(&meta)
	if err != nil {
		logger.Println("could not decode meta.xml")
		return
	}

	if len(meta.Infos) != 1 {
		logger.Println("meta.xml should have exactly 1 info field")
		return
	}

	info := meta.Infos[0]
	if info.Type == "" {
		logger.Println("meta.xml is missing the 'type' field")
		return
	} else if (info.Type != "gamemode") && (info.Type != "map") && (info.Type != "script") && (info.Type != "misc") {
		logger.Println("meta.xml has an invalid 'type' field")
		return
	}

	if info.Version == "" {
		logger.Println("meta.xml is missing the version field for <info>")
		return
	} else if versionSuccess, _ := regexp.MatchString(`^(\d\.\d\.\d|\d\.\d|\d)$`, info.Version); !versionSuccess {
		logger.Println("meta.xml contains a malformed version field (should be in the form X, X.X, or X.X.X)")
		return
	}

	success = true
	return
}
