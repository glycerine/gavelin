package main_test

import (
	"os"
	"time"

	gav "github.com/glycerine/gavelin"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Gavelin", func() {
	var testRootPath string = "gavtest"
	const updateIntervalMilliseconds = 1

	BeforeEach(func() {
		os.RemoveAll(testRootPath)
	})
	AfterEach(func() {
		//		os.RemoveAll(testRootPath)
	})

	Context("when the directory it is watching has an image file added", func() {
		It("should notice within 1 second and include the new image in the display", func() {
			g := gav.NewGavelin(testRootPath, updateIntervalMilliseconds)
			g.Start()
			defer g.Stop()
			Expect(g.DisplayedPngCount()).To(Equal(0))

			gav.GenerateNewPng(testRootPath + "/newpic.png")
			gav.GenerateNewPng(testRootPath + "/coolhist.png")
			time.Sleep(3 * updateIntervalMilliseconds * time.Millisecond)
			Expect(g.DisplayedPngCount()).To(Equal(2))
			Expect(g.FileNames()).To(ContainElement("newpic.png"))
			Expect(g.FileNames()).To(ContainElement("coolhist.png"))
		})
	})

	Context("when the directory it is watching has a new sub-directory added", func() {
		It("should notice within 1 second and include the new sub-directory in the display", func() {
			g := gav.NewGavelin(testRootPath, updateIntervalMilliseconds)
			g.Start()
			defer g.Stop()
			Expect(g.DirCount()).To(Equal(0))
			gav.GenerateNewSubdir(testRootPath + "/subdir1")
			gav.GenerateNewSubdir(testRootPath + "/subdir2")
			time.Sleep(3 * updateIntervalMilliseconds * time.Millisecond)
			Expect(g.DirCount()).To(Equal(2))
			Expect(g.DirList()).To(ContainElement("subdir1"))
		})
	})

})
