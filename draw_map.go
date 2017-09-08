package main

import (
  "fmt"
  "image"
  "image/color"
  "image/draw"
  "image/png"
  "log"
  "math/rand"
  "os"
  "strconv"
)

// Rows:
// rocks 5
// water 6
// soil 7
// sand 8
// wet grass 9
// moist grass 10
// grass 11
// dry grass 12
const (
  PLANTS = iota
  _
  _
  _
  _
  ROCKS
  WATER
  SOIL
  SAND
  WET_GRASS
  MOIST_GRASS
  GRASS
  DRY_GRASS
  MAX_TILE_COLUMNS
)

const (
  PLAIN_0 = iota
  PLAIN_1
  ROCK_PATCH
  DRY_SOIL_PATCH
  WET_SOIL_PATCH
  SAND_PATCH
  YELLOW_FLOWERS
  WHITE_FLOWERS
  PURPLE_FLOWERS_0
  PURPLE_FLOWERS_1
  _
  _
  _
  _
  _
  _
  _
  _
  MAX_TILE_ROWS
)

const TILE_WIDTH = 16
const TILE_HEIGHT = 16


// floor tiles columns:
// - normal, normal = 0, 1
// - rock = 2
// - dry dirt / grass = 3
// - wet dirt / grass = 4
// - sand = 5
// - yellow flowers = 6
// - white flowers = 7
// - purple flowers = 8, 9
// - top left water = 10
// - top water = 11
// - top right water = 12
// - left water = 13
// - right water = 14
// - bottom left water = 15
// - bottom water = 16
// - bottom right water = 17

// OCEAN - water
// RIVER - water
// BEACH - sand
// DRY_ROCK - sand, sand with rock
// MOIST_ROCK - sand, sand with soil, sand with grass
// HEATHLAND - dry grass, dry grass with yellow flowers, dry grass with sand
// SHRUBLAND - moist grass, moist grass with moist and wet soil
// GRASSLAND - grass, grass with yellow and white flowers
// MOORLAND - wet grass, wet grass with wet soil, wet grass with purple
// FENLAND - wet grass
// WOODLAND - grass, grass with rock, grass with soils
// FOREST - soil, soil with grass, soil wet grass

func DrawMap(w *World, hSeed, mSeed, sSeed, fSeed, rSeed int64) {
  // First, create an overworld image that represents each tile with a single
  // pixel.
  overworld := image.NewRGBA(image.Rect(0, 0, w.width, w.height))
  colours := [BIOMES]color.RGBA{{ 51, 166, 204, 255 },  // OCEAN
                                { 0, 102, 102, 255 },   // RIVER
                                { 255, 230, 128, 255 }, // BEACH
                                { 204, 204, 204, 255 }, // DRY_ROCK
                                { 166, 166, 166, 255 }, // MOIST_ROCK
                                { 202, 218, 114, 255 }, // HEATHLAND
                                { 128, 153, 51, 255 },  // SHRUBLAND
                                { 170, 190, 50, 255 },  // GRASSLAND
                                { 217, 179, 255, 255 }, // MOORLAND
                                { 85, 128, 0, 255 },    // FENLAND
                                { 119, 179, 0, 255 },   // WOODLAND
                                { 77, 153, 0, 255 } }   // FOREST

  bounds := overworld.Bounds()
  for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
    for x := bounds.Min.X; x < bounds.Max.X; x++ {
      if w.Feature(x, y) == TREE_FEATURE {
        overworld.Set(x, y, color.RGBA{38, 77, 0, 255})
      } else if w.Feature(x, y) == ROCK_FEATURE {
        overworld.Set(x, y, color.RGBA{220, 220, 220, 255})
      } else {
        overworld.Set(x, y, colours[w.Biome(x, y)])
      }
    }
  }

  filename := "h" + strconv.FormatInt(hSeed, 16) + "-" +
              "m" + strconv.FormatInt(mSeed, 16) + "-" +
              "s" + strconv.FormatInt(sSeed, 16) + "-" +
              "f" + strconv.FormatInt(fSeed, 16) + "-" +
              "r" + strconv.FormatInt(rSeed, 16) + ".png"
  imgFile, err := os.Create(filename)
  if err != nil {
    log.Fatal(err)
  }
  if err := png.Encode(imgFile, overworld); err != nil {
    imgFile.Close();
    log.Fatal(err)
  }
  if err := imgFile.Close(); err != nil {
    log.Fatal(err)
  }
  fmt.Println("overworld image created.")

  // Now use that pixel data to create the game map constructed from tiled
  // sprites.
  tilesheetFile, err := os.Open("outdoor_tiles.png")
  if err != nil {
    log.Fatal(err)
  }
  spritesheet, err := png.Decode(tilesheetFile)
  if err != nil {
    log.Fatal(err)
  }
  fmt.Println("opened and decoded tilesheet")

  // OCEAN:       x = WATER,        y = []
  // RIVER:       x = WATER,        y = []
  // BEACH:       x = SAND,         y = [ROCK, SAND]
  // DRY_ROCK:    x = SOIL,         y = [ROCK, SAND]
  // MOIST_ROCK:  x = SAND,         y = [ROCK, DRY_SOIL, WET_SOIL, SAND]
  // HEATHLAND:   x = DRY_GRASS,    y = [DRY_SOIL, SAND, YELLOW_FLOWERS]
  // SHRUBLAND:   x = GRASS,        y = [ROCK, DRY_SOIL, YELLOW_FLOWERS]
  // GRASSLAND:   x = GRASS,        y = [YELLOW_FLOWERS, WHITE_FLOWERS]
  // MOORLAND:    x = WET_GRASS,    y = [ROCK, WET_SOIL, PURPLE_FLOWERS_0, PURPLE_FLOWERS_1 ]
  // FENLAND:     x = WET_GRASS,    y = [WET_SOIL, WHITE_FLOWERS]
  // WOODLAND:    x = MOIST_GRASS,  y = [ROCK, DRY_SOIL, WET_SOIL, WHITE_FLOWERS]
  // FOREST:      x = MOIST_GRASS,  y = [WET_SOIL, SAND, PURPLE_FLOWERS_0, PURPLE_FLOWERS_1]  
  sprites := make([]image.Rectangle, MAX_TILE_COLUMNS * MAX_TILE_ROWS)
  for y := 0; y < MAX_TILE_COLUMNS; y++ {
    for x := 0; x < MAX_TILE_ROWS; x++ {
      idx := y * MAX_TILE_ROWS + x
      sprites[idx] = image.Rect(x * TILE_WIDTH, y * TILE_HEIGHT,
                                x * TILE_WIDTH + TILE_WIDTH,
                                y * TILE_HEIGHT + TILE_HEIGHT )
    }
  }

  var TILE_ROWS = [...] int {
    WATER,        // OCEAN
    WATER,        // RIVER
    SAND,         // BEACH
    SOIL,         // DRY_ROCK
    SAND,         // MOIST_ROCK
    DRY_GRASS,    // HEATHLAND
    GRASS,        // SHRUBLAND
    GRASS,        // GRASSLAND
    WET_GRASS,    // MOORLAND
    WET_GRASS,    // FENLAND
    MOIST_GRASS,  // WOODLAND
    MOIST_GRASS,  // FOREST
  }

  var TILE_COLUMNS = [...] []int {
    { PLAIN_0, PLAIN_1 },
    { PLAIN_0, PLAIN_1 },
    { PLAIN_0, PLAIN_1, ROCK_PATCH, SAND_PATCH },
    { PLAIN_0, PLAIN_1, ROCK_PATCH, SAND_PATCH },
    { PLAIN_0, PLAIN_1, ROCK_PATCH, DRY_SOIL_PATCH, WET_SOIL_PATCH, SAND_PATCH },
    { PLAIN_0, PLAIN_1, DRY_SOIL_PATCH, SAND_PATCH, YELLOW_FLOWERS },
    { PLAIN_0, PLAIN_1, ROCK_PATCH, DRY_SOIL_PATCH, YELLOW_FLOWERS },
    { PLAIN_0, PLAIN_1, YELLOW_FLOWERS, WHITE_FLOWERS },
    { PLAIN_0, PLAIN_1, ROCK_PATCH, WET_SOIL_PATCH, PURPLE_FLOWERS_0, PURPLE_FLOWERS_1 },
    { PLAIN_0, PLAIN_1, WET_SOIL_PATCH, WHITE_FLOWERS },
    { PLAIN_0, PLAIN_1, ROCK_PATCH, DRY_SOIL_PATCH, WET_SOIL_PATCH, WHITE_FLOWERS },
    { PLAIN_0, PLAIN_1, WET_SOIL_PATCH, SAND_PATCH, PURPLE_FLOWERS_0, PURPLE_FLOWERS_1 },
  }

  mapWidth := w.width * TILE_WIDTH
  mapHeight := w.height * TILE_HEIGHT
  mapImg := image.NewRGBA(image.Rect(0, 0, mapWidth, mapHeight))
  fmt.Println("mapWidth, mapHeight:", mapWidth, mapHeight)

  bounds = mapImg.Bounds()
  fmt.Println("Draw map, size:", bounds.Max.X, bounds.Max.Y)
  for y := 0; y < w.height; y++ {
    for x := 0; x < w.width; x++ {
      biome := w.Biome(x, y)

      column := TILE_COLUMNS[biome]
      colIdx := rand.Intn(len(column))
      row := TILE_ROWS[biome]
      srcR := sprites[row * MAX_TILE_ROWS + column[colIdx]]
      // Copy tile from spritesheet to mapImg
      destR := image.Rect(x * TILE_WIDTH, y * TILE_HEIGHT,
                          x * TILE_WIDTH + TILE_WIDTH,
                          y * TILE_HEIGHT + TILE_HEIGHT) //dp, dp.Add(sr.Size())}
      draw.Draw(mapImg, destR, spritesheet, srcR.Min, draw.Src)
    }
  }

  imgFile, err = os.Create("world-map.png")
  if err != nil {
    log.Fatal(err)
  }
  if err := png.Encode(imgFile, mapImg); err != nil {
    imgFile.Close();
    log.Fatal(err)
  }
  if err := imgFile.Close(); err != nil {
    log.Fatal(err)
  }

}


