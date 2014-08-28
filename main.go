package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/shurcooL/go-goon"
)

type DirWatcher struct {
	Dirpath         string
	UpdateDir       chan string
	Errchan         chan error
	RequestStop     chan bool
	Done            chan bool
	InitialReadDone chan bool
}

func NewDirWatcher(dirpath string) *DirWatcher {
	return &DirWatcher{
		Dirpath:         dirpath,
		UpdateDir:       make(chan string),
		Errchan:         make(chan error),
		RequestStop:     make(chan bool),
		Done:            make(chan bool),
		InitialReadDone: make(chan bool),
	}
}

func (w *DirWatcher) Start() {
	go func() {
		dirstat, err := os.Stat(w.Dirpath)
		if err != nil {
			w.Errchan <- fmt.Errorf("could not os.Stat('%s'): error is '%s", w.Dirpath, err)
			return
		}
		origfiles, err := ioutil.ReadDir(w.Dirpath)
		if err != nil {
			w.Errchan <- fmt.Errorf("could not ioutil.ReadDir('%s'): error is '%s'", w.Dirpath, err)
		}
		close(w.InitialReadDone)

		for {
			select {
			case <-w.RequestStop:
				close(w.Done)
				return
			default:
				stat, err := os.Stat(w.Dirpath)
				if err != nil {
					w.Errchan <- err
					return
				}

				fmt.Printf("stat = %#v  for w.Dirpath='%s'\n", stat, w.Dirpath)

				if stat.Size() != dirstat.Size() || stat.ModTime() != dirstat.ModTime() {
					dirstat = stat
					fmt.Printf("noticed difference from original dirstat[%#v] vs stat[%#v]\n", dirstat, stat)
					w.UpdateDir <- w.Dirpath
				} else {
					newfiles, err := ioutil.ReadDir(w.Dirpath)
					if err != nil {
						w.Errchan <- fmt.Errorf("could not ioutil.ReadDir('%s'): error is '%s'", w.Dirpath, err)
						continue
					}
					if len(newfiles) != len(origfiles) {
						origfiles = newfiles
						w.UpdateDir <- w.Dirpath
						continue
					}
					for i := range origfiles {
						if origfiles[i].Name() != newfiles[i].Name() {
							origfiles = newfiles
							w.UpdateDir <- w.Dirpath
							continue
						}
						if origfiles[i].ModTime() != newfiles[i].ModTime() {
							origfiles = newfiles
							w.UpdateDir <- w.Dirpath
							continue
						}
						if origfiles[i].Size() != newfiles[i].Size() {
							origfiles = newfiles
							w.UpdateDir <- w.Dirpath
							continue
						}
					}
					fmt.Printf("No differences between origfiles[%#v] and newfiles[%#v] found\n", origfiles, newfiles)
					fmt.Printf("origfiles:\n")
					goon.Dump(origfiles)
					fmt.Printf("newfiles:\n")
					goon.Dump(newfiles)
				}

				time.Sleep(1 * time.Second)
			}
		}
	}()
}

func main() {
	g := NewGavelin("gavelin")
	g.Start()
	chStop := make(chan os.Signal, 2)
	signal.Notify(chStop, os.Interrupt, syscall.SIGTERM)

	select {
	case <-chStop:
		g.Stop()
	}
}

type Gavelin struct {
	RootPath      string
	RootDirHandle *os.File
	RequestStop   chan bool
	Done          chan bool

	Watcher *DirWatcher

	PngCount    int
	SubDirCount int
}

func NewGavelin(path string) *Gavelin {
	if !DirExists(path) {
		err := os.MkdirAll(path, 0755)
		if err != nil {
			panic(fmt.Errorf("root file path not found: '%s' and could not be created: '%s'", path, err))
		}
	}
	rootdirhandle, err := os.Open(path)
	if err != nil {
		panic(fmt.Errorf("root file path '%s' could not be opened, error: '%s'", path, err))
	}

	return &Gavelin{
		RootPath:      path,
		RootDirHandle: rootdirhandle,
		RequestStop:   make(chan bool),
		Done:          make(chan bool),
		Watcher:       NewDirWatcher(path),
	}
}

func (g *Gavelin) Start() {
	g.Watcher.Start()
	<-g.Watcher.InitialReadDone
	go func() {
		for {
			select {
			case err := <-g.Watcher.Errchan:
				fmt.Printf("got err on g.Errchan from watcher: '%s'\n", err)

			case udir := <-g.Watcher.UpdateDir:
				fmt.Printf("UpdateDir signaled with: '%s'\n", udir)
				g.Update()

			case <-g.RequestStop:
				g.Watcher.RequestStop <- true
				<-g.Watcher.Done
				g.RootDirHandle.Close()
				close(g.Done)
				return
			}
		}
	}()
}

func (g *Gavelin) Stop() {
	g.RequestStop <- true
	<-g.Done
}

func (g *Gavelin) DisplayedPngCount() int {
	g.Update()
	return g.PngCount
}

type byName []os.FileInfo

func (f byName) Len() int           { return len(f) }
func (f byName) Less(i, j int) bool { return f[i].Name() < f[j].Name() }
func (f byName) Swap(i, j int)      { f[i], f[j] = f[j], f[i] }

func (g *Gavelin) Update() {
	fmt.Printf("Update called.\n")
	flist, err := g.RootDirHandle.Readdir(-1)
	if err != nil {
		panic(fmt.Errorf("call to Readdir() on root file path '%s' failed: '%s'", g.RootPath, err))
	}
	g.PngCount = 0
	g.SubDirCount = 0
	fmt.Printf("size of flist: %d\n", len(flist))

	sort.Sort(byName(flist))

	for i := range flist {
		fmt.Printf("Update on rootdir '%s' sees dir entry '%s'\n", g.RootPath, flist[i].Name())
		if !flist[i].IsDir() {
			if strings.HasSuffix(flist[i].Name(), ".png") {
				g.PngCount++
			}
		} else {
			g.SubDirCount++
		}
	}
}

func (g *Gavelin) DirCount() int {
	g.Update()
	return g.SubDirCount
}

func GenerateNewPng(path string) {
	dest, err := os.Create(path)
	if err != nil {
		panic(fmt.Errorf("could not open destination png file '%s', error: '%s'", path, err))
	}
	defer dest.Close()

	srcpath := "testdata/hist.png"
	src, err := os.Open(srcpath)
	if err != nil {
		panic(fmt.Errorf("could not open source test-ping file '%s', error: '%s'", srcpath, err))
	}
	defer src.Close()
	_, err = io.Copy(dest, src)
	if err != nil {
		panic(fmt.Errorf("could not copy test-ping file '%s' to '%s', error: '%s'", srcpath, path, err))
	}
}

func GenerateNewSubdir(path string) {
	err := os.MkdirAll(path, 0755)
	if err != nil {
		panic(fmt.Errorf("could not create'%s', error: '%s'", path, err))
	}
}
