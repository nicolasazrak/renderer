package main

import (
	"fmt"
	"image"
	"math"
	"time"

	"github.com/tfriedel6/canvas/sdlcanvas"
)

type Scene struct {
	models         []*Model
	zBuffer        []float64
	pixBuffer      []uint8
	width          int
	height         int
	fWidth         float64
	fHeight        float64
	lightPosition  Vector3
	camera         Camera
	trianglesDrawn int
}

type Camera struct {
	position    Vector3
	orientation Vector3
	display     Vector3
}

func (scene *Scene) drawTriangle(model *Model, triangle *Triangle) {
	scene.trianglesDrawn++
	pts := triangle.projection
	minbbox, maxbbox := boundingBox(pts, 0, scene.fWidth-1, 0, scene.fHeight-1)
	A01 := int(pts[0].y - pts[1].y)
	B01 := int(pts[1].x - pts[0].x)
	A12 := int(pts[1].y - pts[2].y)
	B12 := int(pts[2].x - pts[1].x)
	A20 := int(pts[2].y - pts[0].y)
	B20 := int(pts[0].x - pts[2].x)

	w1_row := orient2d(pts[2], pts[0], minbbox.x, minbbox.y)
	w0_row := orient2d(pts[1], pts[2], minbbox.x, minbbox.y)
	w2_row := orient2d(pts[0], pts[1], minbbox.x, minbbox.y)

	for y := minbbox.y; y <= maxbbox.y; y++ {
		w0 := w0_row
		w1 := w1_row
		w2 := w2_row

		for x := minbbox.x; x <= maxbbox.x; x++ {
			if (w0 | w1 | w2) >= 0 {
				sum := float64(w0 + w1 + w2)
				l0 := float64(w0) / sum
				l1 := float64(w1) / sum
				l2 := float64(w2) / sum

				zPos := l0*triangle.verts[0].z + l1*triangle.verts[1].z + l2*triangle.verts[2].z
				zPos = -zPos // WTF ??
				idx := int(x) + (int(y))*scene.width

				if zPos < scene.zBuffer[idx] {
					scene.zBuffer[idx] = zPos
					r, g, b := model.shader.shade(scene, triangle, l0, l1, l2)
					scene.setAt(int(x), int(y), r, g, b)
				}
			}

			w0 += A12
			w1 += A20
			w2 += A01
		}

		// One row step
		w0_row += B12
		w1_row += B20
		w2_row += B01
	}
}

func (scene *Scene) project(triangle *Triangle) {
	distance := 2.
	for i, vert := range triangle.verts {
		denom := 1 - vert.z/distance
		triangle.projection[i].x = (vert.x/denom + 1.) * scene.fWidth / 2.
		triangle.projection[i].y = (vert.y/denom + 1.) * scene.fHeight / 2.
	}
}

func (scene *Scene) drawModel(model *Model) {
	for _, triangle := range model.triangles {
		scene.project(triangle)
		scene.drawTriangle(model, triangle)
	}
}

func (scene *Scene) drawModels() {
	for _, model := range scene.models {
		scene.drawModel(model)
	}
}

func (scene *Scene) setAt(x int, yInverted int, r uint8, g uint8, b uint8) {
	y := scene.height - yInverted - 1
	pixIdx := (x + y*scene.width) * 4
	scene.pixBuffer[pixIdx] = r
	scene.pixBuffer[pixIdx+1] = g
	scene.pixBuffer[pixIdx+2] = b
	scene.pixBuffer[pixIdx+3] = uint8(255)
}

func (scene *Scene) toImage() *image.RGBA {
	image := &image.RGBA{
		Pix:    scene.pixBuffer,
		Stride: scene.width * 4,
		Rect:   image.Rect(0, 0, scene.width, scene.height),
	}

	return image
}

func (scene *Scene) cleanBuffer() {
	for idx := 0; idx < len(scene.pixBuffer); idx += 4 {
		scene.pixBuffer[idx] = uint8(0)
		scene.pixBuffer[idx+1] = uint8(0)
		scene.pixBuffer[idx+2] = uint8(0)
		scene.pixBuffer[idx+3] = uint8(255)
	}

	for i := range scene.zBuffer {
		scene.zBuffer[i] = 999999
	}
	scene.trianglesDrawn = 0
}

func (scene *Scene) render() *image.RGBA {
	scene.cleanBuffer()
	scene.drawModels()
	return scene.toImage()
}

func main() {
	wnd, cv, err := sdlcanvas.CreateWindow(1000, 1000, "Hello")
	if err != nil {
		panic(err)
	}
	defer wnd.Destroy()

	scene := Scene{
		models:        []*Model{},
		zBuffer:       []float64{},
		pixBuffer:     []uint8{},
		width:         cv.Width(),
		height:        cv.Height(),
		fWidth:        float64(cv.Width()),
		fHeight:       float64(cv.Height()),
		lightPosition: Vector3{0, 1, -10},
		camera:        Camera{},
	}

	scene.models = append(scene.models, newXZSquare(-0.2, 0.9, newTextureShader("assets/grass.texture.jpg")))
	scene.models = append(scene.models, newXYSquare(-0.9, 0.9, newTextureShader("assets/brick.texture.jpg")))
	// scene.models = append(scene.models, parseModel("assets/head.obj", newTextureShader("assets/head.texture.tga")))

	scene.pixBuffer = make([]uint8, scene.width*scene.height*4)
	scene.zBuffer = make([]float64, scene.width*scene.height)

	// f, err := os.Create("cpu")
	// if err != nil {
	// 	panic(err)
	// }
	// pprof.StartCPUProfile(f)
	// defer pprof.StopCPUProfile()

	t := float64(0)
	wnd.MainLoop(func() {
		start := time.Now()

		t++
		scene.lightPosition.z = math.Cos(t/10) * 3
		scene.lightPosition.x = math.Sin(t/10) * 3

		scene.models[0].moveX(-0.003)

		cv.PutImageData(scene.render(), 0, 0)
		elapsed := time.Since(start)
		if true {
			fmt.Println(elapsed.String(), "Triangles = ", scene.trianglesDrawn)
		}
	})
}
