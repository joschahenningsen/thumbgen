package thumbgen

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"math"
	"os"
	"os/exec"
	"path"
	"strings"

	"hash/fnv"

	"github.com/tidwall/gjson"
	"golang.org/x/image/draw"
)

type Gen struct {
	file         string
	fileHash     string
	duration     float64 // in seconds
	width        int
	height       int
	frames       []string
	out          string
	quality      int
	interval     int
	frameDir     string
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

func WithStoreSingleFrames(dir string) Option {
	return func(g *Gen) {
		g.frameDir = dir
	}
}

func hash(s string) string {
	h := fnv.New32a()
	h.Write([]byte(s))
	return fmt.Sprintf("%x", h.Sum32())
}

func New(file string, width int, interval int, out string, options ...Option) (*Gen, error) {
	if interval < 1 {
		return nil, errors.New("invalid interval, must be >= 1")
	}
	// check for required software:
	_, err := exec.LookPath("ffmpeg")
	if err != nil {
		return nil, err
	}
	_, err = exec.LookPath("ffprobe")
	if err != nil {
		return nil, err
	}

	// check if file exists
	if !strings.HasPrefix(file, "http") {
		_, err = os.Stat(file)
		if err != nil {
			return nil, err
		}
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

	g := &Gen{file: file, fileHash: hash(file + out + fmt.Sprintf("%d", os.Getpid())), duration: totalDuration, width: width, interval: interval, height: height, frames: []string{}, out: out, frameDir: ""}
	for _, option := range options {
		option(g)
	}
	return g, nil
}

// GetHeight returns the height of the generated thumbnails.
func (g Gen) GetHeight() int {
	return g.height
}

// GenerateOne generates a single thumbnail at the middle of the file.
func (g *Gen) GenerateOne() (string, error) {
	err := g.exportFrameAt(int(g.duration / 2))
	if err != nil {
		return "", err
	}
	return g.frames[len(g.frames)-1], nil
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
	if g.frameDir == "" {
		err = g.cleanup()
		if err != nil {
			return err
		}
	}
	return nil
}

func (g *Gen) generateFrames() error {
	for t := 0; t < int(math.Ceil(g.duration)); t += g.interval {
		err := g.exportFrameAt(t)
		if err != nil {
			return err
		}
		if g.progressChan != nil {
			*g.progressChan <- int(float64(t) / g.duration * 100)
		}
	}
	return nil
}

func (g *Gen) exportFrameAt(time int) error {
	frame := path.Join(g.frameDir, fmt.Sprintf(g.fileHash+"-out-%d.jpeg", time))
	cmd := exec.Command("ffmpeg", "-y", "-ss", fmt.Sprintf("%d", time), "-i", g.file, "-vf", fmt.Sprintf("scale=%d:-1", g.width), "-frames:v", "1", frame)
	fmt.Println(cmd.String())
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg: %s :%v", string(out), err)
	}
	g.frames = append(g.frames, frame)
	return err
}

func (g Gen) merge() error {
	// Number of thumbnails per column.
	d := int(math.Ceil(math.Sqrt(g.duration / float64(g.interval))))

	dst := image.NewRGBA(image.Rect(0, 0, g.width*d, g.height*d))
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
					X: (i % d) * g.width,
					Y: (i / d) * g.height,
				},
				Max: image.Point{
					X: (i%d)*g.width + src.Bounds().Max.X,
					Y: (i/d)*g.height + src.Bounds().Max.Y,
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
	err = jpeg.Encode(f, dst, &jpeg.Options{Quality: g.quality})
	return err
}

func (g Gen) cleanup() error {
	for _, frame := range g.frames {
		err := os.Remove(frame)
		if err != nil && !os.IsNotExist(err) {
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

func CombineThumbs(primary, secondary, dest string) error {
	primaryF, err := os.Open(primary)
	if err != nil {
		return err
	}
	defer primaryF.Close()
	secondaryF, err := os.Open(secondary)
	if err != nil {
		return err
	}
	defer secondaryF.Close()

	primaryImg, _, err := image.Decode(primaryF)
	if err != nil {
		return err
	}
	secondaryImg, _, err := image.Decode(secondaryF)
	if err != nil {
		return err
	}
	secondaryImgS := image.NewRGBA(image.Rect(0, 0, primaryImg.Bounds().Max.X/3, primaryImg.Bounds().Max.Y/3))

	draw.CatmullRom.Scale(secondaryImgS, secondaryImgS.Rect, secondaryImg, secondaryImg.Bounds(), draw.Over, nil)

	// Create a new RGBA image with the same size as the primary image.
	combined := image.NewRGBA(primaryImg.Bounds())
	// Draw the primary image onto the combined image.
	draw.Draw(combined, primaryImg.Bounds(), primaryImg, image.Point{X: 0, Y: 0}, draw.Src)
	// Draw the secondary image onto the combined image at 1/4 the size.
	bounds := image.Rectangle{
		Min: image.Point{X: combined.Bounds().Max.X - combined.Bounds().Max.X/3 - 5, Y: 5},
		Max: image.Point{X: combined.Bounds().Max.X, Y: combined.Bounds().Max.Y},
	}
	draw.Draw(combined, bounds, secondaryImgS, image.Point{X: 0, Y: 0}, draw.Over)
	o, _ := os.Create(dest)
	defer o.Close()
	return jpeg.Encode(o, combined, &jpeg.Options{Quality: 90})
}
