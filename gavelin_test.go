package main_test

import (
	"fmt"
	"os"
	"time"

	gav "github.com/glycerine/gavelin"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Gavelin", func() {
	var testRootPath string = "gavtest"

	BeforeEach(func() {
		os.RemoveAll(testRootPath)
	})
	//	AfterEach(func() {
	//		os.RemoveAll(testRootPath)
	//	})

	Context("when the directory it is watching has an image file added", func() {
		FIt("should notice within 1 second and include the new image in the display", func() {
			g := gav.NewGavelin(testRootPath)
			fmt.Printf("g = %#v\n", g)
			g.Start()
			defer g.Stop()
			Expect(g.DisplayedPngCount()).To(Equal(0))
			fmt.Printf("1st time, g.DisplayedPngCount() = %d\n", g.DisplayedPngCount())

			gav.GenerateNewPng(testRootPath + "/newpic.png")
			time.Sleep(2100 * time.Millisecond)
			Expect(g.DisplayedPngCount()).To(Equal(1))
			fmt.Printf("2nd time, g.DisplayedPngCount() = %d\n", g.DisplayedPngCount())

		})
	})

	Context("when the directory it is watching has a new sub-directory added", func() {
		It("should notice within 1 second and include the new sub-directory in the display", func() {
			g := gav.NewGavelin(testRootPath)
			g.Start()
			defer g.Stop()
			Expect(g.DirCount()).To(Equal(0))
			gav.GenerateNewSubdir(testRootPath + "/subdir")
			time.Sleep(2100 * time.Millisecond)
			Expect(g.DirCount()).To(Equal(1))
		})
	})

})
