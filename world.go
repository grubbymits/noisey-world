package main

import (
  "container/heap"
  "flag"
  "fmt"
  "image"
  "image/color"
  "image/png"
  "math"
  "math/rand"
  "log"
  "os"
  "runtime"
  "strconv"
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
  REGION_AREA / 96,   // HEATHLAND
  REGION_AREA / 64,   // SHRUBLAND
  REGION_AREA / 128,  // GRASSLAND
  REGION_AREA / 128,  // MOORLAND
  REGION_AREA / 128,  // FENLAND
  REGION_AREA / 32,   // WOODLAND
  REGION_AREA / 16,   // FOREST
}

const WATER_LEVEL = -0.4
const WATER_SATURATION = 4
const NO_SOIL = -1.5
const DRY = -0.5
const MOIST = 0
const WET = 0.3
const THICK_SOIL = -0.2
const SHALLOW_SOIL = -0.7
const HIGHLANDS = 0.5
const MIDLANDS = 0

type World struct {
  width, height int
  locations []Location
  regions []Location
  peaks map[*Location]bool
  lakes map[*Location]bool
  hFreq, mFreq, sFreq, fFreq, water float64
}

func CreateWorld(width, height int, hFreq, mFreq, sFreq, fFreq, water float64) *World {
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
  w.water = water

  for y := 0; y < height; y++ {
    for x := 0; x < width; x++ {
       loc := w.Location(x, y)
       loc.x = x
       loc.y =y
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

func (w World) addFeature(x, y int, feature uint8) {
  w.locations[y * w.width + x].feature = feature
}

func (w World) Feature(x, y int) uint8 {
  return w.locations[y * w.width + x].feature
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

func (w World) SetHeight(x, y int, h float64) {
  w.locations[y * w.width + x].height = h
}

func (w World) SetBiome(x, y int, b uint8) {
  w.locations[y * w.width + x].biome = b
}

func (w World) AddRivers() {
  queue := make([]*Location, len(w.peaks))

  i := 0
  for loc, _ := range w.peaks {
    loc.water = w.water + loc.moisture
    queue[i] = loc
    i++
  }

  // Iterate through all queue, which will hold every location eventually.
  for n := 0; n < w.height * w.width; n++ {
    loc := queue[0]
    queue = queue[1:]

    // Receive water from all of the predecessors. 
    for i := 0; i < loc.numPreds; i++ {

      pred := loc.preds[i]
      totalGradient := 0.0

      // Calculate percentage of predecessor's water the successor will
      // receive.
      for j := 0; j < pred.numSuccs; j++ {
        succ := pred.succs[j]
        totalGradient += math.Abs(pred.height) - math.Abs(succ.height)
      }

      gradientRatio := (math.Abs(pred.height) - math.Abs(loc.height)) / totalGradient
      water := gradientRatio * pred.water
      loc.water += water
    }

    if loc.moisture > 1 {
      loc.water *= loc.moisture
    }

    // Discover all of the Location's successors and add them to the queue if
    // they've been discovered by all their predecessors.
    for i := 0; i < loc.numSuccs; i++ {
      succ := loc.succs[i]
      succ.discovered++
      if succ.discovered == succ.numPreds {
        queue = append(queue, succ)
        succ.discovered = 0 // reset for next phase
      }
    }
  }

  // Traverse back up from lakes to fill out the lakes and the lower regions of
  // the rivers.
  queue = make([]*Location, len(w.lakes))
  i = 0
  for loc, _ := range w.lakes {
    queue[i] = loc
    i++
  }
  for n := 0; n < w.height * w.width; n++ {
    loc := queue[0]
    queue = queue[1:]

    for i := 0; i < loc.numPreds; i++ {
      pred := loc.preds[i]
      if loc.water > WATER_SATURATION {
        water := (loc.water - WATER_SATURATION) / float64(loc.numPreds)
        pred.water += water
      }
      pred.discovered++
      if pred.discovered == pred.numSuccs {
        queue = append(queue, pred)
        pred.discovered = 0
      }
    }
  }

  // Second pass down
  queue = make([]*Location, len(w.peaks))
  i = 0
  for loc, _ := range w.peaks {
    queue[i] = loc
    i++
  }

  fmt.Println("Second pass, size of queue: ", len(queue))
  for n := 0; n < w.height * w.width; n++ {
    loc := queue[0]
    queue = queue[1:]

    // Receive water from all of the predecessors. 
    for i := 0; i < loc.numPreds; i++ {

      pred := loc.preds[i]
      totalGradient := 0.0

      // Calculate percentage of predecessor's water the successor will
      // receive.
      for j := 0; j < pred.numSuccs; j++ {
        succ := pred.succs[j]
        totalGradient += math.Abs(pred.height) - math.Abs(succ.height)
      }

      gradientRatio := (math.Abs(pred.height) - math.Abs(loc.height)) / totalGradient
      water := gradientRatio * pred.water
      loc.water += water
    }

    if loc.water > WATER_SATURATION && loc.biome != OCEAN {
      loc.biome = RIVER
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
      lmh := make(LocMaxHeap, REGION_AREA)

      i := 0
      for ry := y; ry < y + REGION_SIZE; ry++ {
        for rx := x; rx < x + REGION_SIZE; rx++ {
          foliage := 0.0
          biome := w.Biome(rx, ry)
          biomeCount[biome]++
          if biome != OCEAN && biome != BEACH && biome != RIVER {
            foliage = w.Foliage(rx, ry)
          }
          lmh[i] = &LocVal{ i, rx, ry, foliage }
          i++
        }
      }
      heap.Init(&lmh)

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
        locVal := heap.Pop(&lmh).(*LocVal)
        w.addFeature(locVal.x, locVal.y, TREE);
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
        otherLoc := w.Location(x, y - 1)
        if otherLoc.height < centreLoc.height {
          centreLoc.addSuccessor(otherLoc)
          hasSuccessor = true
        } else if otherLoc.height > centreLoc.height {
          centreLoc.addPredecessor(otherLoc)
          hasPredecessor = true
        }
      }

      if x + 1 < width {
        otherLoc := w.Location(x + 1, y)
        if otherLoc.height < centreLoc.height {
          centreLoc.addSuccessor(otherLoc)
          hasSuccessor = true
        } else if otherLoc.height > centreLoc.height {
          centreLoc.addPredecessor(otherLoc)
          hasPredecessor = true
        }
      }

      if y + 1 < height {
        otherLoc := w.Location(x, y + 1)
        if otherLoc.height < centreLoc.height {
          centreLoc.addSuccessor(otherLoc)
          hasSuccessor = true
        } else if otherLoc.height > centreLoc.height {
          centreLoc.addPredecessor(otherLoc)
          hasPredecessor = true
        }
      }

      if x - 1 >= 0 {
        otherLoc := w.Location(x - 1, y)
        if otherLoc.height < centreLoc.height {
          centreLoc.addSuccessor(otherLoc)
          hasSuccessor = true
        } else if otherLoc.height > centreLoc.height {
          centreLoc.addPredecessor(otherLoc)
          hasPredecessor = true
        }
      }
      if !hasPredecessor {
        w.addPeak(w.Location(x, y))
      } else if !hasSuccessor {
        w.addLake(w.Location(x, y))
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
      m := 1 * noise.Eval2(freq * xFloat, freq * yFloat) +
             0.50 * noise.Eval2(2 * freq * xFloat, 2 * freq * yFloat) +
             0.25 * noise.Eval2(4 * freq * xFloat, 4 * freq * yFloat) +
             0.125 * noise.Eval2(8 * freq * xFloat, 8 * freq * yFloat)

      w.SetMoisture(x, y, m)
    }
  }
  c <- 1
}

func (w World) CalcBiome(xBegin, xEnd int, c chan int) {
  height := w.height

  for y := 0; y < height; y++ {
    for x := xBegin; x < xEnd; x++ {
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

func GenerateMap(hFreq, mFreq, sFreq, fFreq float64, width, height, numCPUs int) {
  rand.Seed(time.Now().UTC().UnixNano())
  hSeed := rand.Int63()
  mSeed := rand.Int63()
  sSeed := rand.Int63()
  fSeed := rand.Int63()
  fmt.Println("height seed:", hSeed)
  fmt.Println("moisture seed:", mSeed)
  fmt.Println("soil seed:", sSeed)
  fmt.Println("foliage seed:", fSeed)
  hNoise := opensimplex.NewWithSeed(hSeed)
  mNoise := opensimplex.NewWithSeed(mSeed)
  sNoise := opensimplex.NewWithSeed(sSeed)
  fNoise := opensimplex.NewWithSeed(fSeed)

  world := CreateWorld(width, height, hFreq, mFreq, sFreq, fFreq, 1.5)

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

  c = make(chan int, 3*numCPUs + 1)
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
  }
  for i := 0; i < 3*numCPUs + 1; i++ {
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

  img := image.NewRGBA(image.Rect(0, 0, width, height))
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

  bounds := img.Bounds()
  for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
    for x := bounds.Min.X; x < bounds.Max.X; x++ {
      if (world.Feature(x, y) != EMPTY) {
        img.Set(x, y, color.RGBA{38, 77, 0, 255})
      } else {
        img.Set(x, y, colours[world.Biome(x, y)])
      }
    }
  }

  filename := "h" + strconv.FormatInt(hSeed, 10) + "-" +
              "m" + strconv.FormatInt(mSeed, 10) + "-" +
              "s" + strconv.FormatInt(sSeed, 10) + ".png"
  imgFile, err := os.Create(filename)
  if err != nil {
    log.Fatal(err)
  }
  if err := png.Encode(imgFile, img); err != nil {
    imgFile.Close();
    log.Fatal(err)
  }
  if err := imgFile.Close(); err != nil {
    log.Fatal(err)
  }
}

func main() {
  width := flag.Int("width", 2304, "map width")
  height := flag.Int("height", 1536, "map height")
  hFreq := flag.Float64("hFreq", 5, "height noise frequency")
  mFreq := flag.Float64("mFreq", 2, "moisture noise frequency")
  sFreq := flag.Float64("sFreq", 20, "soil depth noise frequency")
  fFreq := flag.Float64("fFreq", 200, "foliage noise frequency")
  threads := flag.Int("cpus", runtime.NumCPU(), "number of cores to use")

  flag.Parse()

  if *width % REGION_SIZE != 0 {
    fmt.Println("width needs to be a factor of", REGION_SIZE)
    return
  } else if *height % REGION_SIZE != 0 {
    fmt.Println("height needs to be a factor of", REGION_SIZE)
    return
  }

  fmt.Println("width, height, threads")
  fmt.Println(*width, ",", *height, ",", *threads)
  GenerateMap(*hFreq, *mFreq, *sFreq, *fFreq, *width, *height, *threads)
}
