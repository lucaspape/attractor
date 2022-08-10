package main

import (
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
	"time"
)

func main() {
	// Create application and scene
	a := app.App()
	scene := core.NewNode()

	// Set the scene to be managed by the gui manager
	gui.Manager().Set(scene)

	// Create perspective camera
	cam := camera.New(1)
	cam.SetPosition(0, 0, 70)
	scene.Add(cam)

	// Set up orbit control for the camera
	camera.NewOrbitControl(cam)

	// Set up callback to update viewport and camera aspect ratio when the window is resized
	onResize := func(evname string, ev interface{}) {
		// Get framebuffer size and update viewport accordingly
		width, height := a.GetSize()
		a.Gls().Viewport(0, 0, int32(width), int32(height))
		// Update the camera's aspect ratio
		cam.SetAspect(float32(width) / float32(height))
	}
	a.Subscribe(window.OnWindowSize, onResize)
	onResize("", nil)

	// Create and add lights to the scene
	scene.Add(light.NewAmbient(&math32.Color{1.0, 1.0, 1.0}, 0.8))
	pointLight := light.NewPoint(&math32.Color{1, 1, 1}, 5.0)
	pointLight.SetPosition(1, 0, 80)
	scene.Add(pointLight)

	// Set background color to gray
	a.Gls().ClearColor(0, 0, 0, 1)

	var spheres []*graphic.Mesh

	l := lorenzAttractor(30000)

	for _, v := range l {
		sMat := material.NewStandard(&math32.Color{
			R: 0,
			G: 0,
			B: 255,
		})

		s := geometry.NewSphere(0.01, 10, 10)
		mesh := graphic.NewMesh(s, sMat)
		mesh.SetPosition(v.X, v.Y, v.Z)
		scene.Add(mesh)

		spheres = append(spheres, mesh)
	}

	var steps float32
	var step float32

	steps = 100
	step = 1

	// Run the application
	a.Run(func(renderer *renderer.Renderer, deltaTime time.Duration) {
		a.Gls().Clear(gls.DEPTH_BUFFER_BIT | gls.STENCIL_BUFFER_BIT | gls.COLOR_BUFFER_BIT)

		if step >= steps {
			step = 1

			for i, s := range spheres {
				p := l[i]
				s.SetPosition(p.X, p.Y, p.Z)
			}
		}

		for i, s := range spheres {
			if i+1 >= len(spheres) {
				continue
			}

			p := l[i]
			np := l[i+1]

			s.SetPosition(p.X+((np.X-p.X)*((1/steps)*step)), p.Y+((np.Y-p.Y)*((1/steps)*step)), p.Z+((np.Z-p.Z)*((1/steps)*step)))
		}

		step++

		renderer.Render(scene, cam)
	})
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
