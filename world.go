package main

import (
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

const (
  OCEAN = iota
  RIVER
  BEACH
  DRY_ROCK
  WET_ROCK
  HEATHLAND
  SHRUBLAND
  GRASSLAND
  MOORLAND
  FENLAND
  FOREST
  BIOMES
)

const WATER_LEVEL = -0.4
const WATER_SATURATION = 4
const NO_SOIL = -1.5
const DRY = -0.5
const WET = 0
const VERY_WET = 0.3
const THICK_SOIL = -0.2
const HIGHLANDS = 0.3


func biome(h, m, s float64) uint8 {
  // Height, Moisture and Soil Depth

  // - bare rock
  // - lichen rock

  // - grassland: anywhere?
  // - moorland: high, wet, deep soil

  // - fenland: low, very wet, deep soil
  // - heathland: low, dry, thin soil

  // - temperate rainforest: low, wet, deep soil
  // - forest: higher range than rainforst, wet, deep soil
  // - shrubland: drier with thinner soil than forest, wider range

  // - marshland: saturated water along rivers
  // - beach

  if (h < WATER_LEVEL) {
    return OCEAN
  } else if (h < -0.3) {
    return BEACH
  }

  // No soil
  if (s < NO_SOIL) {
    if (m < DRY) {
      return DRY_ROCK
    }
    return WET_ROCK
  }

  // Thin soils
  if (s < THICK_SOIL) {
    if (h < HIGHLANDS) {
      if (m < DRY) {
        return HEATHLAND
      }
      return SHRUBLAND
    }
    return GRASSLAND
  }

  // Thick soils
  if (h > HIGHLANDS) {
    if (m > VERY_WET) {
      return MOORLAND
    } else if (m > WET) {
      return FOREST
    } else if (m > DRY) {
      return SHRUBLAND
    }
    return GRASSLAND
  }

  if (m > VERY_WET) {
    return FENLAND
  } else if (m > WET) {
    return FOREST
  } else if (m > DRY) {
    return SHRUBLAND
  }
  return GRASSLAND
}

type Location struct {
  height, moisture, soilDepth float64
  x, y int
  preds, succs [4]*Location
  numPreds, numSuccs int
  discovered, weight int
  water float64
  biome uint8
}

func (l *Location) addSuccessor(other *Location) {
  l.succs[l.numSuccs] = other
  l.numSuccs++
}

func (l *Location) addPredecessor(other *Location) {
  l.preds[l.numPreds] = other
  l.numPreds = l.numPreds + 1
}

type World struct {
  width, height int
  locations []Location
  peaks map[*Location]bool
  lakes map[*Location]bool
  hFreq, mFreq, sFreq, water float64
}

func CreateWorld(width, height int, hFreq, mFreq, sFreq, water float64) *World {
  w := new(World)
  w.width = width;
  w.height = height;
  w.locations = make([]Location, width * height)
  w.peaks = make(map[*Location]bool)
  w.lakes = make(map[*Location]bool)
  w.hFreq = hFreq
  w.mFreq = mFreq
  w.sFreq = sFreq
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

func (w World) Location(x, y int) *Location {
  return &w.locations[y * w.width + x]
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

func (w World) Biome(x, y int) uint8 {
  return w.locations[y * w.width + x].biome
}

func (w World) SetMoisture(x, y int, m float64) {
  w.locations[y * w.width + x].moisture = m
}

func (w World) SetSoilDepth(x, y int, s float64) {
  w.locations[y * w.width + x].soilDepth = s
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

func (w World) AnalyseRegions(c chan int) {
  // Divide the world into regions and calculate attributes of each region.
  // If a loc is a 16x16 tile, a region could be 64x64 tiles.
  // - ratio of ocean
  // - ratio of beach
  // - ratio of rivers
  // - average height
  // - mean average biome
  // - number of trees
  // - number of rocks
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

func (world World) CalcMoisture(xBegin, xEnd int,
                                noise *opensimplex.Noise,
                                c chan int) {
  freq := world.mFreq
  width := world.width
  height := world.height

  for y := 0; y < height; y++ {
    for x := xBegin; x < xEnd; x++ {
      xFloat := float64(x) / float64(width)
      yFloat := float64(y) / float64(height)
      m := 1 * noise.Eval2(freq * xFloat, freq * yFloat) +
             0.50 * noise.Eval2(2 * freq * xFloat, 2 * freq * yFloat) +
             0.25 * noise.Eval2(4 * freq * xFloat, 4 * freq * yFloat) +
             0.125 * noise.Eval2(8 * freq * xFloat, 8 * freq * yFloat)

      world.SetMoisture(x, y, m)
    }
  }
  c <- 1
}

func (world World) CalcBiome(xBegin, xEnd int, c chan int) {
  height := world.height

  for y := 0; y < height; y++ {
    for x := xBegin; x < xEnd; x++ {
      world.SetBiome(x, y, biome(world.Height(x, y), world.Moisture(x, y),
                                 world.SoilDepth(x, y)))
    }
  }
  c <-1 
}

func GenerateMap(hFreq, mFreq, sFreq float64, width, height, numCPUs int) {
  rand.Seed(time.Now().UTC().UnixNano())
  hSeed := rand.Int63()
  mSeed := rand.Int63()
  sSeed := rand.Int63()
  fmt.Println("height seed:", hSeed)
  fmt.Println("moisture seed:", mSeed)
  fmt.Println("soil seed:", sSeed)
  hNoise := opensimplex.NewWithSeed(hSeed)
  mNoise := opensimplex.NewWithSeed(mSeed)
  sNoise := opensimplex.NewWithSeed(sSeed)

  world := CreateWorld(width, height, hFreq, mFreq, sFreq, 1.5)//WATER_SATURATION / 2)

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

  c = make(chan int, 2*numCPUs + 1)
  go world.CalcGradient(c);

  for i := 0; i < numCPUs; i++ {
    go world.CalcMoisture(i * width / numCPUs,
                          (i + 1) * width / numCPUs,
                          mNoise, c)
    go world.CalcSoilDepth(i * width / numCPUs,
                           (i + 1) * width / numCPUs,
                           sNoise, c)
  }
  for i := 0; i < 2*numCPUs + 1; i++ {
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

  fmt.Println("Duration: ", time.Now().Sub(start));
  fmt.Println("Number of peaks: ", len(world.peaks));
  fmt.Println("Numer of lakes: ", len(world.lakes));

  img := image.NewRGBA(image.Rect(0, 0, width, height))
  colours := [BIOMES]color.RGBA{{ 51, 166, 204, 255 },  // OCEAN
                                { 0, 102, 102, 255 },   // RIVER
                                { 255, 230, 128, 255 }, // BEACH
                                { 204, 204, 204, 255 }, // DRY_ROCK
                                { 166, 166, 166, 255 }, // WET_ROCK
                                { 202, 218, 114, 255 }, // HEATHLAND
                                { 170, 190, 50, 255 },  // SHRUBLAND
                                { 128, 153, 51, 255 },  // GRASSLAND
                                { 217, 179, 255, 255 }, // MOORLAND
                                { 85, 128, 0, 255 },    // FENLAND
                                { 77, 153, 0, 255 } }   // FOREST

  bounds := img.Bounds()
  for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
    for x := bounds.Min.X; x < bounds.Max.X; x++ {
      img.Set(x, y, colours[world.Biome(x, y)])
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
  width := flag.Int("width", 2880, "map width")
  height := flag.Int("height", 1800, "map height")
  hFreq := flag.Float64("hfreq", 5, "height noise frequency")
  mFreq := flag.Float64("mfreq", 2, "moisture noise frequency")
  sFreq := flag.Float64("sfreq", 20, "soil depth noise frequency")
  threads := flag.Int("cpus", runtime.NumCPU(), "number of cores to use")

  flag.Parse()

  fmt.Println("width, height, threads")
  fmt.Println(*width, ",", *height, ",", *threads)
  GenerateMap(*hFreq, *mFreq, *sFreq, *width, *height, *threads)
}
