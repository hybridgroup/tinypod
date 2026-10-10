//go:build esp32c3 || rp2350

package main

import (
	"slices"
	"strings"

	"github.com/soypat/fat"
)

// album is a folder of songs. Songs are full paths.
type album struct {
	name  string
	songs []string
}

type library struct {
	albums []album
	songs  []string
}

// scan finds the WAV files in /Music, or in the root if there is no /Music.
// Each folder becomes an album, with songs from one level of subfolders too.
func scan(fs *fat.FS) (*library, error) {
	root := "/Music"
	var info fat.FileInfo
	if fs.Stat(root, &info) != nil || !info.IsDir() {
		root = ""
	}
	lib := &library{}
	songs, dirs, err := list(fs, root)
	if err != nil {
		return nil, err
	}
	lib.songs = songs
	for _, d := range dirs {
		a := album{name: d}
		a.songs, err = albumSongs(fs, root+"/"+d)
		if err != nil {
			return nil, err
		}
		if len(a.songs) > 0 {
			lib.albums = append(lib.albums, a)
			lib.songs = append(lib.songs, a.songs...)
		}
	}
	slices.SortFunc(lib.songs, func(a, b string) int {
		return strings.Compare(strings.ToLower(title(a)), strings.ToLower(title(b)))
	})
	return lib, nil
}

func albumSongs(fs *fat.FS, dir string) ([]string, error) {
	songs, subdirs, err := list(fs, dir)
	if err != nil {
		return nil, err
	}
	for _, s := range subdirs {
		more, _, err := list(fs, dir+"/"+s)
		if err != nil {
			return nil, err
		}
		songs = append(songs, more...)
	}
	return songs, nil
}

// list returns the sorted WAV paths and folder names in dir.
func list(fs *fat.FS, dir string) (songs, dirs []string, err error) {
	var d fat.Dir
	path := dir
	if path == "" {
		path = "/"
	}
	if err := fs.OpenDir(&d, path); err != nil {
		return nil, nil, err
	}
	defer d.Close()
	err = d.ForEachFile(func(fi *fat.FileInfo) error {
		name := fi.Name()
		switch {
		case strings.HasPrefix(name, "."):
		case fi.IsDir():
			dirs = append(dirs, name)
		case strings.HasSuffix(strings.ToLower(name), ".wav"):
			songs = append(songs, dir+"/"+name)
		}
		return nil
	})
	slices.Sort(songs)
	slices.Sort(dirs)
	return songs, dirs, err
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
