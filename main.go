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
)

type DirWatcher struct {
	Dirpath         string
	UpdateDir       chan string
	Errchan         chan error
	RequestStop     chan bool
	Done            chan bool
	InitialReadDone chan bool
}

func NewDirWatcher(dirpath string, udc chan string) *DirWatcher {
	return &DirWatcher{
		Dirpath:         dirpath,
		UpdateDir:       udc,
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

				if stat.Size() != dirstat.Size() || stat.ModTime() != dirstat.ModTime() {
					dirstat = stat
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
	RootPath    string
	RequestStop chan bool
	Done        chan bool
	UpdateDir   chan string // Watchers send on this to tell the root displayer to update a path

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

	udc := make(chan string)
	return &Gavelin{
		RootPath:    path,
		RequestStop: make(chan bool),
		Done:        make(chan bool),
		UpdateDir:   udc,
		Watcher:     NewDirWatcher(path, udc),
	}
}

func (g *Gavelin) Start() {
	g.Watcher.Start()
	<-g.Watcher.InitialReadDone
	go func() {
		for {
			select {
			case err := <-g.Watcher.Errchan:
				fmt.Printf("got err on g.Errchan from watcher for dirpath '%s': error is '%s'\n", g.Watcher.Dirpath, err)

			case udir := <-g.Watcher.UpdateDir:
				g.Update(udir)

			case <-g.RequestStop:
				g.Watcher.RequestStop <- true
				<-g.Watcher.Done
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
	g.Update(g.RootPath)
	return g.PngCount
}

type byName []os.FileInfo

func (f byName) Len() int           { return len(f) }
func (f byName) Less(i, j int) bool { return f[i].Name() < f[j].Name() }
func (f byName) Swap(i, j int)      { f[i], f[j] = f[j], f[i] }

func (g *Gavelin) Update(path string) {

	rootdirhandle, err := os.Open(g.RootPath)
	if err != nil {
		panic(fmt.Errorf("root file path '%s' could not be opened, error: '%s'", g.RootPath, err))
	}
	defer rootdirhandle.Close()

	flist, err := rootdirhandle.Readdir(-1)
	if err != nil {
		panic(fmt.Errorf("call to Readdir() on root file path '%s' failed: '%s'", g.RootPath, err))
	}
	g.PngCount = 0
	g.SubDirCount = 0

	sort.Sort(byName(flist))

	for i := range flist {
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
	g.Update(g.RootPath)
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
