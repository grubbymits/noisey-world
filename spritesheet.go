package main

import (
  "fmt"
  "image"
  "image/draw"
  "image/png"
  "log"
  "os"
)

type SpriteSheet struct {
  tileWidth, tileHeight, tileColumns, tileRows int
  spritesheet image.Image
  sprites []image.Rectangle
}

func CreateSheet(filename string, cols, rows int) *SpriteSheet {
  tilesheetFile, err := os.Open(filename)
  if err != nil {
    log.Fatal(err)
  }

  spritesheet, err := png.Decode(tilesheetFile)
  if err != nil {
    log.Fatal(err)
  }
  fmt.Println("opened and decoded tilesheet")

  sheet := new(SpriteSheet)
  sheet.tileWidth = TILE_WIDTH
  sheet.tileHeight = TILE_HEIGHT
  sheet.tileColumns = cols
  sheet.tileRows = rows
  sheet.spritesheet = spritesheet
  sheet.sprites = make([]image.Rectangle, cols * rows)
  for y := 0; y < rows; y++ {
    for x := 0; x < cols; x++ {
      idx := y * cols + x
      sheet.sprites[idx] = image.Rect(x * TILE_WIDTH, y * TILE_HEIGHT,
                                      x * TILE_WIDTH + TILE_WIDTH,
                                      y * TILE_HEIGHT + TILE_HEIGHT )
    }
  }
  return sheet
}

func (sheet *SpriteSheet) DrawFeature(x, y, idx int, img draw.Image) {
  srcR := sheet.sprites[idx]
  width := sheet.tileWidth
  height := sheet.tileHeight
  destR := image.Rect(x * width, y * height,
                      x * width + width,
                      y * height + height)
  draw.Draw(img, destR, sheet.spritesheet, srcR.Min, draw.Over)
}

func (sheet *SpriteSheet) DrawFloorTile(x, y, idx int, img draw.Image) {
  srcR := sheet.sprites[idx]
  destR := image.Rect(x * TILE_WIDTH, y * TILE_HEIGHT,
                      x * TILE_WIDTH + TILE_WIDTH,
                      y * TILE_HEIGHT + TILE_HEIGHT)
  draw.Draw(img, destR, sheet.spritesheet, srcR.Min, draw.Src)
}
