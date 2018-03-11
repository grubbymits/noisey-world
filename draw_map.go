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
  TREES
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
  _
  _
  GREY_PATH
  MAX_TILE_ROWS
)

const NUM_ROCKS = 8
const NUM_TREES = 4

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
  _
  MAX_TILE_COLUMNS
)

const SHADOW_ROW = GREY_PATH
const (
  RIGHT_SHADOW = 14
  BOTTOM_SHADOW = 15
  TOP_SHADOW = 16
  LEFT_SHADOW = 17
)

// Foliage, treat as continous row.
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
  TREES_END
)

const LIGHT_PINE = TREES_END * 2
const DARK_PINE = LIGHT_PINE + 1
const RIVER_BANK_COLUMN = 10
const GROUND_FEATURE_COLUMN = 18

const TILE_WIDTH = 16
const TILE_HEIGHT = 16

// Ground tiles
var TILE_ROWS = [...] int {
  WATER,        // OCEAN
  WATER,        // RIVER
  SAND,         // BEACH
  SAND,         // DRY_ROCK
  ROCKS,        // WALL
  SOIL,         // MOIST_ROCK
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
  { PLAIN_0, PLAIN_1 }, //, PLAIN_0, PLAIN_1, ROCK_PATCH, SAND_PATCH },
  { PLAIN_0, PLAIN_1 }, //, PLAIN_0, PLAIN_1, ROCK_PATCH, SAND_PATCH },
  { 16, 17 },
  { PLAIN_0, PLAIN_1 }, //, PLAIN_0, PLAIN_1, ROCK_PATCH, SAND_PATCH },
  { PLAIN_0, PLAIN_1 }, //, PLAIN_0, PLAIN_1, DRY_SOIL_PATCH, SAND_PATCH,
  //YELLOW_FLOWERS },
  { PLAIN_0, PLAIN_1 }, //, PLAIN_0, PLAIN_1, ROCK_PATCH, DRY_SOIL_PATCH,
  //YELLOW_FLOWERS },
  { PLAIN_0, PLAIN_1 }, //, PLAIN_0, PLAIN_1, YELLOW_FLOWERS, WHITE_FLOWERS },
  { PLAIN_0, PLAIN_1 }, //, PLAIN_0, PLAIN_1, ROCK_PATCH, WET_SOIL_PATCH,
  //PURPLE_FLOWERS_0, PURPLE_FLOWERS_1 },
  { PLAIN_0, PLAIN_1 }, //, PLAIN_0, PLAIN_1, WET_SOIL_PATCH, WHITE_FLOWERS },
  { PLAIN_0, PLAIN_1 },// , PLAIN_0, PLAIN_1, ROCK_PATCH, DRY_SOIL_PATCH,
  //WET_SOIL_PATCH, WHITE_FLOWERS },
  { PLAIN_0, PLAIN_1 }, //, PLAIN_0, PLAIN_1, WET_SOIL_PATCH, SAND_PATCH,
  //PURPLE_FLOWERS_0, PURPLE_FLOWERS_1 },
}

var BIOME_TREES = [...] []int {
  { },  // OCEAN
  { },  // RIVER
  { },  // BEACH
  { LIGHT_PINE, DARK_PINE },  // DRY_ROCK
  { },  // WALL
  { LIGHT_PINE, DARK_PINE },  // WET_ROCK
  { LIGHT_GREEN_ROUND, LIGHT_GREEN_ROUND_YELLOW, YELLOW_ROUND_0, YELLOW_ROUND_1,
    TREES_END + YELLOW_ROUND_0, TREES_END + YELLOW_ROUND_1 }, // HEATHLAND
  { LIGHT_GREEN_ROUND, TREES_END + LIGHT_GREEN_ROUND,
    TREES_END + DARK_GREEN_ROUND, TREES_END + LIGHT_GREEN_ROUND_YELLOW,
    TREES_END + DARK_GREEN_ROUND_YELLOW },  // SHRUBLAND
  { TREES_END + LIGHT_GREEN_ROUND, TREES_END + DARK_GREEN_ROUND,
    TREES_END + LIGHT_GREEN_ROUND_WHITE, TREES_END + DARK_GREEN_ROUND_WHITE }, // GRASSLAND
  { TREES_END + DARK_PURPLE_ROUND_0, TREES_END + DARK_PURPLE_ROUND_1,
    TREES_END + LIGHT_PURPLE_ROUND_0, TREES_END + LIGHT_PURPLE_ROUND_1 }, // MOORLAND
  { TREES_END + LIGHT_GREEN_ROUND, TREES_END + DARK_GREEN_ROUND },  // FENLAND
  { LIGHT_PINE, DARK_PINE, LIGHT_GREEN_ROUND, DARK_GREEN_ROUND,
    TREES_END + LIGHT_GREEN_ROUND, TREES_END + DARK_GREEN_ROUND },  // WOODLAND
  { LIGHT_PINE, DARK_PINE, LIGHT_GREEN_ROUND, DARK_GREEN_ROUND },   // FOREST
}

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
// WALL - rock
// MOIST_ROCK - sand, sand with soil, sand with grass
// HEATHLAND - dry grass, dry grass with yellow flowers, dry grass with sand
// SHRUBLAND - moist grass, moist grass with moist and wet soil
// GRASSLAND - grass, grass with yellow and white flowers
// MOORLAND - wet grass, wet grass with wet soil, wet grass with purple
// FENLAND - wet grass
// WOODLAND - grass, grass with rock, grass with soils
// FOREST - soil, soil with grass, soil wet grass

type MapRenderer struct {
  mapWidth, mapHeight, tileWidth, tileHeight, tileColumns, tileRows int
  sprites []image.Rectangle
  spritesheet image.Image
  mapImg draw.Image
}

func CreateMapRenderer(width, height, cols, rows int) *MapRenderer {

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

  render := new(MapRenderer)
  render.spritesheet = spritesheet
  render.mapWidth = width
  render.mapHeight = height
  render.tileWidth = TILE_WIDTH
  render.tileHeight = TILE_HEIGHT
  render.tileColumns = cols
  render.tileRows = rows
  render.spritesheet = spritesheet
  render.mapImg = image.NewRGBA(image.Rect(0, 0, width, height))
  render.sprites = make([]image.Rectangle, cols * rows)
  for y := 0; y < rows; y++ {
    for x := 0; x < cols; x++ {
      idx := y * cols + x
      render.sprites[idx] = image.Rect(x * TILE_WIDTH, y * TILE_HEIGHT,
                                       x * TILE_WIDTH + TILE_WIDTH,
                                       y * TILE_HEIGHT + TILE_HEIGHT )
    }
  }
  return render
}

func (renderer *MapRenderer) DrawFeature(x, y, idx int) {
  srcR := renderer.sprites[idx]
  destR := image.Rect(x * TILE_WIDTH, y * TILE_HEIGHT,
                      x * TILE_WIDTH + TILE_WIDTH,
                      y * TILE_HEIGHT + TILE_HEIGHT)
  draw.Draw(renderer.mapImg, destR, renderer.spritesheet, srcR.Min, draw.Over)
}

func (render *MapRenderer) DrawRiverBankFeature(x, y int, feat uint, biome uint8) {
  var offset uint = 0
  switch(feat) {
    case TOP_LEFT_RIVER_FEATURE:
      offset = 0
    case TOP_RIVER_FEATURE:
      offset = 1
    case TOP_RIGHT_RIVER_FEATURE:
      offset = 2
    case LEFT_RIVER_FEATURE:
      offset = 3
    case RIGHT_RIVER_FEATURE:
      offset = 4
    case BOTTOM_LEFT_RIVER_FEATURE:
      offset = 5
    case BOTTOM_RIVER_FEATURE:
      offset = 6
    case BOTTOM_RIGHT_RIVER_FEATURE:
      offset = 7
    default:
    panic("unrecognised river feature")
  }
  offset = feat
  row := TILE_ROWS[biome]
  idx := uint(row * MAX_TILE_COLUMNS + RIVER_BANK_COLUMN) + offset
  render.DrawFeature(x, y, int(idx))
}

func (render *MapRenderer) DrawGroundFeature(x, y int, biome uint8) {
  column := GROUND_FEATURE_COLUMN
  row := TILE_ROWS[biome]
  render.DrawFeature(x, y, row * MAX_TILE_COLUMNS + column)
}

func (render *MapRenderer) DrawFloorTile(x, y int, biome uint8) {
  column := TILE_COLUMNS[biome]
  colIdx := rand.Intn(len(column))
  row := TILE_ROWS[biome]
  srcR := render.sprites[row * MAX_TILE_COLUMNS + column[colIdx]]
  destR := image.Rect(x * TILE_WIDTH, y * TILE_HEIGHT,
                      x * TILE_WIDTH + TILE_WIDTH,
                      y * TILE_HEIGHT + TILE_HEIGHT)
  draw.Draw(render.mapImg, destR, render.spritesheet, srcR.Min, draw.Src)
}

func (render *MapRenderer) ParallelDraw(w *World, xBegin, xEnd int, c chan int) {
  for y := 0; y < w.height; y++ {
    for x := xBegin; x < xEnd; x++ {
      biome := w.Biome(x, y)
      loc := w.Location(x, y)

      if loc.isRiverBank {
        // OCEAN tiles don't have edge tiles, so use BEACH ones.
        if biome == OCEAN {
          biome = BEACH
        }
        render.DrawRiverBankFeature(x, y, loc.riverBank, biome)
        continue;
      }
      if loc.isRiver {
        render.DrawFloorTile(x, y, RIVER)
        continue;
      }

      render.DrawFloorTile(x, y, biome)

      if loc.features == EMPTY {
        continue
      }
      if loc.hasFeature(GROUND_FEATURE) {
        render.DrawGroundFeature(x, y, loc.nearbyBiome)
      }
      if loc.hasFeature(RIGHT_SHADOW_FEATURE) {
        render.DrawFeature(x, y, SHADOW_ROW * MAX_TILE_COLUMNS + RIGHT_SHADOW)
      }
      if loc.hasFeature(BOTTOM_SHADOW_FEATURE) {
        render.DrawFeature(x, y, SHADOW_ROW * MAX_TILE_COLUMNS + BOTTOM_SHADOW)
      } 
      if loc.hasFeature(LEFT_SHADOW_FEATURE) {
        render.DrawFeature(x, y, SHADOW_ROW * MAX_TILE_COLUMNS + LEFT_SHADOW)
      }
      if loc.hasFeature(TOP_SHADOW_FEATURE) {
        render.DrawFeature(x, y, SHADOW_ROW * MAX_TILE_COLUMNS + TOP_SHADOW)
      }
      if loc.hasFeature(ROCK_FEATURE) {
        col := rand.Intn(NUM_ROCKS)
        render.DrawFeature(x, y, ROCKS * MAX_TILE_COLUMNS + col)
      }
      if loc.hasFeature(TREE_FEATURE) {
        trees := BIOME_TREES[biome]
        col := rand.Intn(len(trees))
        render.DrawFeature(x, y, TREES * MAX_TILE_COLUMNS + trees[col])
      }
    }
  }
  c <- 1
}

func DrawMap(w *World, hSeed, mSeed, sSeed, fSeed, rSeed int64, numCPUs int) {
  // First, create an overworld image that represents each tile with a single
  // pixel.
  overworld := image.NewRGBA(image.Rect(0, 0, w.width, w.height))
  colours := [BIOMES]color.RGBA{{ 51, 166, 204, 255 },  // OCEAN
                                { 0, 102, 102, 255 },   // RIVER
                                { 255, 230, 128, 255 }, // BEACH
                                { 204, 204, 204, 255 }, // DRY_ROCK
                                { 204, 204, 204, 255 }, // WALL
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
      if w.hasFeature(x, y, TREE_FEATURE) {
        overworld.Set(x, y, color.RGBA{38, 77, 0, 255})
      } else if w.hasFeature(x, y, ROCK_FEATURE) {
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

  enc := &png.Encoder { CompressionLevel: png.BestSpeed, }

  if err := enc.Encode(imgFile, overworld); err != nil {
    imgFile.Close();
    log.Fatal(err)
  }
  if err := imgFile.Close(); err != nil {
    log.Fatal(err)
  }
  fmt.Println("overworld image created.")

  render := CreateMapRenderer(w.width * TILE_WIDTH, w.height * TILE_HEIGHT,
                              MAX_TILE_COLUMNS, MAX_TILE_ROWS)

  c := make(chan int, numCPUs)
  for i := 0; i < numCPUs; i++ {
    go render.ParallelDraw(w, i * w.width / numCPUs, (i + 1) * w.width / numCPUs, c)
  }
  for i := 0; i < numCPUs; i++ {
    <-c
  }
  fmt.Println("Detailed map rendered in memory.")

  imgFile, err = os.Create("world-map.png")
  if err != nil {
    log.Fatal(err)
  }
  fmt.Println("Encoding detailed map...")
  if err := enc.Encode(imgFile, render.mapImg); err != nil {
    imgFile.Close();
    log.Fatal(err)
  }
  if err := imgFile.Close(); err != nil {
    log.Fatal(err)
  }
  fmt.Println("Done!")

}


