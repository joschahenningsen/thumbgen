package thumbgen

import (
	"bytes"
	"fmt"
	"github.com/tidwall/gjson"
	"image"
	"image/draw"
	"image/jpeg"
	"os"
	"os/exec"
)

type Gen struct {
	file         string
	duration     float64 // in seconds
	width        int
	height       int
	frames       []string
	out          string
	quality      int
	thumbNum     int
	progressChan *chan int
}

type Option func(*Gen)

// WithJpegCompression sets the jpeg compression quality where 0 is the worst quality and 100 is the best possible quality. Default is 95.
// If the output file is not a .jpeg or .jpg, this option has no affect.
func WithJpegCompression(compression int) Option {
	return func(g *Gen) {
		g.quality = compression
	}
}

func WithProgressChan(c *chan int) Option {
	return func(g *Gen) {
		g.progressChan = c
	}
}

func New(file string, width int, thumbNum int, out string, options ...Option) (*Gen, error) {
	// check for required software:
	_, err := exec.LookPath("ffmpeg")
	if err != nil {
		return nil, err
	}
	_, err = exec.LookPath("ffprobe")
	if err != nil {
		return nil, err
	}

	// probe video duration and resolution:
	probed, err := probe(file)
	if err != nil {
		return nil, err
	}
	totalDuration := gjson.Get(probed, "format.duration").Float()
	origW := gjson.Get(probed, "streams.#(codec_type==video).width").Int()
	origH := gjson.Get(probed, "streams.#(codec_type==video).height").Int()
	aspect := float64(width) / float64(origW)
	height := int(float64(origH) * aspect)

	g := &Gen{file: file, duration: totalDuration, width: width, thumbNum: thumbNum, height: height, frames: []string{}, out: out}
	for _, option := range options {
		option(g)
	}
	return g, nil
}

// GetHeight returns the height of the generated thumbnails.
func (g Gen) GetHeight() int {
	return g.height
}

func (g *Gen) Generate() error {
	err := g.generateFrames()
	if err != nil {
		return err
	}
	err = g.merge()
	if err != nil {
		return err
	}
	err = g.cleanup()
	if err != nil {
		return err
	}
	return nil
}

func (g *Gen) generateFrames() error {
	for i := 0; i < 100; i++ {
		err := g.exportFrameAt(int(g.duration/float64(g.thumbNum)) * i)
		if err != nil {
			return err
		}
		if g.progressChan != nil {
			*g.progressChan <- i
		}
	}
	return nil
}

func (g *Gen) exportFrameAt(time int) error {
	cmd := exec.Command("ffmpeg", "-y", "-ss", fmt.Sprintf("%d", time), "-i", g.file, "-vf", fmt.Sprintf("scale=%d:-1", g.width), "-frames:v", "1", "-q:v", "2", fmt.Sprintf("out-%05d.jpeg", time))
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg: %s :%v", string(out), err)
	}
	g.frames = append(g.frames, fmt.Sprintf("out-%05d.jpeg", time))
	return err
}

func (g Gen) merge() error {
	dst := image.NewRGBA(image.Rect(0, 0, g.width*10, g.height*10))
	for i, frame := range g.frames {
		f, err := os.Open(frame)
		if err != nil {
			return err
		}
		src, _, err := image.Decode(f)
		if err != nil {
			return err
		}
		draw.Draw(dst,
			image.Rectangle{
				Min: image.Point{
					X: (i % 10) * g.width,
					Y: (i / 10) * g.height,
				},
				Max: image.Point{
					X: (i%10)*g.width + src.Bounds().Max.X,
					Y: (i/10)*g.height + src.Bounds().Max.Y,
				},
			},
			src,
			image.Point{X: 0, Y: 0},
			draw.Src)
	}
	f, err := os.Create(g.out)
	if err != nil {
		return err
	}
	defer f.Close()
	err = jpeg.Encode(f, dst, &jpeg.Options{Quality: 75})
	return err
}

func (g Gen) cleanup() error {
	for _, frame := range g.frames {
		err := os.Remove(frame)
		if err != nil {
			return err
		}
	}
	return nil
}

func probe(file string) (string, error) {
	cmd := exec.Command("ffprobe", "-of", "json", "-show_format", "-show_streams", file)
	buf := bytes.NewBuffer(nil)
	cmd.Stdout = buf
	err := cmd.Run()
	if err != nil {
		return "", err
	}
	return string(buf.Bytes()), nil
}
