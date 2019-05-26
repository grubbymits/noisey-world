package main

import (
  "container/heap"
  "flag"
  "fmt"
  "math"
  "math/rand"
  "sort"
  "time"
)

import "github.com/ojrac/opensimplex-go"

const REGION_SIZE = 8
const REGION_AREA = REGION_SIZE * REGION_SIZE

var TREE_DENSITY = [BIOMES]int {
  0,  // OCEAN
  0,  // RIVER
  0,  // BEACH
  1,  // DRY_ROCK
  2,  // MOIST_ROCK
  3,  // HEATHLAND
  4,  // SHRUBLAND
  3,  // GRASSLAND
  2,  // MOORLAND
  4,  // FENLAND
  8,  // WOODLAND
  10, // FOREST
}

var PLANT_DENSITY = [BIOMES]int {
  0,
  3,  // RIVER
  0,
  1,  // DRY_ROCK
  2,  // MOIST_ROCK
  6, // HEATHLAND
  9, // SHRUBLAND
  7, // GRASSLAND
  5, // MOORLAND
  8, // FENLAND
  5,  // WOODLAND
  4,  // FOREST
}

var ROCK_DENSITY = [BIOMES]int {
  0,  // OCEAN
  2,  // RIVER
  2,  // BEACH
  10, // DRY_ROCK
  10, // MOIST_ROCK
  1,  // HEATHLAND
  1,  // SHRUBLAND
  2,  // GRASSLAND
  6,  // MOORLAND
  1,  // FENLAND
  2,  // WOODLAND
  1,  // FOREST
}

const WATER_LEVEL = -0.35
const BEACH_LEVEL = WATER_LEVEL + 0.05
const LOWLANDS = BEACH_LEVEL + 0.3
const MIDLANDS = LOWLANDS + 0.3
const HIGHLANDS = MIDLANDS + 0.3
const NO_SOIL = -1.5
const DRY = -0.5
const MOIST = 0
const WET = 0.3
const THICK_SOIL = -0.2
const SHALLOW_SOIL = -0.7

type World struct {
  width, height int
  locations []Location
  shoreline []*Location
  regions []Location
  hFreq, mFreq, sFreq, tFreq, pFreq, rFreq, water float64
}

func CreateWorld(width, height int, hFreq, mFreq, sFreq, tFreq, pFreq, rFreq,
                 water float64) *World {
  w := new(World)
  w.width = width;
  w.height = height;
  w.locations = make([]Location, width * height)
  w.regions = make([]Location, width * height / REGION_AREA)
  w.shoreline = make([]*Location, 0, 50)
  w.hFreq = hFreq
  w.mFreq = mFreq
  w.sFreq = sFreq
  w.tFreq = tFreq
  w.pFreq = pFreq
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

func (w World) Tree(x, y int) float64 {
  return w.locations[y * w.width + x].tree
}

func (w World) Rock(x, y int) float64 {
  return w.locations[y * w.width + x].rock
}

func (w World) Plant(x, y int) float64 {
  return w.locations[y * w.width + x].plant
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

func (w World) SetTree(x, y int, t float64) {
  w.locations[y * w.width + x].tree = t
}

func (w World) SetRock(x, y int, r float64) {
  w.locations[y * w.width + x].rock = r
}

func (w World) SetPlant(x, y int, p float64) {
  w.locations[y * w.width + x].plant = p
}

func (w World) SetHeight(x, y int, h float64) {
  w.Location(x, y).height = h
}

func (w World) SetTerrace(x, y int, t uint8) {
  w.locations[y * w.width + x].terrace = t
}

func (w World) SetBiome(x, y int, b uint8) {
  w.Location(x, y).biome = b
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

func (w World) AddWater(loc *Location, saturate float64) {
  if loc.water > saturate && loc.biome != OCEAN {
    fmt.Println("Adding river tile")
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

func (w World) AddRivers(saturate float64) {
  queue := make([]Location, len(w.locations))
  copy(queue, w.locations)
  sort.Sort(ByHeight(queue))

  for n := 0; n < len(queue); n++ {
    loc := queue[n]

    minHeight := loc.height
    var lowest *Location
    for i := 0; i < loc.numSuccs; i++ {
      succ := loc.succs[i];
      if !w.isRiverValid(succ) {
        continue
      }
      if succ.height < minHeight {
        minHeight = succ.height
        lowest = succ
      }
    }

    if lowest == nil {
      continue
    }

    from := w.Location(loc.x, loc.y)
    to := w.Location(lowest.x, lowest.y)
    from.water += from.moisture
    to.water += from.water
    w.AddWater(to, saturate)
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
  // the required number of locations for each tree, rock and plant.
  for y := 0; y < w.height; y += REGION_SIZE {
    for x := xBegin; x < xEnd; x += REGION_SIZE {
      biomeCount := [BIOMES]int{0}
      treeHeap := make(LocMaxHeap, REGION_AREA)
      rockHeap := make(LocMaxHeap, REGION_AREA)
      plantHeap := make(LocMaxHeap, REGION_AREA)

      i := 0
      for ry := y; ry < y + REGION_SIZE; ry++ {
        for rx := x; rx < x + REGION_SIZE; rx++ {
          tree := 0.0
          rock := 0.0
          plant := 0.0
          loc := w.Location(rx, ry)
          biome := loc.biome
          biomeCount[biome]++
          if !loc.isWall {
            if biome != OCEAN && biome != BEACH && !loc.isRiver {
              tree = w.Tree(rx, ry)
            }
            if !loc.isRiverBank {
              plant = w.Plant(rx, ry)
            }
            rock = w.Rock(rx, ry)
          }
          treeHeap[i] = &LocVal{ i, rx, ry, tree }
          rockHeap[i] = &LocVal{ i, rx, ry, rock }
          plantHeap[i] = &LocVal{ i, rx, ry, plant }
          i++
        }
      }
      heap.Init(&treeHeap)
      heap.Init(&rockHeap)
      heap.Init(&plantHeap)

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
      for i := 0; i < ROCK_DENSITY[maxBiome]; {
        locVal := heap.Pop(&rockHeap).(*LocVal)
        if w.Location(locVal.x, locVal.y).hasFeature(TREE_FEATURE) {
          if rockHeap.Len() == 0 {
            break
          }
          continue
        }
        w.addFeature(locVal.x, locVal.y, ROCK_FEATURE);
        i++
      }
      for i := 0; i < PLANT_DENSITY[maxBiome]; i++ {
        locVal := heap.Pop(&plantHeap).(*LocVal)
        if w.Location(locVal.x, locVal.y).hasFeature(TREE_FEATURE) ||
           w.Location(locVal.x, locVal.y).hasFeature(ROCK_FEATURE) {
          if plantHeap.Len() == 0 {
            break
          }
          continue
        }
        w.addFeature(locVal.x, locVal.y, PLANT_FEATURE);
        i++
      }
    }
  }
  c <- 1
}

func (w World) CalcGradient() {
  width := w.width
  height := w.height

  for y := 0; y < height; y++ {
    for x := 0; x < width; x++ {

      centreLoc := w.Location(x, y)
      //hasPredecessor := false
      //hasSuccessor := false

      if y - 1 >= 0 {
        north := w.Location(x, y - 1)
        if north.height <= centreLoc.height {
          centreLoc.addSuccessor(north)
          //hasSuccessor = true
          if north.terrace < centreLoc.terrace {
            centreLoc.addFeature(HORIZONTAL_SHADOW_FEATURE)
          }
        } else if north.height >= centreLoc.height {
          //centreLoc.addPredecessor(north)
          //hasPredecessor = true
          if north.terrace > centreLoc.terrace {
            centreLoc.addFeature(HORIZONTAL_SHADOW_FEATURE)
          }
        }
      }

      if y + 1 < height {
        south := w.Location(x, y + 1)
        if south.height <= centreLoc.height {
          centreLoc.addSuccessor(south)
          //hasSuccessor = true
        } else if south.height >= centreLoc.height {
          //centreLoc.addPredecessor(south)
          //hasPredecessor = true
        }
      }

      if x + 1 < width {
        east := w.Location(x + 1, y)
        if east.height <= centreLoc.height {
          centreLoc.addSuccessor(east)
          //hasSuccessor = true
        } else if east.height >= centreLoc.height {
          //centreLoc.addPredecessor(east)
          //hasPredecessor = true
          if east.terrace > centreLoc.terrace {
            centreLoc.addFeature(LEFT_SHADOW_FEATURE)
          }
        }
      }

      if x - 1 >= 0 {
        west := w.Location(x - 1, y)
        if west.height <= centreLoc.height {
          centreLoc.addSuccessor(west)
          //hasSuccessor = true
        } else if west.height >= centreLoc.height {
          //centreLoc.addPredecessor(west)
          //hasPredecessor = true
          if west.terrace > centreLoc.terrace {
            centreLoc.addFeature(RIGHT_SHADOW_FEATURE)
          }
        }
      }

      if x + 1 < width && y - 1 >= 0 {
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

      // Calculate the total gradient of the successors, which will be used to
      // determine the ratio of water flow. 
      for i := 0; i < centreLoc.numSuccs; i++ {
        succ := centreLoc.succs[i]
        centreLoc.totalGradient += math.Abs(centreLoc.height) - math.Abs(succ.height)
      }
    }
  }
}

func (w World) FindHighest(xBegin, xEnd int, highest **Location, c chan int) {
  *highest = w.Location(xBegin, 0)

  for y := 0; y < w.height; y++ {
    for x := xBegin; x < xEnd; x++ {
      loc := w.Location(x, y)
      if loc.hasFeature(TREE_FEATURE) ||
         loc.hasFeature(ROCK_FEATURE) ||
         loc.isWall || loc.isRiver || loc.isRiverBank {
        continue
      }
      if loc.height > (*highest).height {
        *highest = loc
      }
    }
  }
  c <-1
}

func (world World) CalcHeight(xBegin, xEnd int,
                              base, edgeUp, edgeDown, falloff float64,
                              noise *opensimplex.Noise, c chan int) {
  freq := world.hFreq
  width := world.width
  height := world.height
  n := *noise;
  cx := float64(width / 2);
  cy := float64(height / 2);

  for y := 0; y < height; y++ {
    yFloat := float64(y) / float64(height)
    yBias := 0.0
    for x := xBegin; x < xEnd; x++ {
      xFloat := float64(x) / float64(width)
      xBias :=  0.0
      h := base +
           0.75 * n.Eval2(freq * xFloat, freq * yFloat) +
           0.50 * n.Eval2(2 * freq * xFloat, 2 * freq * yFloat) +
           0.25 * n.Eval2(4 * freq * xFloat, 4 * freq * yFloat) +
           0.125 * n.Eval2(8 * freq * xFloat, 8 * freq * yFloat) +
	   xBias + yBias

      nx := (cx - float64(x)) / cx
      ny := (cy - float64(y)) / cy
      distance := float64(2*math.Max(math.Abs(nx), math.Abs(ny)))
      edgeUp := 0.07
      edgeDown := 0.5
      falloff := 1.5
      h += edgeUp - edgeDown * math.Pow(distance, falloff)

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
	n := *noise

  for y := 0; y < height; y++ {
    for x := xBegin; x < xEnd; x++ {
      xFloat := float64(x) / float64(width)
      yFloat := float64(y) / float64(height)
      s := 1 * n.Eval2(freq * xFloat, freq * yFloat) +
             0.50 * n.Eval2(2 * freq * xFloat, 2 * freq * yFloat) +
             0.25 * n.Eval2(4 * freq * xFloat, 4 * freq * yFloat) +
             0.125 * n.Eval2(8 * freq * xFloat, 8 * freq * yFloat)

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
  n := *noise

  for y := 0; y < height; y++ {
    for x := xBegin; x < xEnd; x++ {
      xFloat := float64(x) / float64(width)
      yFloat := float64(y) / float64(height)
      m :=   0.75 * n.Eval2(freq * xFloat, freq * yFloat) +
             0.50 * n.Eval2(2 * freq * xFloat, 2 * freq * yFloat) +
             0.25 * n.Eval2(4 * freq * xFloat, 4 * freq * yFloat) +
             0.125 * n.Eval2(8 * freq * xFloat, 8 * freq * yFloat)

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

func (w World) CalcTrees(xBegin, xEnd int,
                         noise *opensimplex.Noise, c chan int) {
  freq := w.tFreq
  width := w.width
  height := w.height
	n := *noise

  for y := 0; y < height; y++ {
    for x := xBegin; x < xEnd; x++ {
      xFloat := float64(x) / float64(width)
      yFloat := float64(y) / float64(height)
      f := 1 * n.Eval2(freq * xFloat, freq * yFloat) +
             0.50 * n.Eval2(2 * freq * xFloat, 2 * freq * yFloat) +
             0.25 * n.Eval2(4 * freq * xFloat, 4 * freq * yFloat) +
             0.125 * n.Eval2(8 * freq * xFloat, 8 * freq * yFloat)

      w.SetTree(x, y, f)
    }
  }
  c <- 1
}

func (w World) CalcPlants(xBegin, xEnd int,
                          noise *opensimplex.Noise, c chan int) {
  freq := w.pFreq
  width := w.width
  height := w.height
	n := *noise

  for y := 0; y < height; y++ {
    for x := xBegin; x < xEnd; x++ {
      xFloat := float64(x) / float64(width)
      yFloat := float64(y) / float64(height)
      f := 1 * n.Eval2(freq * xFloat, freq * yFloat) +
             0.50 * n.Eval2(2 * freq * xFloat, 2 * freq * yFloat) +
             0.25 * n.Eval2(4 * freq * xFloat, 4 * freq * yFloat) +
             0.125 * n.Eval2(8 * freq * xFloat, 8 * freq * yFloat)

      w.SetPlant(x, y, f)
    }
  }
  c <- 1
}

func (w World) CalcRock(xBegin, xEnd int,
                        noise *opensimplex.Noise, c chan int) {
  freq := w.rFreq
  width := w.width
  height := w.height
	n := *noise

  for y := 0; y < height; y++ {
    for x := xBegin; x < xEnd; x++ {
      xFloat := float64(x) / float64(width)
      yFloat := float64(y) / float64(height)
      r := 1 * n.Eval2(freq * xFloat, freq * yFloat) +
             0.50 * n.Eval2(2 * freq * xFloat, 2 * freq * yFloat) +
             0.25 * n.Eval2(4 * freq * xFloat, 4 * freq * yFloat) +
             0.125 * n.Eval2(8 * freq * xFloat, 8 * freq * yFloat)

      w.SetRock(x, y, r)
    }
  }
  c <- 1
}

func (w *World) FindRuntimePath(start *Location) {
  graph := CreateGraph(w)
  frontier := make([]*GraphNode, 1)
  frontier[0] = graph.getNode(start)
  cameFrom := make(map[*GraphNode] *GraphNode)

  for i := range frontier {
    current := frontier[i]
    neighbours := current.neighbours
    for n := 0; n < current.numNeighbours; n++ {
      next := neighbours[n]
      if cameFrom[next] == nil {
        frontier = append(frontier, next)
        cameFrom[next] = current
      }
    }
  }
}

func (w *World) GeneratePath(start, goal *Location) bool {
  graph := CreateGraph(w)
  startNode := graph.getNode(start)
  goalNode := graph.getNode(goal)
  frontier := make(NodeQueue, 1)
  frontier[0] = &SortableNode{startNode, 0}
  came_from := make(map[*GraphNode] *GraphNode)
  cost_so_far := make(map[*GraphNode] float64)
  cost_so_far[startNode] = 0

  found := false
  for ; len(frontier) > 0; {
    current := frontier[0].node
    frontier = frontier[1:]

    if current == goalNode {
      found = true
      break
    }

    neighbours := current.neighbours
    for n := 0; n < current.numNeighbours; n++ {
      next := neighbours[n]
      new_cost := cost_so_far[current] + graph.cost(current, next)

      next_cost, visited := cost_so_far[next]
      if !visited || new_cost < next_cost {
        cost_so_far[next] = new_cost
        priority := new_cost // + heuristic
        frontier = append(frontier, &SortableNode{ next, priority })
        came_from[next] = current
      }
    }
    sort.Sort(NodeQueue(frontier))
  }

  if found {
    fmt.Println("found a path")
    next := came_from[goalNode]

    for ; next != startNode; {
      next = came_from[next]
      next.loc.addFeature(PATH_FEATURE)
    }
  } else {
    fmt.Println("failed to find a path")
  }
  return found
}

func GenerateMap(hFreq, heightBaseline, edgeUp, edgeDown, falloff,
                 mFreq, water, saturate,
                 sFreq, tFreq, pFreq, rFreq float64,
                 width, height, numCPUs int) {
  rand.Seed(time.Now().UTC().UnixNano())
  hSeed := rand.Int63()
  mSeed := rand.Int63()
  sSeed := rand.Int63()
  tSeed := rand.Int63()
  pSeed := rand.Int63()
  rSeed := rand.Int63()
  fmt.Println("height seed:", hSeed)
  fmt.Println("moisture seed:", mSeed)
  fmt.Println("soil seed:", sSeed)
  fmt.Println("tree seed:", tSeed)
  fmt.Println("plant seed:", pSeed)
  fmt.Println("rock seed:", rSeed)
  hNoise := opensimplex.New(hSeed)
  mNoise := opensimplex.New(mSeed)
  sNoise := opensimplex.New(sSeed)
  tNoise := opensimplex.New(tSeed)
  pNoise := opensimplex.New(pSeed)
  rNoise := opensimplex.New(rSeed)

  world := CreateWorld(width, height, hFreq, mFreq, sFreq, tFreq, pFreq, rFreq,
                       water)

  start := time.Now()

  numThreads := 6 * numCPUs
  c := make(chan int, numThreads)
  for i := 0; i < numCPUs; i++ {
    xBegin := i * width / numCPUs
    xEnd := (i + 1) * width / numCPUs
    go world.CalcHeight(xBegin, xEnd, heightBaseline, edgeUp, edgeDown,
                        falloff, &hNoise, c)
    go world.CalcMoisture(xBegin, xEnd, &mNoise, c)
    go world.CalcSoilDepth(xBegin, xEnd,  &sNoise, c)
    go world.CalcTrees(xBegin, xEnd, &tNoise, c)
    go world.CalcPlants(xBegin, xEnd, &pNoise, c)
    go world.CalcRock(xBegin, xEnd, &rNoise, c)
  }

  for i := 0; i < numThreads; i++ {
    <-c
  }

  world.CalcGradient();

  c = make(chan int, numCPUs)
  for i := 0; i < numCPUs; i++ {
    go world.CalcBiome(i * width / numCPUs,
                       (i + 1) * width / numCPUs, c)
  }
  for i := 0; i < numCPUs; i++ {
    <-c
  }

  world.AddRivers(saturate)

  numThreads = numCPUs * 2
  c = make(chan int, numThreads)
  for i := 0; i < numCPUs; i++ {
    xBegin := i * width / numCPUs
    xEnd := (i + 1) * width / numCPUs
    go world.AddRiverBanks(xBegin, xEnd, c)
    go world.AddGroundFeature(xBegin, xEnd, c)
  }
  for i := 0; i < numThreads; i++ {
    <-c
  }

  c = make(chan int, numCPUs)
  for i := 0; i < numCPUs; i++ {
    xBegin := i * width / numCPUs
    xEnd := (i + 1) * width / numCPUs
    go world.AnalyseRegions(xBegin, xEnd, c)
  }
  for i := 0; i < numCPUs; i++ {
    <-c
  }

  for y := 0; y < world.height; y++ {
    for x := 0; x < world.width; x++ {
      loc := world.Location(x, y)
      if loc.biome == BEACH {
        world.shoreline = append(world.shoreline, loc)
      }
    }
  }

  fmt.Println("size of shoreline: ", len(world.shoreline))
  lowest := world.shoreline[0]
  for i := 1; i < len(world.shoreline); i++ {
    beach := world.shoreline[i]
    if beach.height < lowest.height {
      lowest = beach
    }
  }

  numThreads = 4
  highs := make([]*Location, 0, numThreads)
  c = make(chan int, numThreads)
  for i := 0; i < numThreads; i++ {
    xBegin := i * width / numThreads
    xEnd := (i + 1) * width / numThreads
    highs = append(highs, nil)
    go world.FindHighest(xBegin, xEnd, &(highs[i]), c)
  }
  for i := 0; i < numThreads; i++ {
    <-c
  }
  highest := highs[0]
  for i := 1; i < len(highs); i++ {
    if highs[i].height > highest.height {
      highest = highs[i]
    }
  }

  world.GeneratePath(lowest, highest)

  fmt.Println("Duration: ", time.Now().Sub(start));

  DrawMap(world, hSeed, mSeed, sSeed, tSeed, rSeed, numCPUs)
  ExportJSON(world)
}

func main() {
  // 64 x 48 = 1024 x 768
  // 128 x 96 = 2048 x 1546
  // 192 x 144 = 3072 x 2304
  // 192 x 192 = 3072 x 3072
  width := flag.Int("width", 192, "map width")
  height := flag.Int("height", 144, "map height")
  hFreq := flag.Float64("hFreq", 1.6, "height noise frequency")
  bias := flag.Float64("bias", 0.0, "height bias")
  edgeUp := flag.Float64("raise-edge", 0.05, "raise edges")
  edgeDown := flag.Float64("lower-edge", 0.4, "lower edges")
  falloff := flag.Float64("falloff", 5.0, "falloff rate")
  mFreq := flag.Float64("mFreq", 1.0, "moisture noise frequency")
  water := flag.Float64("water", 50, "water")
  saturate := flag.Float64("saturate", 10, "water saturation level") 
  sFreq := flag.Float64("sFreq", 20, "soil depth noise frequency")
  tFreq := flag.Float64("tFreq", 200, "tree noise frequency")
  pFreq := flag.Float64("pFreq", 200, "plant noise frequency")
  rFreq := flag.Float64("rFreq", 200, "rock noise frequency")
  threads := flag.Int("threads", 1, "number of cores to use")

  flag.Parse()

  if *width % (REGION_SIZE * *threads) != 0 {
    fmt.Println("With a region size of", REGION_SIZE,
                ", width needs to be a factor of", *threads * REGION_SIZE,
                "to use", *threads, "threads.")
    return
  } else if *height % (REGION_SIZE * *threads) != 0 {
    fmt.Println("With a region size of", REGION_SIZE,
                ", height needs to be a factor of", *threads * REGION_SIZE,
                "to use", *threads, "threads.")
    return
  }

  fmt.Println("width, height, threads")
  fmt.Println(*width, ",", *height, ",", *threads)
  GenerateMap(*hFreq, *bias, *edgeUp, *edgeDown, *falloff,
              *mFreq, *water, *saturate,
              *sFreq, *tFreq, *pFreq, *rFreq,
              *width, *height, *threads)
}
