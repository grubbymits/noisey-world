package main

import (
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
  LEFT_VERTICAL_WATER_SHADOW
  RIGHT_VERTICAL_WATER_SHADOW
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

const (
  SMALL_GRASS = iota
  LARGE_GRASS
  BLUE_FLOWER
  RED_FLOWER
  PINK_FLOWER
  PURPLE_FLOWER
  WHITE_FLOWER
  YELLOW_FLOWER
  MUSHROOM_0
  MUSHROOM_1
  MUSHROOM_2
  MUSHROOM_3
  MUSHROOM_4
  MUSHROOM_5
  WHITE_LILY
  LARGE_LILY
  TWO_LILLIES
  SMALL_LILY
  NUM_PLANTS
)

const (
  DRY_SMALL_GREY_0 = iota
  DRY_SMALL_GREY_1
  DRY_SMALL_GREY_2
  DRY_MEDIUM_GREY_0
  DRY_MEDIUM_GREY_1
  DRY_MEDIUM_GREY_2
  DRY_LARGE_GREY_0
  DRY_LARGE_GREY_1
  DRY_LARGE_GREY_2
  WET_SMALL_GREY_0
  WET_SMALL_GREY_1
  WET_SMALL_GREY_2
  WET_MEDIUM_GREY_0
  WET_MEDIUM_GREY_1
  WET_MEDIUM_GREY_2
  WET_LARGE_GREY_0
  WET_LARGE_GREY_1
  WET_LARGE_GREY_2
  WATER_BROWN_0
  WATER_BROWN_1
  WATER_BROWN_2
  WATER_GREY_0
  WATER_GREY_1
  WATER_GREY_2
  NUM_ROCKS
)

const (
  PATH_0 = iota
  _
  _
  _
  _
  _
  _
  _
  _
  _
  _
  _
  _
  _
  _
  NUM_PATHS
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
