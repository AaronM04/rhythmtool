RhythmTool
==========

Manipulate a Rhythmbox playlist file. Either shuffle or sort the tracks in a
playlist based on the folder they're in, and also either shuffle or short the
folders.

# Install

```
go get -u github.com/aaronm04/rhythmtool
```

# Usage

Rhythmtool always reads from an XML file at a path relative to your home
directory: `.local/share/rhythmbox/playlists.xml`. By default it doesn't change
any files. Specify a `-out` file path to write a new playlist XML file
containing the newly shuffled playlist as well as your pre-existing playlists.

```
$ rhythmtool -h
Usage of rhythmtool:
  -display
    	whether to display info on static playlists
  -displayAll
    	whether to display the song file paths as well
  -out string
    	the file path to write the processed XML to
  -shuffleDirs
    	whether to shuffle all dirs in playlists (default true)
  -shuffleInDir
    	whether to shuffle the songs in one directory (default true)
```

# License

Released into the public domain.
