// Manipulate a Rhythmbox playlist file.
package main

import (
	cryptorand "crypto/rand"
	"encoding/xml"
	"flag"
	"fmt"
	"log"
	"math/big"
	"math/rand"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const playlistsRelPath = ".local/share/rhythmbox/playlists.xml" // relative to ${HOME}

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
	rnd          = Seeded()
)

func main() {
	flag.Parse()

	// -displayAll=true implies -display=true
	if *doDisplayAll {
		*doDisplay = true
	}

	homePath, err := os.UserHomeDir()
	if err != nil {
		log.Fatal("UserHomeDir", err)
	}
	playlistsPath := filepath.Join(homePath, playlistsRelPath)

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

	// shuffle the first static Playlist found, saving it as a new Playlist at the end
	// TODO: consider flags to 1) print list of playlist names, 2) select which playlist to create a shuffled copy of
	for _, p := range doc.Playlists {
		if p.Type != "static" {
			continue
		}

		if *doDisplay {
			fmt.Println("===")
			display(&p)
		}

		newP := p
		randNum := rnd.Int31() % (1 << 24)
		newP.Name += fmt.Sprintf("_SHUFFLED_%s_%d", time.Now().Format("2006-01-02"), randNum)
		newP.Locations = shuffle(p.Locations, *shuffleDirs, *shuffleInDir)
		doc.Playlists = append(doc.Playlists, newP)
		break
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

// shuffle returns a shuffled copy of the supplied slice of Locations according to two boolean options.
// shuffleDirs - shuffle the directories; if false, they are sorted.
// shuffleInDir - shuffle within one directory; if false, they are sorted.
func shuffle(locations []Location, shuffleDirs, shuffleInDir bool) []Location {
	all := make(map[string][]Location) // map of dir => contents of dir
	var dirs []string                  // all dirs - determines order they get written out in
	for _, l := range locations {
		dir, _ := l.Split()
		if _, ok := all[dir]; !ok {
			dirs = append(dirs, dir)
		}
		all[dir] = append(all[dir], l)
	}

	// shuffle according to options
	if shuffleDirs {
		rnd.Shuffle(len(dirs), func(i, j int) {
			dirs[i], dirs[j] = dirs[j], dirs[i]
		})
	} else {
		sort.Strings(dirs)
	}
	if shuffleInDir {
		// shuffle
		for _, dir := range dirs {
			thisDir := all[dir]
			rnd.Shuffle(len(thisDir), func(i, j int) {
				thisDir[i], thisDir[j] = thisDir[j], thisDir[i]
			})
		}
	} else {
		// sort
		for _, dir := range dirs {
			thisDir := all[dir]
			sort.Slice(thisDir, func(i, j int) bool {
				_, fileI := thisDir[i].Split()
				_, fileJ := thisDir[j].Split()
				return strings.Compare(fileI, fileJ) == -1
			})
		}
	}

	var result []Location
	for _, dir := range dirs {
		result = append(result, all[dir]...)
	}
	if len(result) != len(locations) {
		log.Fatal("len(result) != len(locations)", len(result), len(locations))
	}
	return result
}

func Seeded() *rand.Rand {
	nBig, err := cryptorand.Int(cryptorand.Reader, big.NewInt(1<<62))
	if err != nil {
		log.Fatal("Int", err)
	}
	if !nBig.IsInt64() {
		log.Fatal("not an int64")
	}
	n := nBig.Int64()
	return rand.New(rand.NewSource(n))
}
