package main

import (
	"flag"
	"fmt"
	"github.com/joschahenningsen/thumbgen"
	"github.com/schollz/progressbar/v3"
	"os"
)

func main() {
	i := flag.String("i", "", "The input video file to generate thumbnails for.")
	o := flag.String("o", "", "The output sprite.")
	w := flag.Int("w", 0, "The width of the sprite.")
	n := flag.Int("n", 0, "The number of thumbnails to generate.")
	jpegQ := flag.Int("q", 0, "Quality of jpeg output (optional).")
	flag.Parse()
	if *i == "" || *o == "" || *w == 0 || *n == 0 {
		flag.Usage()
		return
	}
	fmt.Printf("Generating thumbnails for %s\n", *i)
	progress := make(chan int)

	var opts []thumbgen.Option
	if jpegQ != nil {
		opts = append(opts, thumbgen.WithJpegCompression(*jpegQ))
	}
	opts = append(opts, thumbgen.WithProgressChan(&progress))

	gen, err := thumbgen.New(*i, *w, *n, *o, opts...)
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
	fmt.Printf("Generated %d thumbnails. (resolution:%dx%d)\n", *n, *w, gen.GetHeight())
}
