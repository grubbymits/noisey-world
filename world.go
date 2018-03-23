package main

import (
  "container/heap"
  "flag"
  "fmt"
  "math"
  "math/rand"
  "runtime"
  "time"
)

import "github.com/ojrac/opensimplex-go"

const REGION_SIZE = 64
const REGION_AREA = REGION_SIZE * REGION_SIZE
var TREE_DENSITY = [...]int {
  0,  // OCEAN
  0,  // RIVER
  REGION_SIZE / 1024,  // BEACH
  REGION_AREA / 512,  // DRY_ROCK
  REGION_AREA / 256,  // MOIST_ROCK
  REGION_AREA / 64,   // HEATHLAND
  REGION_AREA / 32,   // SHRUBLAND
  REGION_AREA / 96,  // GRASSLAND
  REGION_AREA / 128,  // MOORLAND
  REGION_AREA / 128,  // FENLAND
  REGION_AREA / 16,   // WOODLAND
  REGION_AREA / 16,   // FOREST
}

var ROCK_DENSITY = [...]int {
  REGION_SIZE / 32,   // OCEAN
  REGION_SIZE / 32,   // RIVER
  REGION_SIZE / 32,   // BEACH
  REGION_AREA / 16,   // DRY_ROCK
  REGION_AREA / 16,   // MOIST_ROCK
  REGION_AREA / 1024, // HEATHLAND
  REGION_AREA / 512,  // SHRUBLAND
  REGION_AREA / 256,  // GRASSLAND
  REGION_AREA / 256,  // MOORLAND
  REGION_AREA / 1024, // FENLAND
  REGION_AREA / 512,  // WOODLAND
  REGION_AREA / 512,  // FOREST
}

const WATER_LEVEL = -0.35
const BEACH_LEVEL = WATER_LEVEL + 0.05
const LOWLANDS = BEACH_LEVEL + 0.3
const MIDLANDS = LOWLANDS + 0.3
const HIGHLANDS = MIDLANDS + 0.3
const WATER_SATURATION = 100
const NO_SOIL = -1.5
const DRY = -0.5
const MOIST = 0
const WET = 0.3
const THICK_SOIL = -0.2
const SHALLOW_SOIL = -0.7

type World struct {
  width, height int
  locations []Location
  regions []Location
  peaks map[*Location]bool
  lakes map[*Location]bool
  hFreq, mFreq, sFreq, fFreq, rFreq, water float64
}

func CreateWorld(width, height int, hFreq, mFreq, sFreq, fFreq, rFreq,
                 water float64) *World {
  w := new(World)
  w.width = width;
  w.height = height;
  w.locations = make([]Location, width * height)
  w.regions = make([]Location, width * height / REGION_AREA)
  w.peaks = make(map[*Location]bool)
  w.lakes = make(map[*Location]bool)
  w.hFreq = hFreq
  w.mFreq = mFreq
  w.sFreq = sFreq
  w.fFreq = fFreq
  w.rFreq = rFreq
  w.water = water

  for y := 0; y < height; y++ {
    for x := 0; x < width; x++ {
      loc := w.Location(x, y)
      loc.x = x
      loc.y = y
    }
  }
  return w
}

func (w World) addPeak(l *Location) {
  w.peaks[l] = true
}

func (w World) addLake(l *Location) {
  w.lakes[l] = true
}

func (w World) addFeature(x, y int, feature uint) {
  w.locations[y * w.width + x].features |= feature
}

func (w World) hasFeature(x, y int, feat uint) bool {
  return w.locations[y * w.width + x].hasFeature(feat)
}

func (w World) Location(x, y int) *Location {
  return &w.locations[y * w.width + x]
}

func (w World) Region(x, y int) *Location {
  rx := x % REGION_SIZE
  ry := y % REGION_SIZE
  return &w.regions[ry * w.width / REGION_SIZE + rx]
}

func (w World) Moisture(x, y int) float64 {
  return w.locations[y * w.width + x].moisture
}

func (w World) Height(x, y int) float64 {
  return w.locations[y * w.width + x].height
}

func (w World) SoilDepth(x, y int) float64 {
  return w.locations[y * w.width + x].soilDepth
}

func (w World) Foliage(x, y int) float64 {
  return w.locations[y * w.width + x].foliage
}

func (w World) Rock(x, y int) float64 {
  return w.locations[y * w.width + x].rock
}

func (w World) Terrace(x, y int) uint8 {
  return w.locations[y * w.width + x].terrace
}

func (w World) Biome(x, y int) uint8 {
  return w.locations[y * w.width + x].biome
}

func (w World) SetMoisture(x, y int, m float64) {
  w.locations[y * w.width + x].moisture = m
}

func (w World) SetSoilDepth(x, y int, s float64) {
  w.locations[y * w.width + x].soilDepth = s
}

func (w World) SetFoliage(x, y int, f float64) {
  w.locations[y * w.width + x].foliage = f
}

func (w World) SetRock(x, y int, r float64) {
  w.locations[y * w.width + x].rock = r
}

func (w World) SetHeight(x, y int, h float64) {
  w.locations[y * w.width + x].height = h
}

func (w World) SetTerrace(x, y int, t uint8) {
  w.locations[y * w.width + x].terrace = t
}

func (w World) SetBiome(x, y int, b uint8) {
  w.locations[y * w.width + x].biome = b
}

func (w World) isRiverValid(centre *Location) bool {
  if centre.biome == OCEAN {
    return false
  }
  if centre.x <= 0 || centre.y <= 0 ||
     centre.x + 1 >= w.width || centre.y + 1 >= w.height {
    return false
  }

  for x := -1; x < 2; x++ {
    for y := -1; y < 2; y++ {
      loc := w.Location(centre.x + x, centre.y + y)

      if loc.x + 1 < w.width  && loc.y - 1 >= 0 {
        east := w.Location(loc.x + 1, loc.y)
        NE := w.Location(loc.x + 1, loc.y - 1)
        if loc.terrace < east.terrace && !NE.isRiver {
          return false
        }
      }

      if loc.x + 1 < w.width && loc.y - 1 >= 0 && loc.x - 1 >= 0 {
        east := w.Location(loc.x + 1, loc.y)
        NW := w.Location(loc.x - 1, loc.y - 1)
        if loc.terrace > east.terrace && !NW.isRiver {
          return false
        }
      }

      if loc.x - 1 >= 0 && loc.y - 1 >= 0 {
        west := w.Location(loc.x - 1, loc.y)
        NW := w.Location(loc.x - 1, loc.y - 1)
        if loc.terrace < west.terrace && !NW.isRiver {
          return false
        }
      }

      if loc.x - 1 >= 0 && loc.y - 1 >= 0 && loc.x + 1 < w.width {
        west := w.Location(loc.x - 1, loc.y)
        NE := w.Location(loc.x + 1, loc.y - 1)
        if loc.terrace > west.terrace && !NE.isRiver {
          return false
        }
      }
    }
  }
  return true
}

func (w World) AddWater(loc *Location) {
  loc.water += loc.moisture
  loc.water -= loc.soilDepth
  if loc.water > WATER_SATURATION && loc.biome != OCEAN {
    loc.isRiver = true
    west := w.Location(loc.x - 1, loc.y)
    east := w.Location(loc.x + 1, loc.y)
    if west.terrace > loc.terrace {
      loc.addFeature(RIGHT_WATER_SHADOW_FEATURE)
    }
    if east.terrace > loc.terrace {
      loc.addFeature(LEFT_WATER_SHADOW_FEATURE)
    }
    // Square up the water so that a body of water is a minimum of 3x3 tiles.
    // This allows for a puddle of water to be surrounded in suitable tiles.
    if loc.x > 1 && loc.y > 1 && loc.x + 1 < w.width && loc.y + 1 < w.height {
      for x := -1; x < 2; x++ {
        for y := -1; y < 2; y++ {
          adjLoc := w.Location(loc.x + x, loc.y + y)
          adjLoc.isRiver = true
        }
      }
    }
  }
}

func (w World) AddRivers() {
  queue := make([]*Location, len(w.peaks))
  i := 0
  totalWater := 0.0

  for loc, _ := range w.peaks {
    loc.water = w.water * loc.moisture * loc.height
    totalWater += loc.water
    queue[i] = loc
    i++
  }
  fmt.Println("Total water added:", totalWater)

  // Iterate through all queue, which will hold every location eventually.
  for n := 0; n < w.height * w.width; n++ {
    loc := queue[0]
    queue = queue[1:]

    // Receive water from all of the predecessors. 
    for i := 0; i < loc.numPreds; i++ {

      pred := loc.preds[i]

      if pred.water < 0 {
        continue
      }
      if pred.y > loc.y {
        continue
      }

      gradientRatio := 0.0
      if pred.terrace > loc.terrace {
        gradientRatio = 1.0;
      } else {
        // Calculate percentage of predecessor's water the successor will
        // receive.
        gradientRatio = (math.Abs(pred.height) - math.Abs(loc.height)) / pred.totalGradient
      }
      water := gradientRatio * pred.water
      loc.water += water
    }

    if w.isRiverValid(loc) {
      w.AddWater(loc)
    } else {
      if loc.y + 1 < w.height {
        south := w.Location(loc.x, loc.y + 1)
        if w.isRiverValid(south) {
          w.AddWater(south)
        }
      }
    }

    // Discover all of the Location's successors and add them to the queue if
    // they've been discovered by all their predecessors.
    for i := 0; i < loc.numSuccs; i++ {
      succ := loc.succs[i]
      succ.discovered++
      if succ.discovered == succ.numPreds {
        queue = append(queue, succ)
      }
    }
  }
}

// Look around each tile, recording the number of tiles which differ from its
// biome. The most often occuring differing biome can be used to overlay a
// tile as a feature. Skip OCEAN, RIVER and WALL tiles as positions to begin
// the search. Also dismiss OCEAN and RIVER tiles during the search and don't
// try to diffuse biome that are in different terraces.
func (w World) AddGroundFeature(xBegin, xEnd int, c chan int) {
  /*
  OCEAN
  RIVER
  BEACH
  DRY_ROCK
  MOIST_ROCK
  HEATHLAND
  SHRUBLAND
  GRASSLAND
  MOORLAND
  FENLAND
  WOODLAND
  FOREST
  */
  for y := 0; y < w.height; y++ {
    for x := xBegin; x < xEnd; x++ {
      loc := w.Location(x, y);
      if loc.isRiver || loc.isRiverBank || loc.isWall  || loc.biome == OCEAN ||
        loc.biome == BEACH {
        continue
      }

      var biomes [BIOMES]uint
      for dx := -1; dx < 2; dx++ {
        for dy := -1; dy < 2; dy++ {
          if dx == 0 && dy == 0 {
            continue
          }

          if x + dx >= w.width || x + dx < 0 {
            continue
          }
          if y + dy >= w.height || y + dy < 0 {
            continue
          }

          otherLoc := w.Location(x + dx, y + dy)
          if otherLoc.isRiver || otherLoc.biome == OCEAN ||
            otherLoc.terrace != loc.terrace || otherLoc.biome == loc.biome {
            continue
          }
          biomes[otherLoc.biome]++
        }
      }
      var largest uint = 0
      var biome uint8 = 0
      for i, val := range biomes {
        if val > largest {
          largest = val
          biome = uint8(i)
        }
      }
      if biome != 0 && biome != loc.biome {
        loc.addFeature(GROUND_FEATURE)
        loc.nearbyBiome = biome
      }
    }
  }
  c <- 1
}

func (w World) AddRiverBanks(xBegin, xEnd int, c chan int) {
  
  for y := 0; y < w.height; y ++ {
    for x := xBegin; x < xEnd; x ++ {
      loc := w.Location(x, y);
      if !loc.isRiver && loc.biome != OCEAN {
        continue;
      }
      N := false
      E := false
      S := false
      W := false

      if y > 0 {
        biome := w.Location(x, y - 1).biome
        N = !w.Location(x, y - 1).isRiver && biome != OCEAN 
      }
      if y < w.height - 1 {
        biome := w.Location(x, y + 1).biome
        S = !w.Location(x, y + 1).isRiver && biome != OCEAN
      }
      if x > 0 {
        biome := w.Location(x - 1, y).biome
        W = !w.Location(x - 1, y).isRiver && biome != OCEAN
      }
      if x < w.width - 1 {
        biome := w.Location(x + 1, y).biome
        E = !w.Location(x + 1, y).isRiver && biome != OCEAN
      }

      // Location == land:
      // N and W = TL
      // N and E = TR
      // N = T
      // S and W = BL
      // S and E = BR
      // S = B
      // E = R
      // W = L
      if !N && !S && !E && !W {
        continue;
      } else if N && W {
        loc.setRiverBank(TOP_LEFT_RIVER_FEATURE)
      } else if N && E {
        loc.setRiverBank(TOP_RIGHT_RIVER_FEATURE)
      } else if N {
        loc.setRiverBank(TOP_RIVER_FEATURE)
      } else if S && W {
        loc.setRiverBank(BOTTOM_LEFT_RIVER_FEATURE)
      } else if S && E {
        loc.setRiverBank(BOTTOM_RIGHT_RIVER_FEATURE)
      } else if S {
        loc.setRiverBank(BOTTOM_RIVER_FEATURE)
      } else if E {
        loc.setRiverBank(RIGHT_RIVER_FEATURE)
      } else if W {
        loc.setRiverBank(LEFT_RIVER_FEATURE)
      }
    }
  }
  c <-1 
}

func (w World) AnalyseRegions(xBegin, xEnd int, c chan int) {
  // Divide the world into regions and calculate attributes of each region.
  // If a loc is a 16x16 tile, a region could be 64x64 tiles.
  // - ratio of ocean
  // - ratio of beach
  // - ratio of rivers
  // - average height
  // - most often occurring biome
  // - number of trees
  // - number of rocks
  // For each region, use the most often occuring biome to choose the number of
  // trees for that region. Iterate through the locations put them into a max
  // heap. Once all the locations have been visited, sort the heap and pop off
  // the required number of locations for each tree.
  for y := 0; y < w.height; y += REGION_SIZE {
    for x := xBegin; x < xEnd; x += REGION_SIZE {
      biomeCount := [BIOMES]int{0}
      treeHeap := make(LocMaxHeap, REGION_AREA)
      rockHeap := make(LocMaxHeap, REGION_AREA)

      i := 0
      for ry := y; ry < y + REGION_SIZE; ry++ {
        for rx := x; rx < x + REGION_SIZE; rx++ {
          foliage := 0.0
          rock := 0.0
          biome := w.Biome(rx, ry)
          biomeCount[biome]++
          if biome != OCEAN && biome != BEACH &&
             !w.Location(rx, ry).isRiver && !w.Location(rx, y).isWall {
            foliage = w.Foliage(rx, ry)
          }
          if !w.Location(rx, ry).isWall {
            rock = w.Rock(rx, ry)
          }
          treeHeap[i] = &LocVal{ i, rx, ry, foliage }
          rockHeap[i] = &LocVal{ i, rx, ry, rock }
          i++
        }
      }
      heap.Init(&treeHeap)
      heap.Init(&rockHeap)

      maxCount := 0
      maxBiome := 0
      region := w.Region(x, y)
      for i := 0; i < len(biomeCount); i++ {
        if biomeCount[i] > maxCount {
          maxBiome = i
          maxCount = biomeCount[i]
        }
      }
      region.biome = uint8(maxBiome)

      for i := 0; i < TREE_DENSITY[maxBiome]; i++ {
        locVal := heap.Pop(&treeHeap).(*LocVal)
        w.addFeature(locVal.x, locVal.y, TREE_FEATURE);
      }
      for i := 0; i < ROCK_DENSITY[maxBiome]; i++ {
        locVal := heap.Pop(&rockHeap).(*LocVal)
        w.addFeature(locVal.x, locVal.y, ROCK_FEATURE);
      }
    }
  }
  c <- 1
}

func (w World) CalcGradient(c chan int) {
  width := w.width
  height := w.height

  for y := 0; y < height; y++ {
    for x := 0; x < width; x++ {

      centreLoc := w.Location(x, y)
      hasPredecessor := false
      hasSuccessor := false

      if y - 1 >= 0 {
        north := w.Location(x, y - 1)
        if north.height <= centreLoc.height {
          centreLoc.addSuccessor(north)
          hasSuccessor = true
          if north.terrace < centreLoc.terrace {
            centreLoc.addFeature(HORIZONTAL_SHADOW_FEATURE)
          }
        } else if north.height >= centreLoc.height {
          centreLoc.addPredecessor(north)
          hasPredecessor = true
          if north.terrace > centreLoc.terrace {
            centreLoc.addFeature(HORIZONTAL_SHADOW_FEATURE)
          }
        }
      }

      if y + 1 < height {
        south := w.Location(x, y + 1)
        if south.height <= centreLoc.height {
          centreLoc.addSuccessor(south)
          hasSuccessor = true
        } else if south.height >= centreLoc.height {
          centreLoc.addPredecessor(south)
          hasPredecessor = true
        }
      }

      if x + 1 < width {
        east := w.Location(x + 1, y)
        if east.height <= centreLoc.height {
          centreLoc.addSuccessor(east)
          hasSuccessor = true
        } else if east.height >= centreLoc.height {
          centreLoc.addPredecessor(east)
          hasPredecessor = true
          if east.terrace > centreLoc.terrace {
            centreLoc.addFeature(LEFT_SHADOW_FEATURE)
          }
        }
      }

      if x - 1 >= 0 {
        west := w.Location(x - 1, y)
        if west.height <= centreLoc.height {
          centreLoc.addSuccessor(west)
          hasSuccessor = true
        } else if west.height >= centreLoc.height {
          centreLoc.addPredecessor(west)
          hasPredecessor = true
          if west.terrace > centreLoc.terrace {
            centreLoc.addFeature(RIGHT_SHADOW_FEATURE)
          }
        }
      }

      if x + 1 < height && y - 1 >= 0 {
        topRight := w.Location(x + 1, y - 1)
        top := w.Location(x, y - 1)
        right := w.Location(x + 1, y)
        if topRight.terrace > centreLoc.terrace &&
           top.terrace == centreLoc.terrace &&
           right.terrace == centreLoc.terrace {
          centreLoc.addFeature(BOTTOM_LEFT_SHADOW_FEATURE)
        }
      }

      if x - 1 >= 0 && y - 1 >= 0 {
        topLeft := w.Location(x - 1, y - 1)
        top := w.Location(x, y - 1)
        left := w.Location(x - 1, y)
        if topLeft.terrace > centreLoc.terrace &&
           top.terrace == centreLoc.terrace &&
           left.terrace == centreLoc.terrace {
          centreLoc.addFeature(BOTTOM_RIGHT_SHADOW_FEATURE)
        }
      }

      if !hasPredecessor {
        w.addPeak(w.Location(x, y))
      } else if !hasSuccessor {
        w.addLake(w.Location(x, y))
      }

      // Calculate the total gradient of the successors, which will be used to
      // determine the ratio of water flow. 
      for i := 0; i < centreLoc.numSuccs; i++ {
        succ := centreLoc.succs[i]
        centreLoc.totalGradient += math.Abs(centreLoc.height) - math.Abs(succ.height)
      }
    }
  }
  c <- 1
}

func (world World) CalcHeight(xBegin, xEnd int,
                                 noise *opensimplex.Noise, c chan int) {
  freq := world.hFreq
  width := world.width
  height := world.height
  heightBias := float64(3 / height)

  for y := 0; y < height; y++ {
    for x := xBegin; x < xEnd; x++ {
      xFloat := float64(x) / float64(width)
      yFloat := float64(y) / float64(height)
      h := 1 * noise.Eval2(freq * xFloat, freq * yFloat) +
             0.50 * noise.Eval2(2 * freq * xFloat, 2 * freq * yFloat) +
             0.25 * noise.Eval2(4 * freq * xFloat, 4 * freq * yFloat) +
             0.125 * noise.Eval2(8 * freq * xFloat, 8 * freq * yFloat) +
             heightBias * float64(height - y)

      //h  = math.Pow(h, float64(1 + (y / height)))
      if h > HIGHLANDS {
        world.SetTerrace(x, y, 4)
      } else if h > MIDLANDS {
        world.SetTerrace(x, y, 3)
      } else if h > LOWLANDS {
        world.SetTerrace(x, y, 2)
      } else if h > BEACH {
        world.SetTerrace(x, y, 1)
      } else {
        world.SetTerrace(x, y, 0)
      }
      world.SetHeight(x, y, h)
    }
  }
  c <- 1
}

func (world World) CalcSoilDepth(xBegin, xEnd int,
                                 noise *opensimplex.Noise, c chan int) {
  freq := world.sFreq
  width := world.width
  height := world.height

  for y := 0; y < height; y++ {
    for x := xBegin; x < xEnd; x++ {
      xFloat := float64(x) / float64(width)
      yFloat := float64(y) / float64(height)
      s := 1 * noise.Eval2(freq * xFloat, freq * yFloat) +
             0.50 * noise.Eval2(2 * freq * xFloat, 2 * freq * yFloat) +
             0.25 * noise.Eval2(4 * freq * xFloat, 4 * freq * yFloat) +
             0.125 * noise.Eval2(8 * freq * xFloat, 8 * freq * yFloat)

      s -= world.Height(x, y)
      world.SetSoilDepth(x, y, s)
    }
  }
  c <- 1
}

func (w World) CalcMoisture(xBegin, xEnd int,
                                noise *opensimplex.Noise,
                                c chan int) {
  freq := w.mFreq
  width := w.width
  height := w.height

  for y := 0; y < height; y++ {
    for x := xBegin; x < xEnd; x++ {
      xFloat := float64(x) / float64(width)
      yFloat := float64(y) / float64(height)
      m := 1.2 * noise.Eval2(freq * xFloat, freq * yFloat) +
             0.60 * noise.Eval2(2 * freq * xFloat, 2 * freq * yFloat) +
             0.3 * noise.Eval2(4 * freq * xFloat, 4 * freq * yFloat) +
             0.15 * noise.Eval2(8 * freq * xFloat, 8 * freq * yFloat)

      w.SetMoisture(x, y, m)
    }
  }
  c <- 1
}

func (w World) CalcBiome(xBegin, xEnd int, c chan int) {
  height := w.height

  for x := xBegin; x < xEnd; x++ {
    w.SetBiome(x, 0, biome(w.Height(x, 0), w.Moisture(x, 0),
               w.SoilDepth(x, 0)))
  }

  for y := 1; y < height; y++ {
    for x := xBegin; x < xEnd; x++ {
      if w.Terrace(x, y - 1) > w.Terrace(x, y) {
        w.Location(x, y - 1).isWall = true;
      }
      w.SetBiome(x, y, biome(w.Height(x, y), w.Moisture(x, y),
                 w.SoilDepth(x, y)))
    }
  }
  c <-1 
}

func (w World) CalcFoliage(xBegin, xEnd int,
                           noise *opensimplex.Noise, c chan int) {
  freq := w.fFreq
  width := w.width
  height := w.height

  for y := 0; y < height; y++ {
    for x := xBegin; x < xEnd; x++ {
      xFloat := float64(x) / float64(width)
      yFloat := float64(y) / float64(height)
      f := 1 * noise.Eval2(freq * xFloat, freq * yFloat) +
             0.50 * noise.Eval2(2 * freq * xFloat, 2 * freq * yFloat) +
             0.25 * noise.Eval2(4 * freq * xFloat, 4 * freq * yFloat) +
             0.125 * noise.Eval2(8 * freq * xFloat, 8 * freq * yFloat)

      w.SetFoliage(x, y, f)
    }
  }
  c <- 1
}

func (w World) CalcRock(xBegin, xEnd int,
                        noise *opensimplex.Noise, c chan int) {
  freq := w.rFreq
  width := w.width
  height := w.height

  for y := 0; y < height; y++ {
    for x := xBegin; x < xEnd; x++ {
      xFloat := float64(x) / float64(width)
      yFloat := float64(y) / float64(height)
      r := 1 * noise.Eval2(freq * xFloat, freq * yFloat) +
             0.50 * noise.Eval2(2 * freq * xFloat, 2 * freq * yFloat) +
             0.25 * noise.Eval2(4 * freq * xFloat, 4 * freq * yFloat) +
             0.125 * noise.Eval2(8 * freq * xFloat, 8 * freq * yFloat)

      w.SetRock(x, y, r)
    }
  }
  c <- 1
}

func GenerateMap(hFreq, mFreq, sFreq, fFreq, rFreq float64,
                 width, height, numCPUs int) {
  rand.Seed(time.Now().UTC().UnixNano())
  hSeed := rand.Int63()
  mSeed := rand.Int63()
  sSeed := rand.Int63()
  fSeed := rand.Int63()
  rSeed := rand.Int63()
  fmt.Println("height seed:", hSeed)
  fmt.Println("moisture seed:", mSeed)
  fmt.Println("soil seed:", sSeed)
  fmt.Println("foliage seed:", fSeed)
  fmt.Println("rock seed:", rSeed)
  hNoise := opensimplex.NewWithSeed(hSeed)
  mNoise := opensimplex.NewWithSeed(mSeed)
  sNoise := opensimplex.NewWithSeed(sSeed)
  fNoise := opensimplex.NewWithSeed(fSeed)
  rNoise := opensimplex.NewWithSeed(rSeed)

  world := CreateWorld(width, height, hFreq, mFreq, sFreq, fFreq, rFreq, 10)

  start := time.Now()

  // Height can be computed in parallel, but needs to happen before anything
  // else.
  c := make(chan int, numCPUs)
  for i := 0; i < numCPUs; i++ {
    go world.CalcHeight(i * width / numCPUs,
                            (i + 1) * width / numCPUs,
                            hNoise, c)
  }
  for i := 0; i < numCPUs; i++ {
    <-c
  }

  c = make(chan int, 4*numCPUs + 1)
  go world.CalcGradient(c);

  for i := 0; i < numCPUs; i++ {
    go world.CalcMoisture(i * width / numCPUs,
                          (i + 1) * width / numCPUs,
                          mNoise, c)
    go world.CalcSoilDepth(i * width / numCPUs,
                           (i + 1) * width / numCPUs,
                           sNoise, c)
    go world.CalcFoliage(i * width / numCPUs,
                         (i + 1) * width / numCPUs,
                         fNoise, c)
    go world.CalcRock(i * width / numCPUs,
                         (i + 1) * width / numCPUs,
                         rNoise, c)
  }
  for i := 0; i < 4*numCPUs + 1; i++ {
    <-c
  }

  c = make(chan int, numCPUs)
  for i := 0; i < numCPUs; i++ {
    go world.CalcBiome(i * width / numCPUs,
                       (i + 1) * width / numCPUs, c)
  }
  for i := 0; i < numCPUs; i++ {
    <-c
  }

  world.AddRivers()

  c = make(chan int, numCPUs)
  for i := 0; i < numCPUs; i++ {
    go world.AddRiverBanks(i * width / numCPUs,
                           (i + 1) * width / numCPUs, c)
  }
  for i := 0; i < numCPUs; i++ {
    <-c
  }

  c = make(chan int, numCPUs)
  for i := 0; i < numCPUs; i++ {
    go world.AddGroundFeature(i * width / numCPUs,
                              (i + 1) * width / numCPUs, c)
  }
  for i := 0; i < numCPUs; i++ {
    <-c
  }

  c = make(chan int, numCPUs)
  for i := 0; i < numCPUs; i++ {
    go world.AnalyseRegions(i * width / numCPUs,
                            (i + 1) * width / numCPUs, c)
  }
  for i := 0; i < numCPUs; i++ {
    <-c
  }
  c = make(chan int, numCPUs)

  fmt.Println("Duration: ", time.Now().Sub(start));
  fmt.Println("Number of peaks: ", len(world.peaks));
  fmt.Println("Numer of lakes: ", len(world.lakes));

  DrawMap(world, hSeed, mSeed, sSeed, fSeed, rSeed, numCPUs)
}

func main() {
  width := flag.Int("width", 768, "map width")
  height := flag.Int("height", 512, "map height")
  hFreq := flag.Float64("hFreq", 5, "height noise frequency")
  mFreq := flag.Float64("mFreq", 2, "moisture noise frequency")
  sFreq := flag.Float64("sFreq", 20, "soil depth noise frequency")
  fFreq := flag.Float64("fFreq", 200, "foliage noise frequency")
  rFreq := flag.Float64("rFreq", 200, "rock noise frequency")
  threads := flag.Int("cpus", runtime.NumCPU(), "number of cores to use")

  flag.Parse()

  if *width % (REGION_SIZE * *threads) != 0 {
    fmt.Println("width needs to be a factor of", *threads * REGION_SIZE)
    return
  } else if *height % (REGION_SIZE * *threads) != 0 {
    fmt.Println("height needs to be a factor of", *threads * REGION_SIZE)
    return
  }

  fmt.Println("width, height, threads")
  fmt.Println(*width, ",", *height, ",", *threads)
  GenerateMap(*hFreq, *mFreq, *sFreq, *fFreq, *rFreq, *width, *height, *threads)
}
