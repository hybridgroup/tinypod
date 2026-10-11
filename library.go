package tinypod

import (
	"io"
	"slices"
	"strings"
)

// album is a folder of songs. Songs are full paths.
type album struct {
	name  string
	songs []string
}

// library finds the albums and songs in a Storage and opens them for the deck.
type library struct {
	st     Storage
	albums []album
	songs  []string
}

func newLibrary(st Storage) (*library, error) {
	lib := &library{st: st}
	if err := lib.scan(); err != nil {
		return nil, err
	}
	return lib, nil
}

// scan finds the WAV files in /Music, or in the root if there is no /Music.
// Each folder becomes an album, with songs from one level of subfolders too.
func (lib *library) scan() error {
	lib.albums, lib.songs = nil, nil
	root := "/Music"
	songs, dirs, err := lib.st.List(root)
	if err != nil {
		root = ""
		songs, dirs, err = lib.st.List(root)
		if err != nil {
			return err
		}
	}
	lib.songs = songs
	for _, d := range dirs {
		a := album{name: d}
		a.songs, err = lib.albumSongs(root + "/" + d)
		if err != nil {
			return err
		}
		if len(a.songs) > 0 {
			lib.albums = append(lib.albums, a)
			lib.songs = append(lib.songs, a.songs...)
		}
	}
	slices.SortFunc(lib.songs, func(a, b string) int {
		return strings.Compare(strings.ToLower(title(a)), strings.ToLower(title(b)))
	})
	return nil
}

func (lib *library) albumSongs(dir string) ([]string, error) {
	songs, subdirs, err := lib.st.List(dir)
	if err != nil {
		return nil, err
	}
	for _, s := range subdirs {
		more, _, err := lib.st.List(dir + "/" + s)
		if err != nil {
			return nil, err
		}
		songs = append(songs, more...)
	}
	return songs, nil
}

func (lib *library) open(song string) (io.ReadCloser, error) {
	return lib.st.Open(song)
}

func (lib *library) changed() bool {
	return lib.st.Changed()
}

// reload reads the songs again after the user copied files over USB. Stop the
// deck first.
func (lib *library) reload() {
	if err := lib.st.Reload(); err != nil {
		println("reload:", err.Error())
	}
	if err := lib.scan(); err != nil {
		println("scan:", err.Error())
		lib.albums, lib.songs = nil, nil
	}
}

func (lib *library) usb() bool {
	return lib.st.USB()
}

// title is the file name of a song path without the extension.
func title(path string) string {
	path = path[strings.LastIndexByte(path, '/')+1:]
	return path[:len(path)-len(".wav")]
}

// albumOf is the folder that holds a song.
func albumOf(path string) string {
	path = path[:max(strings.LastIndexByte(path, '/'), 0)]
	return path[strings.LastIndexByte(path, '/')+1:]
}
