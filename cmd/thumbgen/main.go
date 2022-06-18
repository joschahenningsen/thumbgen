package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/joschahenningsen/thumbgen"
	"github.com/schollz/progressbar/v3"
)

func main() {
	i := flag.String("i", "", "The input video file to generate thumbnails for.")
	o := flag.String("o", "", "The output sprite.")
	w := flag.Int("w", 0, "The width of the sprite.")
	t := flag.Int("t", 0, "The interval between to thumbnails in seconds.")
	jpegQ := flag.Int("q", 0, "Quality of jpeg output (optional).")
	fDir := flag.String("f", "", "Store single frames at <path> (optional).")
	flag.Parse()
	if *i == "" || *o == "" || *w == 0 || *t == 0 {
		flag.Usage()
		return
	}
	fmt.Printf("Generating thumbnails for %s\n", *i)
	progress := make(chan int)

	var opts []thumbgen.Option
	if jpegQ != nil {
		opts = append(opts, thumbgen.WithJpegCompression(*jpegQ))
	}
	if fDir != nil {
		opts = append(opts, thumbgen.WithStoreSingleFrames(*fDir))
	}
	opts = append(opts, thumbgen.WithProgressChan(&progress))

	gen, err := thumbgen.New(*i, *w, *t, *o, opts...)
	if err != nil {
		fmt.Println("create generator failed: ", err.Error())
		os.Exit(1)
	}
	go func() {
		bar := progressbar.Default(100)
		for {
			p := <-progress
			err := bar.Set(p)
			if err != nil {
				fmt.Println("progressbar set failed: ", err.Error())
			}
			if p == 100 {
				break
			}
		}
	}()
	err = gen.Generate()
	if err != nil {
		fmt.Println("generating thumbnails failed: ", err.Error())
		os.Exit(1)
	}
	fmt.Printf("Finished generating thumbnails. (resolution:%dx%d)\n", *w, gen.GetHeight())
}
