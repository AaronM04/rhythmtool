package main

import (
	"encoding/xml"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const playlistsPath = "/home/aaron/.local/share/rhythmbox/playlists.xml"

type RhythmDBPlaylists struct {
	XMLName   xml.Name
	Playlists []Playlist `xml:"playlist"`
}

type Location string

func (l Location) Text() string {
	escapedStr, err := url.QueryUnescape(string(l))
	if err != nil {
		log.Fatal("unescape", string(l), err)
	}
	// strip file:// from beginning
	if escapedStr[:7] == "file://" {
		escapedStr = escapedStr[7:]
	}
	return escapedStr
}

func (l Location) Split() (string, string) {
	text := l.Text()
	return filepath.Split(text)
}

type Playlist struct {
	Locations     []Location   `xml:"location"`
	Name          string       `xml:"name,attr"`
	ShowBrowser   bool         `xml:"show-browser,attr"`
	BrowserPos    int          `xml:"browser-position,attr"`
	SearchType    string       `xml:"search-type,attr"`
	Type          string       `xml:"type,attr"`
	SortKey       string       `xml:"sort-key,attr,omitempty"`
	SortDirection *int         `xml:"sort-direction,attr,omitempty"`
	Conjunction   *Conjunction `xml:"conjunction"`
}

type Conjunction struct {
	Data string `xml:",innerxml"`
}

var (
	out          = flag.String("out", "", "the file path to write the processed XML to")
	shuffleDirs  = flag.Bool("shuffleDirs", true, "whether to shuffle all dirs in playlists")
	shuffleInDir = flag.Bool("shuffleInDir", true, "whether to shuffle the songs in one directory")
	doDisplay    = flag.Bool("display", false, "whether to display info on static playlists")
	doDisplayAll = flag.Bool("displayAll", false, "whether to display the song file paths as well")
)

func main() {
	flag.Parse()

	// -displayAll=true implies -display=true
	if *doDisplayAll {
		*doDisplay = true
	}

	inFile, err := os.Open(playlistsPath)
	if err != nil {
		log.Fatal("open", playlistsPath, err)
	}
	defer inFile.Close()

	decoder := xml.NewDecoder(inFile)
	doc := &RhythmDBPlaylists{}
	err = decoder.Decode(doc)
	if err != nil {
		log.Fatal("decode", err)
	}

	for _, p := range doc.Playlists {
		if p.Type != "static" {
			continue
		}

		if *doDisplay {
			fmt.Println("===")
			display(&p)
		}

		shuffle(p.Locations, *shuffleDirs, *shuffleInDir)
	}

	// output the XML
	if *out != "" {
		outFile, err := os.Create(*out)
		if err != nil {
			log.Fatal("out file", err)
		}
		defer outFile.Close()

		// write header in case that matters
		_, err = outFile.WriteString(`<?xml version="1.0"?>` + "\n")
		if err != nil {
			log.Fatal("out file: write header", err)
		}

		encoder := xml.NewEncoder(outFile)
		encoder.Indent("", "  ")
		err = encoder.Encode(doc)
		if err != nil {
			log.Fatal("encode", err)
		}
	}
}

func display(p *Playlist) {
	fmt.Println("len(Locations):", len(p.Locations))
	if *doDisplayAll {
		for _, l := range p.Locations {
			fmt.Println("  ", l.Text())
		}
	}
	fmt.Println("Name:", p.Name)
	fmt.Println("ShowBrowser:", p.ShowBrowser)
	fmt.Println("BrowserPos:", p.BrowserPos)
	fmt.Println("SearchType:", p.SearchType)
	fmt.Println("Type:", p.Type)
	fmt.Println("SortKey:", p.SortKey)
	fmt.Println("SortDirection:", *p.SortDirection)
	if p.Conjunction != nil {
		fmt.Println("Conjunction:", *p.Conjunction)
	}
}

// shuffle changes the order of the elements in the supplied slice of Locations according to two boolean options.
// shuffleDirs - shuffle the directories; if false, they are sorted.
// shuffleInDir - shuffle within one directory; if false, they are sorted.
func shuffle(locations []Location, shuffleDirs, shuffleInDir bool) {
	all := make(map[string][]Location) // map of dir => contents of dir
	var dirs []string                  // all dirs - determines order they get written out in
	for _, l := range locations {
		dir, _ := l.Split()
		dirs = append(dirs, dir)
		all[dir] = append(all[dir], l)
	}

	// shuffle according to options
	if shuffleDirs {
		rand.Shuffle(len(dirs), func(i, j int) {
			dirs[i], dirs[j] = dirs[j], dirs[i]
		})
	} else {
		sort.Strings(dirs)
	}
	if shuffleInDir {
		// shuffle
		for _, dir := range dirs {
			rand.Shuffle(len(all[dir]), func(i, j int) {
				all[dir][i], all[dir][j] = all[dir][j], all[dir][i]
			})
		}
	} else {
		// sort
		for _, dir := range dirs {
			sort.Slice(all[dir], func(i, j int) bool {
				_, fileI := all[dir][i].Split()
				_, fileJ := all[dir][j].Split()
				return strings.Compare(fileI, fileJ) == -1
			})
		}
	}

	var result []Location
	for _, dir := range dirs {
		result = append(result, all[dir]...)
	}
	copy(locations, result)
}
