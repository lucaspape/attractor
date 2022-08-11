package main

import (
	"fmt"
	"github.com/g3n/engine/app"
	"github.com/g3n/engine/camera"
	"github.com/g3n/engine/core"
	"github.com/g3n/engine/geometry"
	"github.com/g3n/engine/gls"
	"github.com/g3n/engine/graphic"
	"github.com/g3n/engine/gui"
	"github.com/g3n/engine/light"
	"github.com/g3n/engine/material"
	"github.com/g3n/engine/math32"
	"github.com/g3n/engine/renderer"
	"github.com/g3n/engine/window"
	"image"
	"image/color"
	"image/png"
	"os"
	"strconv"
	"sync"
	"time"
)

var frameDir = "frames"

var n int32 = 100000

var threads = 4

var steps float32 = 100
var step float32 = 1

var wg sync.WaitGroup

func main() {
	a := app.App()
	scene := core.NewNode()

	gui.Manager().Set(scene)

	cam := camera.New(1)
	cam.SetPosition(0, 0, 70)
	scene.Add(cam)

	camera.NewOrbitControl(cam)

	onResize := func(evname string, ev interface{}) {
		width, height := a.GetSize()
		a.Gls().Viewport(0, 0, int32(width), int32(height))
		cam.SetAspect(float32(width) / float32(height))
	}
	a.Subscribe(window.OnWindowSize, onResize)
	onResize("", nil)

	scene.Add(light.NewAmbient(&math32.Color{R: 1.0, G: 1.0, B: 1.0}, 0.8))
	pointLight := light.NewPoint(&math32.Color{R: 1, G: 1, B: 1}, 5.0)
	pointLight.SetPosition(1, 0, 80)
	scene.Add(pointLight)

	a.Gls().ClearColor(0, 0, 0, 1)

	sMat := material.NewStandard(&math32.Color{
		R: 0,
		G: 0,
		B: 255,
	})

	vectors := lorenzAttractor(n)
	var spheres []*graphic.Mesh

	s := geometry.NewSphere(0.01, 10, 10)

	for _, v := range vectors {
		mesh := graphic.NewMesh(s, sMat)

		mesh.SetPosition(v.X, v.Y, v.Z)
		scene.Add(mesh)

		spheres = append(spheres, mesh)
	}

	chunks := chunkSlice(spheres, len(spheres)/threads)
	vectorChunks := chunkSlice(vectors, len(spheres)/threads)

	lastFrameTime := time.Now().UnixNano()
	frames := 0

	var frame int64
	frame = 0

	fpsText := gui.NewLabel("0 fps")
	fpsText.SetPosition(10, 10)
	fpsText.SetSize(40, 40)
	scene.Add(fpsText)

	frameTimeText := gui.NewLabel("")
	frameTimeText.SetPosition(10, 40)
	frameTimeText.SetSize(40, 40)
	scene.Add(frameTimeText)

	a.Run(func(renderer *renderer.Renderer, deltaTime time.Duration) {
		a.Gls().Clear(gls.DEPTH_BUFFER_BIT | gls.STENCIL_BUFFER_BIT | gls.COLOR_BUFFER_BIT)

		frameTimeText.SetText(strconv.FormatInt(deltaTime.Milliseconds(), 10) + " ms/frame")

		if (time.Now().UnixNano()-lastFrameTime)/1000000000.0 >= 1 {
			fpsText.SetText(strconv.Itoa(frames) + " fps")

			frames = 0
			lastFrameTime = time.Now().UnixNano()
		}

		if step >= steps {
			step = 1

			for i, chunk := range chunks {
				wg.Add(1)
				go resetAnimation(vectorChunks[i], chunk)
			}

			wg.Wait()
		}

		for i, chunk := range chunks {
			wg.Add(1)
			go animate(vectorChunks[i], chunk)
		}

		wg.Wait()

		step++

		_ = renderer.Render(scene, cam)

		saveFramebuffer(a, frame)

		frames++
		frame++
	})
}

func saveFramebuffer(a *app.Application, frame int64) {
	width, height := a.GetSize()
	data := a.Gls().ReadPixels(0, 0, width, height, gls.RGBA, gls.UNSIGNED_BYTE)

	saveFrame(data, frame, width, height)
}

func resetAnimation(vectors []math32.Vector3, spheres []*graphic.Mesh) {
	for i, s := range spheres {
		p := vectors[i]
		s.SetPosition(p.X, p.Y, p.Z)
	}

	wg.Done()
}

func animate(vectors []math32.Vector3, spheres []*graphic.Mesh) {
	for i, s := range spheres {
		if i+1 >= len(spheres) {
			continue
		}

		p := vectors[i]
		np := vectors[i+1]

		s.SetPosition(p.X+((np.X-p.X)*((1/steps)*step)), p.Y+((np.Y-p.Y)*((1/steps)*step)), p.Z+((np.Z-p.Z)*((1/steps)*step)))
	}

	wg.Done()
}

func lorenzAttractor(n int32) []math32.Vector3 {
	var v []math32.Vector3

	var (
		h float32
		a float32
		b float32
		c float32

		x0 float32
		y0 float32
		z0 float32
		x1 float32
		y1 float32
		z1 float32
	)

	h = 0.01
	a = 10.0
	b = 28.0
	c = 8.0 / 3.0

	x0 = 0.1
	y0 = 0
	z0 = 0

	var i int32

	for i = 0; i < n; i++ {
		x1 = x0 + h*a*(y0-x0)
		y1 = y0 + h*(x0*(b-z0)-y0)
		z1 = z0 + h*(x0*y0-c*z0)

		x0 = x1
		y0 = y1
		z0 = z1

		v = append(v, math32.Vector3{
			X: x0,
			Y: y0,
			Z: z0,
		})
	}

	return v
}

func chunkSlice[Type comparable](slice []Type, chunkSize int) [][]Type {
	var chunks [][]Type
	for i := 0; i < len(slice); i += chunkSize {
		end := i + chunkSize

		// necessary check to avoid slicing beyond
		// slice capacity
		if end > len(slice) {
			end = len(slice)
		}

		chunks = append(chunks, slice[i:end])
	}

	return chunks
}

func saveFrame(data []byte, number int64, width int, height int) {
	upLeft := image.Point{
		X: 0,
		Y: 0,
	}

	lowRight := image.Point{
		X: width,
		Y: height,
	}

	img := image.NewRGBA(image.Rectangle{
		Min: upLeft,
		Max: lowRight,
	})

	var (
		r uint8
		g uint8
		b uint8
		a uint8
	)

	r = 0
	g = 0
	b = 0
	a = 0

	var rgbas []color.RGBA

	k := 1

	for _, bit := range data {
		switch k {
		case 1:
			r = bit
			break
		case 2:
			g = bit
			break
		case 3:
			b = bit
			break
		case 4:
			a = bit
			k = 0

			rgbas = append(rgbas, color.RGBA{
				R: r,
				G: g,
				B: b,
				A: a,
			})

			break
		}

		k++
	}

	x := width
	y := 0

	for i := len(rgbas) - 1; i >= 0; i-- {
		img.Set(x, y, rgbas[i])

		if x == 0 {
			y++
			x = width
		}

		x--
	}

	out, err := os.Create(frameDir + "/frame_" + strconv.FormatInt(number, 10) + ".png")

	if err != nil {
		fmt.Println(err)
		return
	}

	defer out.Close()

	err = png.Encode(out, img)

	if err != nil {
		fmt.Println(err)
	}
}
