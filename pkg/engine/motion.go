package engine

import (
	"encoding/base64"
	"fmt"
	"image"
	"image/draw"
	"io"
	"math"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// 5x7 matrix font for digits 0-9
var DigitFont5x7 = map[rune][7][5]byte{
	'0': {
		{0, 1, 1, 1, 0},
		{1, 0, 0, 0, 1},
		{1, 0, 0, 1, 1},
		{1, 0, 1, 0, 1},
		{1, 1, 0, 0, 1},
		{1, 0, 0, 0, 1},
		{0, 1, 1, 1, 0},
	},
	'1': {
		{0, 0, 1, 0, 0},
		{0, 1, 1, 0, 0},
		{0, 0, 1, 0, 0},
		{0, 0, 1, 0, 0},
		{0, 0, 1, 0, 0},
		{0, 0, 1, 0, 0},
		{0, 1, 1, 1, 0},
	},
	'2': {
		{0, 1, 1, 1, 0},
		{1, 0, 0, 0, 1},
		{0, 0, 0, 0, 1},
		{0, 0, 1, 1, 0},
		{0, 1, 0, 0, 0},
		{1, 0, 0, 0, 0},
		{1, 1, 1, 1, 1},
	},
	'3': {
		{1, 1, 1, 1, 0},
		{0, 0, 0, 0, 1},
		{0, 0, 0, 0, 1},
		{0, 1, 1, 1, 0},
		{0, 0, 0, 0, 1},
		{0, 0, 0, 0, 1},
		{1, 1, 1, 1, 0},
	},
	'4': {
		{1, 0, 0, 0, 1},
		{1, 0, 0, 0, 1},
		{1, 0, 0, 0, 1},
		{1, 1, 1, 1, 1},
		{0, 0, 0, 0, 1},
		{0, 0, 0, 0, 1},
		{0, 0, 0, 0, 1},
	},
	'5': {
		{1, 1, 1, 1, 1},
		{1, 0, 0, 0, 0},
		{1, 1, 1, 1, 0},
		{0, 0, 0, 0, 1},
		{0, 0, 0, 0, 1},
		{1, 0, 0, 0, 1},
		{0, 1, 1, 1, 0},
	},
	'6': {
		{0, 1, 1, 1, 0},
		{1, 0, 0, 0, 0},
		{1, 1, 1, 1, 0},
		{1, 0, 0, 0, 1},
		{1, 0, 0, 0, 1},
		{1, 0, 0, 0, 1},
		{0, 1, 1, 1, 0},
	},
	'7': {
		{1, 1, 1, 1, 1},
		{0, 0, 0, 0, 1},
		{0, 0, 0, 1, 0},
		{0, 0, 1, 0, 0},
		{0, 1, 0, 0, 0},
		{0, 1, 0, 0, 0},
		{0, 1, 0, 0, 0},
	},
	'8': {
		{0, 1, 1, 1, 0},
		{1, 0, 0, 0, 1},
		{1, 0, 0, 0, 1},
		{0, 1, 1, 1, 0},
		{1, 0, 0, 0, 1},
		{1, 0, 0, 0, 1},
		{0, 1, 1, 1, 0},
	},
	'9': {
		{0, 1, 1, 1, 0},
		{1, 0, 0, 0, 1},
		{1, 0, 0, 0, 1},
		{0, 1, 1, 1, 1},
		{0, 0, 0, 0, 1},
		{0, 0, 0, 0, 1},
		{0, 1, 1, 1, 0},
	},
}

type MotionCaptcha struct {
	Width      int      `json:"width"`
	Height     int      `json:"height"`
	FrameCount int      `json:"frameCount"`
	FPS        int      `json:"fps"`
	GrainSize  int      `json:"grainSize"`
	Mime       string   `json:"mime,omitempty"`
	Video      string   `json:"video,omitempty"`  // Base64 MP4 (GOTCHA version 1 style)
	Poster     string   `json:"poster,omitempty"` // Base64 first-frame PNG RGB
	Frames     []string `json:"frames,omitempty"`
}

// Appearance lets an integrator brand a challenge without changing its motion mechanics.
// Colors are CSS-style hex values: the first is the dark color; later colors are highlights.
type Appearance struct {
	Palette [][3]byte
	Texture string
}

func ParseAppearance(colors []string, texture string) Appearance {
	appearance := Appearance{Palette: [][3]byte{{0, 0, 0}, {255, 255, 255}}, Texture: "noise"}
	if texture == "dots" || texture == "diagonal" || texture == "grid" || texture == "waves" {
		appearance.Texture = texture
	}
	if len(colors) < 2 {
		return appearance
	}
	palette := make([][3]byte, 0, len(colors))
	for _, value := range colors {
		value = strings.TrimPrefix(strings.TrimSpace(value), "#")
		if len(value) != 6 {
			continue
		}
		r, errR := strconv.ParseUint(value[0:2], 16, 8)
		g, errG := strconv.ParseUint(value[2:4], 16, 8)
		b, errB := strconv.ParseUint(value[4:6], 16, 8)
		if errR == nil && errG == nil && errB == nil {
			palette = append(palette, [3]byte{byte(r), byte(g), byte(b)})
		}
	}
	if len(palette) >= 2 {
		appearance.Palette = palette
	}
	return appearance
}

const (
	// Compact captcha widget: digits stay centered while noise slides.
	captchaWidth    = 960
	captchaHeight   = 400
	captchaGrain    = 2
	captchaFPS      = 30
	captchaDuration = 5.0
	captchaSpeed    = 0.45
	captchaFeather  = 0.0
	captchaStroke   = 6
)

var boldFontPaths = []string{
	"/System/Library/Fonts/Supplemental/Arial Bold.ttf",
	"/System/Library/Fonts/Supplemental/Helvetica.ttc",
	"/Library/Fonts/Arial Bold.ttf",
	"/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf",
	"/usr/share/fonts/dejavu/DejaVuSans-Bold.ttf",
}

// FitDigitLayout picks the largest 5x7 scale that still fits the canvas.
func FitDigitLayout(charCount, width, height int) (scale, spacing int) {
	n := charCount
	if n < 1 {
		n = 1
	}
	maxScaleH := (height - 6) / 7
	if maxScaleH < 6 {
		maxScaleH = 6
	}
	for s := maxScaleH; s >= 6; s-- {
		gap := s
		if gap < 6 {
			gap = 6
		}
		total := n*5*s + (n-1)*gap
		if total <= width-12 {
			return s, gap
		}
	}
	return 6, 6
}

// CreateTextMask renders 5x7 matrix digits into a 2D binary mask
func CreateTextMask(text string, width, height, scale, spacing int) []byte {
	mask := make([]byte, width*height)
	chars := []rune(text)
	charW := 5 * scale
	charH := 7 * scale

	totalW := len(chars)*charW + (len(chars)-1)*spacing
	startX := (width - totalW) / 2
	if startX < 8 {
		startX = 8
	}
	startY := (height - charH) / 2
	if startY < 8 {
		startY = 8
	}

	for idx, ch := range chars {
		grid, ok := DigitFont5x7[ch]
		if !ok {
			grid = DigitFont5x7['0']
		}
		originX := startX + idx*(charW+spacing)
		originY := startY

		for r := 0; r < 7; r++ {
			for c := 0; c < 5; c++ {
				if grid[r][c] == 1 {
					for py := 0; py < scale; py++ {
						for px := 0; px < scale; px++ {
							x := originX + c*scale + px
							y := originY + r*scale + py
							if x >= 0 && x < width && y >= 0 && y < height {
								mask[y*width+x] = 1
							}
						}
					}
				}
			}
		}
	}

	return mask
}

// GenerateMotionFrames builds a compact captcha video: digits stay centered
// while background noise slides horizontally and text-region noise slides vertically.
func GenerateMotionFrames(code string, defenseLevel string, appearance Appearance) *MotionCaptcha {
	width := captchaWidth
	height := captchaHeight
	grain := captchaGrain
	fps := captchaFPS
	frameCount := int(math.Max(1, math.Round(captchaDuration*float64(fps))))
	speed := captchaSpeed

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	mask := dilateMask(renderSoftTextMask(code, width, height, captchaFeather), width, height, captchaStroke)
	bg := tileHorizontal(makeTexture(rng, height, width, grain, appearance.Texture), height, width)
	fg := tileVertical(makeTexture(rng, height, width, grain, appearance.Texture), height, width)

	encoder, _ := startMP4Encoder(width, height, fps)
	var poster []byte
	var fallback [][]byte

	for f := 0; f < frameCount; f++ {
		bgOff := int(math.Round(float64(f)*speed)) % width
		// A faster vertical foreground against a slower horizontal background
		// gives the eye a decisive motion boundary around each digit.
		fgOff := int(math.Round(float64(f)*speed*1.9)) % height
		if defenseLevel == "ultra" {
			fgOff = int(math.Round(float64(f)*speed*2.2)) % height
		}

		rgb := make([]byte, width*height*3)
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				a := mask[y*width+x]
				bgVal := bg[y*(width*2)+(bgOff+x)]
				fgVal := fg[(fgOff+y)*width+x]
				v := bgVal*(1.0-a) + fgVal*a
				sx, sy := x+bgOff, y
				if a > 0.5 {
					sx, sy = x, y+fgOff
				}
				color := appearance.Palette[0]
				if v >= 0.5 {
					color = appearance.Palette[1+(sx*17+sy*31)%(len(appearance.Palette)-1)]
				}
				i := (y*width + x) * 3
				rgb[i], rgb[i+1], rgb[i+2] = color[0], color[1], color[2]
			}
		}
		if f == 0 {
			poster = rgbToRGBA(rgb, width, height)
		}
		if encoder != nil {
			if werr := encoder.write(rgb); werr != nil {
				_ = encoder.abort()
				encoder = nil
			}
		} else if len(fallback) < 48 {
			fallback = append(fallback, rgb)
		}
	}

	out := &MotionCaptcha{
		Width:      width,
		Height:     height,
		FrameCount: frameCount,
		FPS:        fps,
		GrainSize:  grain,
		Poster:     base64.StdEncoding.EncodeToString(poster),
	}

	if encoder != nil {
		if video, cerr := encoder.close(); cerr == nil && len(video) > 0 {
			out.Mime = "video/mp4"
			out.Video = base64.StdEncoding.EncodeToString(video)
			return out
		}
	}

	frames := make([]string, len(fallback))
	for i := range fallback {
		frames[i] = base64.StdEncoding.EncodeToString(rgbToRGBA(fallback[i], width, height))
	}
	out.Frames = frames
	if len(frames) > 0 {
		out.FrameCount = len(frames)
	}
	return out
}

func makeGrayscaleNoise(rng *rand.Rand, height, width, grain int) []float32 {
	cols := int(math.Ceil(float64(width) / float64(grain)))
	rows := int(math.Ceil(float64(height) / float64(grain)))
	base := make([]float32, rows*cols)
	for i := range base {
		if rng.Intn(2) == 0 {
			base[i] = 0
		} else {
			base[i] = 1
		}
	}
	out := make([]float32, height*width)
	for y := 0; y < height; y++ {
		by := y / grain
		if by >= rows {
			by = rows - 1
		}
		for x := 0; x < width; x++ {
			bx := x / grain
			if bx >= cols {
				bx = cols - 1
			}
			out[y*width+x] = base[by*cols+bx]
		}
	}
	return out
}

func makeTexture(rng *rand.Rand, height, width, grain int, texture string) []float32 {
	if texture == "noise" {
		return makeGrayscaleNoise(rng, height, width, grain)
	}
	phaseX, phaseY := rng.Intn(12), rng.Intn(12)
	out := make([]float32, height*width)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			xx, yy := x+phaseX, y+phaseY
			on := false
			switch texture {
			case "dots":
				on = xx%6 == 0 && yy%6 == 0
			case "diagonal":
				on = (xx+yy)%10 < 3
			case "grid":
				on = xx%8 < 2 || yy%8 < 2
			case "waves":
				on = math.Sin(float64(xx)*0.65)+math.Sin(float64(yy)*0.45) > 0.45
			}
			if on {
				out[y*width+x] = 1
			}
		}
	}
	return out
}

func tileHorizontal(src []float32, height, width int) []float32 {
	dst := make([]float32, height*width*2)
	for y := 0; y < height; y++ {
		copy(dst[y*width*2:y*width*2+width], src[y*width:(y+1)*width])
		copy(dst[y*width*2+width:y*width*2+width*2], src[y*width:(y+1)*width])
	}
	return dst
}

func tileVertical(src []float32, height, width int) []float32 {
	dst := make([]float32, height*2*width)
	copy(dst[:height*width], src)
	copy(dst[height*width:], src)
	return dst
}

func rgbToRGBA(rgb []byte, width, height int) []byte {
	out := make([]byte, width*height*4)
	for i, j := 0, 0; i < len(rgb); i, j = i+3, j+4 {
		out[j] = rgb[i]
		out[j+1] = rgb[i+1]
		out[j+2] = rgb[i+2]
		out[j+3] = 255
	}
	return out
}

func renderSoftTextMask(text string, width, height int, feather float64) []float32 {
	mask := make([]float32, width*height)
	preferred := int(float64(height) * 0.96)
	if face, size := fitBoldFace(text, width, height, preferred); face != nil {
		img := image.NewGray(image.Rect(0, 0, width, height))
		draw.Draw(img, img.Bounds(), image.Black, image.Point{}, draw.Src)
		d := &font.Drawer{Dst: img, Src: image.White, Face: face}
		bounds, _ := d.BoundString(text)
		textW := (bounds.Max.X - bounds.Min.X).Ceil()
		textH := (bounds.Max.Y - bounds.Min.Y).Ceil()
		x := (width-textW)/2 - bounds.Min.X.Floor()
		y := (height-textH)/2 - bounds.Min.Y.Floor()
		d.Dot = fixed.P(x, y)
		d.DrawString(text)
		_ = size
		for i, px := range img.Pix {
			mask[i] = float32(px) / 255.0
		}
	} else {
		scale, spacing := FitDigitLayout(len(text), width, height)
		hard := CreateTextMask(text, width, height, scale, spacing)
		for i, v := range hard {
			if v == 1 {
				mask[i] = 1
			}
		}
	}
	if feather > 0 {
		mask = boxBlur(mask, width, height, int(math.Max(1, math.Round(feather))))
	}
	return mask
}

func fitBoldFace(text string, width, height, preferred int) (font.Face, int) {
	maxW := int(float64(width) * 0.96)
	maxH := int(float64(height) * 0.94)
	for size := preferred; size >= 28; size -= 4 {
		face := loadBoldFace(float64(size))
		if face == nil {
			return nil, 0
		}
		d := &font.Drawer{Face: face}
		b, _ := d.BoundString(text)
		if (b.Max.X-b.Min.X).Ceil() <= maxW && (b.Max.Y-b.Min.Y).Ceil() <= maxH {
			return face, size
		}
	}
	return loadBoldFace(28), 28
}

func loadBoldFace(size float64) font.Face {
	for _, p := range boldFontPaths {
		if filepath.Ext(p) == ".ttc" {
			continue
		}
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		tt, err := opentype.Parse(data)
		if err != nil {
			continue
		}
		face, err := opentype.NewFace(tt, &opentype.FaceOptions{Size: size, DPI: 72, Hinting: font.HintingFull})
		if err != nil {
			continue
		}
		return face
	}
	return nil
}

func dilateMask(mask []float32, width, height, radius int) []float32 {
	if radius <= 0 {
		return mask
	}
	out := make([]float32, len(mask))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			max := mask[y*width+x]
			if max >= 1 {
				out[y*width+x] = 1
				continue
			}
			for dy := -radius; dy <= radius && max < 1; dy++ {
				yy := y + dy
				if yy < 0 || yy >= height {
					continue
				}
				for dx := -radius; dx <= radius; dx++ {
					xx := x + dx
					if xx < 0 || xx >= width {
						continue
					}
					if v := mask[yy*width+xx]; v > max {
						max = v
						if max >= 1 {
							break
						}
					}
				}
			}
			out[y*width+x] = max
		}
	}
	return out
}

func boxBlur(src []float32, width, height, radius int) []float32 {
	tmp := make([]float32, len(src))
	dst := make([]float32, len(src))
	copy(dst, src)
	passes := 2
	for p := 0; p < passes; p++ {
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				var sum float32
				var n float32
				for k := -radius; k <= radius; k++ {
					xx := x + k
					if xx < 0 || xx >= width {
						continue
					}
					sum += dst[y*width+xx]
					n++
				}
				tmp[y*width+x] = sum / n
			}
		}
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				var sum float32
				var n float32
				for k := -radius; k <= radius; k++ {
					yy := y + k
					if yy < 0 || yy >= height {
						continue
					}
					sum += tmp[yy*width+x]
					n++
				}
				dst[y*width+x] = sum / n
			}
		}
	}
	return dst
}

type mp4Encoder struct {
	cmd     *exec.Cmd
	pipe    io.WriteCloser
	dir     string
	outPath string
}

func startMP4Encoder(width, height, fps int) (*mp4Encoder, error) {
	dir, err := os.MkdirTemp("", "gotcha-mp4-*")
	if err != nil {
		return nil, err
	}
	outPath := filepath.Join(dir, "challenge.mp4")
	cmd := exec.Command("ffmpeg",
		"-y",
		"-loglevel", "error",
		"-f", "rawvideo",
		"-pix_fmt", "rgb24",
		"-s", fmt.Sprintf("%dx%d", width, height),
		"-r", strconv.Itoa(fps),
		"-i", "-",
		"-an",
		"-c:v", "libx264",
		"-preset", "veryfast",
		"-crf", "23",
		"-pix_fmt", "yuv420p",
		"-movflags", "+faststart",
		outPath,
	)
	cmd.Stderr = os.Stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		os.RemoveAll(dir)
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		os.RemoveAll(dir)
		return nil, err
	}
	return &mp4Encoder{cmd: cmd, dir: dir, outPath: outPath, pipe: stdin}, nil
}

func (e *mp4Encoder) write(frame []byte) error {
	_, err := e.pipe.Write(frame)
	return err
}

func (e *mp4Encoder) abort() error {
	if e.pipe != nil {
		_ = e.pipe.Close()
	}
	if e.cmd != nil {
		_ = e.cmd.Process.Kill()
		_ = e.cmd.Wait()
	}
	return os.RemoveAll(e.dir)
}

func (e *mp4Encoder) close() ([]byte, error) {
	if e.pipe != nil {
		_ = e.pipe.Close()
	}
	err := e.cmd.Wait()
	if err != nil {
		os.RemoveAll(e.dir)
		return nil, err
	}
	data, err := os.ReadFile(e.outPath)
	os.RemoveAll(e.dir)
	return data, err
}
