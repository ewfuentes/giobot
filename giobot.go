package main

import (
	"fmt"
	"image/color"
	"strconv"
	"time"

	"github.com/andyleap/gioframework"
	"github.com/go-gl/gl/v2.1/gl"
	"github.com/go-gl/glfw/v3.2/glfw"
	"github.com/golang/freetype/truetype"
	"github.com/llgcode/draw2d"
	"github.com/llgcode/draw2d/draw2dgl"
	"github.com/llgcode/draw2d/draw2dkit"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goitalic"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/goregular"

	"github.com/ewfuentes/giobot/ai"
)

type MyFontCache map[string]*truetype.Font

func (fc MyFontCache) Store(fd draw2d.FontData, font *truetype.Font) {
	fc[fd.Name] = font
}

func (fc MyFontCache) Load(fd draw2d.FontData) (*truetype.Font, error) {
	font, stored := fc[fd.Name]
	if !stored {
		return nil, fmt.Errorf("Font %s is not stored in font cache.", fd.Name)
	}
	return font, nil
}

var run int = 1
var width, height int = 800, 800
var redraw bool = true
var game *gioframework.Game

var playerColors []color.RGBA = []color.RGBA{
	{225, 39, 39, 255},
	{5, 41, 250, 255},
	{48, 105, 1, 255},
	{94, 25, 109, 255},
	{45, 107, 107, 255},
	{24, 52, 0, 255},
	{239, 150, 40, 255},
	{96, 17, 17, 255},
}

type Action struct {
	from int
	to   int
	is50 bool
}

var actionToMake Action = Action{-1, -1, false}

func onKey(w *glfw.Window, key glfw.Key, scancode int,
	action glfw.Action, mods glfw.ModifierKey) {

	if action != glfw.Press {
		return
	}

	switch key {
	case glfw.KeyQ:
		w.SetShouldClose(true)
	case glfw.KeySpace:
		actionToMake.from = -1
		actionToMake.to = -1
	case glfw.KeyA:
		actionToMake.to = actionToMake.from - 1
	case glfw.KeyD:
		actionToMake.to = actionToMake.from + 1
	case glfw.KeyW:
		actionToMake.to = actionToMake.from - 18
	case glfw.KeyS:
		actionToMake.to = actionToMake.from + 18
	}

	if actionToMake.from > 0 && actionToMake.to > 0 {
		fmt.Printf("Attacking from %v to %v\r\n", actionToMake.from, actionToMake.to)
		game.Attack(actionToMake.from,
			actionToMake.to,
			actionToMake.is50)
		actionToMake.from = actionToMake.to
		actionToMake.to = -1
	}
}

func getMapCellIdxFromMouseXY(x, y float64) int {
	border := float64(25)
	rowStep := float64(750 / 18.0)
	colStep := rowStep

	rowIdx := int((y - border) / rowStep)
	colIdx := int((x - border) / colStep)

	return rowIdx*18 + colIdx
}

func onClick(w *glfw.Window, button glfw.MouseButton,
	action glfw.Action, mod glfw.ModifierKey) {
	x, y := w.GetCursorPos()
	cellIdx := getMapCellIdxFromMouseXY(x, y)
	fmt.Println(button, action, mod, x, y, cellIdx)
	actionToMake.from = cellIdx
}

func redrawGame(g *gioframework.Game) {
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

	gc := draw2dgl.NewGraphicContext(width, height)
	gc.SetFontData(draw2d.FontData{
		Name:   "gomono",
		Family: draw2d.FontFamilyMono,
		Style:  draw2d.FontStyleNormal,
	})
	gc.SetLineWidth(1)

	// Draw the grid
	border := float64(25) //pixels
	gridLimitLeft := border
	gridLimitRight := float64(width) - border
	gridLimitTop := border
	gridLimitBottom := float64(height) - border
	colStep := (gridLimitRight - gridLimitLeft) / float64(g.Width)
	rowStep := (gridLimitBottom - gridLimitTop) / float64(g.Height)

	// Draw map items
	for r := 0; r < g.Height; r++ {
		for c := 0; c < g.Width; c++ {
			// Get top left and bottom right corners
			x1, y1 := colStep*float64(c)+border, rowStep*float64(r)+border
			x2, y2 := x1+colStep, y1+rowStep
			currCell := g.GameMap[r*g.Width+c]

			// draw the background color
			if currCell.Faction < -2 {
				gc.SetFillColor(color.RGBA{0x44, 0x44, 0x44, 0xFF})
			} else if currCell.Faction < -1 {
				gc.SetFillColor(color.RGBA{0x88, 0x88, 0x88, 0xFF})
			} else if currCell.Faction >= 0 {
				gc.SetFillColor(playerColors[currCell.Faction])
			} else if currCell.Type == gioframework.City {
				gc.SetFillColor(color.RGBA{0x88, 0x88, 0x88, 0xFF})
			} else {
				gc.SetFillColor(color.RGBA{0xFF, 0xFF, 0xFF, 0xFF})
			}
			draw2dkit.Rectangle(gc, x1, y1, x2, y2)
			gc.Fill()

			// Add army count
			if currCell.Armies > 0 {
				numArmiesStr := strconv.Itoa(currCell.Armies)
				left, top, right, bottom := gc.GetStringBounds(numArmiesStr)
				gc.SetFillColor(color.White)
				strX := rowStep/2 - (left+right)/2
				strY := colStep/2 - (top+bottom)/2
				gc.FillStringAt(strconv.Itoa(currCell.Armies), x1+strX, y1+strY)
			}

			// Add Tile type string
			stringToWrite := ""
			switch currCell.Faction {
			case gioframework.FogObstacle:
				stringToWrite = "FO"
			case gioframework.Mountain:
				stringToWrite = "M"
			}
			switch currCell.Type {
			case gioframework.General:
				stringToWrite = "G"
			case gioframework.City:
				stringToWrite = "C"
			}
			gc.SetFillColor(color.White)
			_, top, _, _ := gc.GetStringBounds(stringToWrite)
			gc.FillStringAt(stringToWrite, x1, y1-top)

		}
	}

	// Draw Grid
	gc.SetFillColor(color.Alpha{0})
	gc.SetStrokeColor(color.RGBA{0x88, 0x88, 0x88, 0xFF})
	for i := 0; i < g.Height+2; i++ {
		gc.MoveTo(gridLimitLeft, gridLimitTop+float64(i)*rowStep)
		gc.LineTo(gridLimitRight, gridLimitTop+float64(i)*rowStep)
	}
	for i := 0; i < g.Width+2; i++ {
		gc.MoveTo(gridLimitLeft+float64(i)*colStep, gridLimitTop)
		gc.LineTo(gridLimitLeft+float64(i)*colStep, gridLimitBottom)
	}
	gc.FillStroke()
	gl.Flush()

}

func reshape(window *glfw.Window, w, h int) {
	gl.ClearColor(1, 1, 1, 1)
	gl.Viewport(0, 0, int32(w), int32(h))
	gl.MatrixMode(gl.PROJECTION_MATRIX)
	gl.LoadIdentity()
	gl.Ortho(0, float64(w), 0, float64(h), -1, 1)
	gl.Scalef(1, -1, 1)
	gl.Translatef(0, float32(-h), 0)
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	gl.Disable(gl.DEPTH_TEST)
	width, height = w, h
	//redraw = true
}

func initMapWindow() *glfw.Window {
	err := glfw.Init()
	if err != nil {
		panic(err)
	}

	window, err := glfw.CreateWindow(width, height, "Generals.io", nil, nil)
	if err != nil {
		panic(err)
	}

	window.MakeContextCurrent()
	//	window.SetSizeCallback(reshape)
	window.SetKeyCallback(onKey)
	//	window.SetCharCallback(onChar)
	window.SetMouseButtonCallback(onClick)

	glfw.SwapInterval(1)

	err = gl.Init()
	if err != nil {
		panic(err)
	}

	reshape(window, width, height)

	// Setup font cache
	fontCache := MyFontCache{}

	TTFs := map[string]([]byte){
		"goregular": goregular.TTF,
		"gobold":    gobold.TTF,
		"goitalic":  goitalic.TTF,
		"gomono":    gomono.TTF,
	}

	for fontName, TTF := range TTFs {
		font, err := truetype.Parse(TTF)
		if err != nil {
			panic(err)
		}
		fontCache.Store(draw2d.FontData{Name: fontName}, font)
	}

	draw2d.SetFontCache(fontCache)
	draw2d.SetFontNamer(func(fd draw2d.FontData) string {
		fmt.Println(fd)
		return "gomono"
	})
	return window
}

func main() {
	window := initMapWindow()
	defer glfw.Terminate()
	fmt.Println("Hello World!")
	c, err := gioframework.Connect("us", "testBotBotId", "testBotBotName")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("Connected, starting goroutine")
	go c.Run()
	time.Sleep(1 * time.Second)
	redraw = false
	for {
		game = c.JoinCustomGame("asdf1234")
		game.SetForceStart(true)
		started := false
		game.Start = func(playerIndex int, users []string) {
			fmt.Println("Game started with ", users)
			started = true
		}
		game.Won = func() {
			fmt.Println("Won game!")
			window.SetShouldClose(true)
		}
		game.Lost = func() {
			fmt.Println("Lost game...")
			window.SetShouldClose(true)
		}
		game.Update = func(update gioframework.GameUpdate) {
			redraw = true
		}
		fmt.Println("Waiting for game to start...")
		for !started {
			time.Sleep(1 * time.Second)
		}

		aiCtx := ai.InitContext(game)
		fmt.Println("Context:", aiCtx)

		for !window.ShouldClose() {
			time.Sleep(100 * time.Millisecond)
			if redraw {
				redrawGame(game)
				aiCtx.ProcessGame(game)
				window.SwapBuffers()
				redraw = false
			}
			glfw.PollEvents()
		}
	}

	fmt.Println("Done!")
}
