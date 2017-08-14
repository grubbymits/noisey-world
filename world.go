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
  BEACH
  SCORCHED
  BARE
  TUNDRA
  SNOW
  TEMPERATE_DESERT
  SHRUBLAND
  TAIGA
  GRASSLAND
  TEMPERATE_DECIDUOUS_FOREST
  TEMPERATE_RAIN_FOREST
  SUBTROPICAL_DESERT
  TROPICAL_SEASONAL_FOREST
  TROPICAL_RAIN_FOREST
)

const WATER_LEVEL = -0.4
const WATER_SATURATION = 1

func biome(h, m float64) uint8 {
  if (h < WATER_LEVEL) {
    return OCEAN
  } else if (h < -0.3) {
    return BEACH
  } else if (h > 0.7) {
    if (m < -0.6) {
      return SCORCHED
    } else if (m < -0.2) {
      return BARE
    } else if (m < 0.2) {
      return TUNDRA
    }
    return SNOW
  } else if (h > 0.5) {
    if (m < -0.6) {
      return TEMPERATE_DESERT
    } else if (m < -0.2) {
      return SHRUBLAND
    }
    return TAIGA;
  } else if (h > 0.2) {
    if (m < -0.6) {
      return TEMPERATE_DESERT
    } else if (m < 0) {
      return GRASSLAND
    } else if (m < 0.6) {
      return TEMPERATE_DECIDUOUS_FOREST
    }
    return TEMPERATE_RAIN_FOREST
  }

  if (m < -0.8) {
    return SUBTROPICAL_DESERT
  } else if (m < -0.2) {
    return GRASSLAND
  } else if (m < 0.4) {
    return TROPICAL_SEASONAL_FOREST
  } else {
    return TROPICAL_RAIN_FOREST
  }
}

type Location struct {
  height, moisture float64
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
  hFreq, mFreq, water float64
}

func CreateWorld(width, height int, hFreq, mFreq, water float64) *World {
  w := new(World)
  w.width = width;
  w.height = height;
  w.locations = make([]Location, width * height)
  w.peaks = make(map[*Location]bool)
  w.lakes = make(map[*Location]bool)
  w.hFreq = hFreq
  w.mFreq = mFreq
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

func (w World) Biome(x, y int) uint8 {
  return w.locations[y * w.width + x].biome
}

func (w World) SetMoisture(x, y int, m float64) {
  w.locations[y * w.width + x].moisture = m
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

  // On the second pass, the amount of water passed to the successors isn't
  // based upon the gradient but the amount of water. The water is distributed
  // to try to even the flow.
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
      totalWater := 0.0

      // Calculate percentage of predecessor's water the successor will
      // receive.
      for j := 0; j < pred.numSuccs; j++ {
        succ := pred.succs[j]
        totalWater += succ.water
      }

      waterRatio := totalWater / loc.water
      water := waterRatio * (pred.water - WATER_SATURATION)
      loc.water += water
    }

    if loc.water > WATER_SATURATION {
      w.SetBiome(loc.x, loc.y, OCEAN)
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

      h  = math.Pow(h, float64(1 + (y / height)))
      world.SetHeight(x, y, h)
    }
  }
  c <- 1
}

func (world World) CalcBiome(xBegin, xEnd int,
                                  noise *opensimplex.Noise,
                                  c chan int) {
  freq := world.mFreq
  width := world.width
  height := world.height
  //tempBias := 1 / height

  for y := 0; y < height; y++ {
    for x := xBegin; x < xEnd; x++ {
      xFloat := float64(x) / float64(width)
      yFloat := float64(y) / float64(height)
      m := 1 * noise.Eval2(freq * xFloat, freq * yFloat) +
             0.50 * noise.Eval2(2 * freq * xFloat, 2 * freq * yFloat) +
             0.25 * noise.Eval2(4 * freq * xFloat, 4 * freq * yFloat) +
             0.125 * noise.Eval2(8 * freq * xFloat, 8 * freq * yFloat)

      //m = math.Pow(m, tempBias * float64(y))
      world.SetMoisture(x, y, m)
      world.SetBiome(x, y, biome(world.Height(x, y), m))
      //world.CalcGradient(x, y)
    }
  }
  c <- 1
}

func GenerateMap(hFreq, mFreq float64, width, height, numCPUs int) {
  rand.Seed(time.Now().UTC().UnixNano())
  hSeed := rand.Int63()
  mSeed := rand.Int63()
  fmt.Println("height seed:", hSeed)
  fmt.Println("moisture seed:", mSeed)
  hNoise := opensimplex.NewWithSeed(hSeed)
  mNoise := opensimplex.NewWithSeed(mSeed)

  world := CreateWorld(width, height, hFreq, mFreq, 10)

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

  c = make(chan int, numCPUs + 1)
  go world.CalcGradient(c);

  for i := 0; i < numCPUs; i++ {
    go world.CalcBiome(i * width / numCPUs,
                          (i + 1) * width / numCPUs,
                          mNoise, c)
  }
  for i := 0; i < numCPUs + 1; i++ {
    <-c
  }
  world.AddRivers()

  fmt.Println("Duration: ", time.Now().Sub(start));
  fmt.Println("Number of peaks: ", len(world.peaks));
  fmt.Println("Numer of lakes: ", len(world.lakes));

  img := image.NewRGBA(image.Rect(0, 0, width, height))
  colours := [15]color.RGBA{ { 51, 166, 204, 255 },  // OCEAN
                            { 255, 230, 128, 255 }, // BEACH
                            { 153, 153, 102, 255 }, // SCORCHED
                            { 204, 204, 204, 255 }, // BARE
                            { 230, 184, 0, 255 },  // TUNDRA
                            { 230, 230, 230, 255 }, // SNOW
                            { 170, 190, 50, 255 },  // TEMPERATURE_DESERT
                            { 166, 204, 51, 255 },  // SHRUBLAND
                            { 51, 153, 77, 255 },   // TAIGA
                            { 128, 153, 51, 255 },  // GRASSLAND
                            { 96, 159, 96, 255 },   // TEMPERATE_DECID
                            { 77, 153, 0, 255 },    // TEMPERATE_RAIN
                            { 255, 230, 128, 255 }, // SUBTROPICAL_DESERT
                            { 102, 153, 0, 255 },   // TROPICAL_SEASONAL
                            { 85, 128, 0, 255 } }    // TROPICAL_RAIN

  bounds := img.Bounds()
  for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
    for x := bounds.Min.X; x < bounds.Max.X; x++ {
      img.Set(x, y, colours[world.Biome(x, y)])
    }
  }

  filename := "h" + strconv.FormatInt(hSeed, 10) + "-" +
              "m" + strconv.FormatInt(mSeed, 10) + ".png"
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
  width := flag.Int("width", 4000, "map width")
  height := flag.Int("height", 4000, "map height")
  hFreq := flag.Float64("hfreq", 5, "height noise frequency")
  mFreq := flag.Float64("mfreq", 2, "moisture noise frequency")
  threads := flag.Int("cpus", runtime.NumCPU(), "number of cores to use")

  flag.Parse()

  fmt.Println("width, height, threads")
  fmt.Println(*width, ",", *height, ",", *threads)
  GenerateMap(*hFreq, *mFreq, *width, *height, *threads)
}
