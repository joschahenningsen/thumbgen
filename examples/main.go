package main

import (
	"fmt"
	"thumbgen"
)

func main() {
	t, err := thumbgen.New("/home/joscha/Downloads/theo-2022-05-05-14-15COMB.mp4", 360, 100, "out.jpg", thumbgen.WithJpegCompression(100))
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
