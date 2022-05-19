package thumbgen

import (
	"fmt"
	"github.com/tidwall/gjson"
	ffmpeg "github.com/u2takey/ffmpeg-go"
	"image"
	"image/draw"
	"image/png"
	"log"
	"os"
	"os/exec"
)

type thumbgen struct {
	file     string
	duration int // in seconds
	width    int
	height   int
	frames   []string
}

func New(file string, width int) (*thumbgen, error) {
	probe, err := ffmpeg.Probe(file)
	if err != nil {
		return nil, err
	}
	totalDuration := gjson.Get(probe, "format.duration").Float()
	origW := gjson.Get(probe, "streams.#(codec_type==video).width").Int()
	origH := gjson.Get(probe, "streams.#(codec_type==video).height").Int()
	aspect := float64(width) / float64(origW)
	height := int(float64(origH) * aspect)
	t := &thumbgen{file, 0, width, height, []string{}}
	log.Println(t)
	for i := 0; i < 100; i++ {
		fmt.Println(i)
		t.exportFrameAt(int(totalDuration/100) * i)
	}
	t.merge()
	t.cleanup()
	return t, nil
}

func (t *thumbgen) exportFrameAt(time int) error {
	cmd := exec.Command("ffmpeg", "-y", "-ss", fmt.Sprintf("%d", time), "-i", t.file, "-vf", fmt.Sprintf("scale=%d:-1", t.width), "-frames:v", "1", "-q:v", "2", fmt.Sprintf("out-%05d.png", time))
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Println(err)
		log.Println(string(out))
	}
	t.frames = append(t.frames, fmt.Sprintf("out-%05d.png", time))
	return err
}

func (t thumbgen) merge() error {
	dst := image.NewRGBA(image.Rect(0, 0, t.width*10, t.height*10))
	for i, frame := range t.frames {
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
					X: (i % 10) * t.width,
					Y: (i / 10) * t.height,
				},
				Max: image.Point{
					X: (i%10)*t.width + src.Bounds().Max.X,
					Y: (i/10)*t.height + src.Bounds().Max.Y,
				},
			},
			src,
			image.Point{X: 0, Y: 0},
			draw.Src)
	}
	f, err := os.Create("out.png")
	if err != nil {
		return err
	}
	err = png.Encode(f, dst)
	return err
}

func (t thumbgen) cleanup() error {
	for _, frame := range t.frames {
		err := os.Remove(frame)
		if err != nil {
			return err
		}
	}
	return nil
}
