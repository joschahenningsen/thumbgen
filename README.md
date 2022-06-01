# thumbgen

Thumbgen generates a thumbnail sprite for a given video file, usable by a variety of video players like videojs with https://github.com/phloxic/videojs-sprite-thumbnails.

## Usage

### Via command line:
- `go install github.com/joschahenningsen/thumbgen/cmd/thumbgen@latest`
```bash
// generate a thumbnail sprite for video.mp4 with a width of 100px containing 100 thumbnails:
$ thumbgen -i video.mp4 -w 150 -n 100 -o thumbs.jpeg
```

```go
g, err := thumbgen.New("video.mp4", 360, 100, "out.jpg")
if err != nil {
	fmt.Println(err)
}
err = g.Generate()
if err != nil {
	fmt.Println(err)
	return
}
```

### Advanced: 

You can pass a jpeg compression factor (0: worst quality, 100: best) to `New`:

```go
g, err := thumbgen.New("video.mp4", 360, 100, "out.jpg", thumbgen.WithJpegCompression(90))
```

If you wish to track the progress, pass a channel to `New`:

```go
progress := make(chan int)
g, err := thumbgen.New("video.mp4", 360, 100, "out.jpg", thumbgen.WithProgressChan(&progress))
go func(){
	for {
		p := <-progress
		fmt.Println("progress: ", p, "%") // or whatever
		if p == 100 {
			break
		}
	}
}()
g.Generate()
```

If you desire to keep all frames (out0000.jpeg, ...) pass a path to `New`:


```go
g, err := thumbgen.New("video.mp4", 360, 100, "out.jpg", thumbgen.WithStoreSingleFrames("/tmp"))
```