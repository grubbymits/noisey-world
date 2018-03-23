package main

import (
  "fmt"
  "image"
  "image/draw"
  "image/png"
  "log"
  "os"
)

// Floor tile rows
const (
  SOIL = iota
  SAND
  WET_GRASS
  MOIST_GRASS
  GRASS
  DRY_GRASS
  ROCK
  WATER
  MAX_TILE_ROWS
)

// Floor tile columns
const (
  PLAIN_0 = iota
  PLAIN_1
  TOP_LEFT_WATER
  TOP_WATER
  TOP_RIGHT_WATER
  LEFT_WATER
  RIGHT_WATER
  BOTTOM_LEFT_WATER
  BOTTOM_WATER
  BOTTOM_RIGHT_WATER
  WALL_0
  WALL_1
  BLEND
  MAX_TILE_COLUMNS
)

const (
  LEFT_VERTICAL_SHADOW = iota
  HORIZONTAL_SHADOW
  RIGHT_VERTICAL_SHADOW
  BOTTOM_LEFT_SHADOW
  BOTTOM_RIGHT_SHADOW
  NUM_SHADOWS
)

// Foliage
const (
  LIGHT_GREEN_ROUND = iota
  DARK_GREEN_ROUND
  LIGHT_GREEN_ROUND_WHITE
  DARK_GREEN_ROUND_WHITE
  WHITE_ROUND_0
  WHITE_ROUND_1
  LIGHT_GREEN_ROUND_PURPLE
  DARK_GREEN_ROUND_PURPLE
  DARK_PURPLE_ROUND_0
  DARK_PURPLE_ROUND_1
  LIGHT_PURPLE_ROUND_0
  LIGHT_PURPLE_ROUND_1
  LIGHT_GREEN_ROUND_YELLOW
  DARK_GREEN_ROUND_YELLOW
  YELLOW_ROUND_0
  YELLOW_ROUND_1
  ORANGE_ROUND_0
  ORANGE_ROUND_1
  RED_ROUND_0
  RED_ROUND_1
  LIGHT_PINE
  DARK_PINE
  NUM_TREES
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
