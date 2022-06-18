package main

import (
	"fmt"
	"github.com/joschahenningsen/thumbgen"
)

func main() {
	t, err := thumbgen.New("/home/alex/Videos/it-sec.mp4", 160, 30, "out.jpg", thumbgen.WithJpegCompression(100))
	if err != nil {
		fmt.Println(err)
	}
	err = t.Generate()
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("done")
}
