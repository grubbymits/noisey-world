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

/*
const LIGHT_PINE = TREES_END * 2
const DARK_PINE = LIGHT_PINE + 1
const RIVER_BANK_COLUMN = 10
*/
const TILE_WIDTH = 16
const TILE_HEIGHT = 16

// Ground tiles row for each biome.
var TILE_ROWS = [...] int {
  WATER,        // OCEAN
  WATER,        // RIVER
  SAND,         // BEACH
  ROCK,         // DRY_ROCK
  SOIL,         // MOIST_ROCK
  DRY_GRASS,    // HEATHLAND
  GRASS,        // SHRUBLAND
  GRASS,        // GRASSLAND
  WET_GRASS,    // MOORLAND
  WET_GRASS,    // FENLAND
  MOIST_GRASS,  // WOODLAND
  MOIST_GRASS,  // FOREST
}

// Columns choices for standard floor tiles for each biome.
var TILE_COLUMNS = [...] []int {
  { PLAIN_0, PLAIN_1 },
  { PLAIN_0, PLAIN_1 },
  { PLAIN_0, PLAIN_1 },
  { PLAIN_0, PLAIN_1 },
  { PLAIN_0, PLAIN_1 },
  { PLAIN_0, PLAIN_1 },
  { PLAIN_0, PLAIN_1 },
  { PLAIN_0, PLAIN_1 },
  { PLAIN_0, PLAIN_1 },
  { PLAIN_0, PLAIN_1 },
  { PLAIN_0, PLAIN_1 },
  { PLAIN_0, PLAIN_1 },
  { PLAIN_0, PLAIN_1 },
}

var BIOME_TREES = [BIOMES] []int {
  { },  // OCEAN
  { },  // RIVER
  { },  // BEACH
  { LIGHT_PINE, DARK_PINE },  // DRY_ROCK
  { LIGHT_PINE, DARK_PINE },  // WET_ROCK
  // HEATHLAND
  { LIGHT_GREEN_ROUND, LIGHT_GREEN_ROUND_YELLOW, YELLOW_ROUND_0, YELLOW_ROUND_1 },
  // SHRUBLAND
  { LIGHT_GREEN_ROUND, DARK_GREEN_ROUND, LIGHT_GREEN_ROUND_YELLOW,
    DARK_GREEN_ROUND_YELLOW },
  // GRASSLAND
  { LIGHT_GREEN_ROUND, DARK_GREEN_ROUND, LIGHT_GREEN_ROUND_WHITE,
    DARK_GREEN_ROUND_WHITE },
  // MOORLAND
  { LIGHT_PINE, DARK_PINE, DARK_PURPLE_ROUND_0, DARK_PURPLE_ROUND_1,
    LIGHT_PURPLE_ROUND_0, LIGHT_PURPLE_ROUND_1 },
  // FENLAND
  { LIGHT_GREEN_ROUND, DARK_GREEN_ROUND },
  // WOODLAND
  { LIGHT_GREEN_ROUND, DARK_GREEN_ROUND, LIGHT_GREEN_ROUND, DARK_GREEN_ROUND },
  // FOREST
  { LIGHT_PINE, DARK_PINE, LIGHT_GREEN_ROUND, DARK_GREEN_ROUND },
}

var BIOME_PLANTS = [BIOMES] []int {
  // OCEAN
  { },
  // RIVER
  { WHITE_LILY, LARGE_LILY, TWO_LILLIES, SMALL_LILY },
  // BEACH
  { },
  // DRY_ROCK
  { SMALL_GRASS },
  // WET_ROCK
  { SMALL_GRASS, LARGE_GRASS },
  // HEATHLAND
  { SMALL_GRASS, LARGE_GRASS, WHITE_FLOWER, YELLOW_FLOWER },
  // SHRUBLAND
  { SMALL_GRASS, LARGE_GRASS, WHITE_FLOWER, PINK_FLOWER, PURPLE_FLOWER },
  // GRASSLAND
  { LARGE_GRASS, WHITE_FLOWER, PINK_FLOWER, PURPLE_FLOWER, BLUE_FLOWER },
  // MOORLAND
  { SMALL_GRASS, LARGE_GRASS, PINK_FLOWER, PURPLE_FLOWER, BLUE_FLOWER, YELLOW_FLOWER },
  // FENLAND
  { SMALL_GRASS, LARGE_GRASS, BLUE_FLOWER, YELLOW_FLOWER },
  // WOODLAND
  { WHITE_FLOWER, YELLOW_FLOWER, MUSHROOM_3, MUSHROOM_4, MUSHROOM_5 },
  // FOREST
  { WHITE_FLOWER, YELLOW_FLOWER, MUSHROOM_0, MUSHROOM_1, MUSHROOM_2, MUSHROOM_3,
    MUSHROOM_4, MUSHROOM_5 },
}

var BIOME_ROCKS = [BIOMES] []int {
  // OCEAN
  { WATER_GREY_0, WATER_GREY_1, WATER_GREY_2 },
  // RIVER
  { WATER_BROWN_0, WATER_BROWN_1, WATER_BROWN_2 },
  // BEACH
  { DRY_BROWN_0, DRY_BROWN_1, DRY_BROWN_2, DRY_GREY_0, DRY_GREY_1, DRY_GREY_2},
  // DRY_ROCK
  { DRY_BROWN_0, DRY_BROWN_1, DRY_BROWN_2, DRY_GREY_0, DRY_GREY_1, DRY_GREY_2},
  // WET_ROCK
  { WET_BROWN_0, WET_BROWN_1, WET_BROWN_2, WET_GREY_0, WET_GREY_1, WET_GREY_2},
  // HEATHLAND
  { DRY_GREY_0, DRY_GREY_1, DRY_GREY_2},
  // SHRUBLAND
  { DRY_BROWN_0, DRY_BROWN_1, DRY_BROWN_2},
  // GRASSLAND
  { DRY_BROWN_0, DRY_BROWN_1, DRY_BROWN_2, WET_GREY_0, WET_GREY_1, WET_GREY_2},
  // MOORLAND
  { WET_BROWN_0, WET_BROWN_1, WET_BROWN_2, WET_GREY_0, WET_GREY_1, WET_GREY_2},
  // FENLAND
  { WET_BROWN_0, WET_BROWN_1, WET_BROWN_2, WET_GREY_0, WET_GREY_1, WET_GREY_2},
  // WOODLAND
  { WET_BROWN_0, WET_BROWN_1, WET_BROWN_2, WET_GREY_0, WET_GREY_1, WET_GREY_2},
  // FOREST
  { WET_BROWN_0, WET_BROWN_1, WET_BROWN_2, WET_GREY_0, WET_GREY_1, WET_GREY_2},
}

type MapRenderer struct {
  mapWidth, mapHeight, tileWidth, tileHeight, tileColumns, tileRows int
  floorSheet *SpriteSheet
  shadowSheet *SpriteSheet
  treeSheet *SpriteSheet
  plantSheet *SpriteSheet
  rockSheet *SpriteSheet
  mapImg draw.Image
}

func CreateMapRenderer(width, height int) *MapRenderer {
  render := new(MapRenderer)
  render.mapWidth = width
  render.mapHeight = height
  render.mapImg = image.NewRGBA(image.Rect(0, 0, width, height))
  render.floorSheet = CreateSheet("outdoor_floor_tiles.png", MAX_TILE_COLUMNS,
                                  MAX_TILE_ROWS)
  render.shadowSheet = CreateSheet("shadows.png", NUM_SHADOWS, 1)
  render.treeSheet = CreateSheet("trees.png", NUM_TREES, 2)
  render.rockSheet = CreateSheet("rocks.png", NUM_ROCKS, 1)
  render.plantSheet = CreateSheet("plants.png", NUM_PLANTS, 1)
  return render
}

func (render *MapRenderer) DrawRiverBankFeature(x, y int, feat uint, biome uint8) {
  var col int = 0
  switch(feat) {
    case TOP_LEFT_RIVER_FEATURE:
      col = TOP_LEFT_WATER
    case TOP_RIVER_FEATURE:
      col = TOP_WATER
    case TOP_RIGHT_RIVER_FEATURE:
      col = TOP_RIGHT_WATER
    case LEFT_RIVER_FEATURE:
      col = LEFT_WATER
    case RIGHT_RIVER_FEATURE:
      col = RIGHT_WATER
    case BOTTOM_LEFT_RIVER_FEATURE:
      col = BOTTOM_LEFT_WATER
    case BOTTOM_RIVER_FEATURE:
      col = BOTTOM_WATER
    case BOTTOM_RIGHT_RIVER_FEATURE:
      col = BOTTOM_RIGHT_WATER
    default:
    panic("unrecognised river feature")
  }
  //offset = feat
  row := TILE_ROWS[biome]
  idx := row * MAX_TILE_COLUMNS + col
  render.floorSheet.DrawFeature(x, y, idx, render.mapImg)
}

func (render *MapRenderer) DrawFeatures(loc *Location, biome uint8, x, y int) {
  if loc.isWall {
    row := TILE_ROWS[biome]
    walls := [2]int { WALL_0, WALL_1 }
    colIdx := rand.Intn(len(walls))
    col := walls[colIdx]
    render.floorSheet.DrawFeature(x, y, row * MAX_TILE_COLUMNS + col,
                                  render.mapImg)
    return
  }

  if loc.hasFeature(GROUND_FEATURE) {
    col := BLEND
    row := TILE_ROWS[loc.nearbyBiome]
    render.floorSheet.DrawFeature(x, y, row * MAX_TILE_COLUMNS + col,
                                  render.mapImg)
  }

  if biome != RIVER {
  if loc.hasFeature(RIGHT_SHADOW_FEATURE) {
    render.shadowSheet.DrawFeature(x, y, RIGHT_VERTICAL_SHADOW, render.mapImg)
  }
  if loc.hasFeature(LEFT_SHADOW_FEATURE) {
    render.shadowSheet.DrawFeature(x, y, LEFT_VERTICAL_SHADOW, render.mapImg)
  }
  if loc.hasFeature(HORIZONTAL_SHADOW_FEATURE) {
    render.shadowSheet.DrawFeature(x, y, HORIZONTAL_SHADOW, render.mapImg)
  }
  if loc.hasFeature(BOTTOM_LEFT_SHADOW_FEATURE) {
    render.shadowSheet.DrawFeature(x, y, BOTTOM_LEFT_SHADOW, render.mapImg)
  }
  if loc.hasFeature(BOTTOM_RIGHT_SHADOW_FEATURE) {
    render.shadowSheet.DrawFeature(x, y, BOTTOM_RIGHT_SHADOW, render.mapImg)
  }
  } else {
    if loc.hasFeature(LEFT_WATER_SHADOW_FEATURE) {
      render.shadowSheet.DrawFeature(x, y, LEFT_VERTICAL_WATER_SHADOW, render.mapImg)
    }
    if loc.hasFeature(RIGHT_WATER_SHADOW_FEATURE) {
      render.shadowSheet.DrawFeature(x, y, RIGHT_VERTICAL_WATER_SHADOW, render.mapImg)
    }
  }
  if loc.hasFeature(TREE_FEATURE) {
    trees := BIOME_TREES[biome]
    if len(trees) != 0 {
      col := rand.Intn(len(trees))
      rows := [2]int { 0, 1 }
      rowIdx := rand.Intn(len(rows))
      row := rows[rowIdx]
      render.treeSheet.DrawFeature(x, y, row * NUM_TREES + trees[col],
                                 render.mapImg)
    }
  }
  if loc.hasFeature(ROCK_FEATURE) {
    rocks := BIOME_ROCKS[biome]
    idx := rand.Intn(len(rocks))
    rock := rocks[idx]
    render.rockSheet.DrawFeature(x, y, rock, render.mapImg)
  }
  if loc.hasFeature(PLANT_FEATURE) {
    plants := BIOME_PLANTS[biome]
    if len(plants) != 0 {
      idx := rand.Intn(len(plants))
      plant := plants[idx]
      render.plantSheet.DrawFeature(x, y, plant, render.mapImg)
    }
  }
}

func (render *MapRenderer) DrawFloorTile(x, y int, biome uint8) {
  column := TILE_COLUMNS[biome]
  colIdx := rand.Intn(len(column))
  row := TILE_ROWS[biome]
  idx := row * MAX_TILE_COLUMNS + column[colIdx]
  render.floorSheet.DrawFloorTile(x, y, idx, render.mapImg)
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
        if loc.isWall {
          biome = RIVER
        }
        render.DrawRiverBankFeature(x, y, loc.riverBank, biome)
        //render.DrawFeatures(loc, biome, x, y)
      } else if loc.isRiver {
        render.DrawFloorTile(x, y, RIVER)
        render.DrawFeatures(loc, RIVER, x, y)
      } else {
        render.DrawFloorTile(x, y, biome)
        render.DrawFeatures(loc, loc.biome, x, y)
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

  render := CreateMapRenderer(w.width * TILE_WIDTH, w.height * TILE_HEIGHT)

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


