package main

import (
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

func biome(h, m float64) uint8 {      
  if (h < -0.4) {
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

type GradNode struct {
  preds, succs []*GradNode
  weight int
}

func (n GradNode) finalise() {
  for _, pred := range n.preds {
    n.weight += (pred.weight / len(pred.succs))
  }
}

type Location struct {
  height, moisture float64
  biome uint8
  node GradNode
}

func (l Location) addSuccessor(other *Location) {
  l.node.succs = append(l.node.succs, &other.node)
}

func (l Location) addPredecessor(other *Location) {
  l.node.preds = append(l.node.preds, &other.node)
}

type WorldMap struct {
  width, height int
  locations []Location
  gradientMap []GradNode
  hFreq, mFreq float64
}

func (w WorldMap) Location(x, y int) *Location {
  return &w.locations[y * w.width + x]
}

func (w WorldMap) Moisture(x, y int) float64 {
  return w.locations[y * w.width + x].moisture
}

func (w WorldMap) Height(x, y int) float64 {
  return w.locations[y * w.width + x].height
}

func (w WorldMap) Biome(x, y int) uint8 {
  return w.locations[y * w.width + x].biome
}

func (w WorldMap) SetMoisture(x, y int, m float64) {
  w.locations[y * w.width + x].moisture = m
}

func (w WorldMap) SetHeight(x, y int, h float64) {
  w.locations[y * w.width + x].height = h
}

func (w WorldMap) SetBiome(x, y int, b uint8) {
  w.locations[y * w.width + x].biome = b
}
/*
func (world WorldMap) AddRivers() {
  width := world.width
  height := world.height

  for y := 0; y < height; y++ {
    for x := 0; x < width; x++ {
      startHeight := world.heightMap[(y * width) + x]
      if startHeight > 0.7
    }
  }
}*/

func (w WorldMap) CalcGradient(xBegin, xEnd int, c chan int) {
  width := w.width
  height := w.height

  for y := 0; y < height; y++ {
    for x := xBegin; x < xEnd; x++ {

      centreLoc := w.Location(x, y)

      if y - 1 >= 0 {
        otherLoc := w.Location(x, y - 1)
        if otherLoc.height < centreLoc.height {
          centreLoc.addSuccessor(otherLoc)
        } else if otherLoc.height > centreLoc.height {
          centreLoc.addPredecessor(otherLoc)
        }
      }

      if x + 1 < width {
        otherLoc := w.Location(x + 1, y)
        if otherLoc.height < centreLoc.height {
          centreLoc.addSuccessor(otherLoc)
        } else if otherLoc.height > centreLoc.height {
          centreLoc.addPredecessor(otherLoc)
        }
      }

      if y + 1 < height {
        otherLoc := w.Location(x, y + 1)
        if otherLoc.height < centreLoc.height {
          centreLoc.addSuccessor(otherLoc)
        }   else if otherLoc.height > centreLoc.height {
          centreLoc.addPredecessor(otherLoc)
        }
      }

      if x - 1 >= 0 {
        otherLoc := w.Location(x - 1, y)
        if otherLoc.height < centreLoc.height {
          centreLoc.addSuccessor(otherLoc)
        } else if otherLoc.height > centreLoc.height {
          centreLoc.addPredecessor(otherLoc)
        }
      }
      centreLoc.node.finalise()
    }
  }
  c <- 1
}

func (world WorldMap) CalcHeight(xBegin, xEnd int,
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

func (world WorldMap) CalcBiome(xBegin, xEnd int,
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

  world := WorldMap{ width, height,
                     make([]Location, width * height),
                     make([]GradNode, width * height),
                     hFreq, mFreq }

  start := time.Now()
  c := make(chan int, numCPUs)
  for i := 0; i < numCPUs; i++ {
    go world.CalcHeight(i * width / numCPUs,
                            (i + 1) * width / numCPUs,
                            hNoise, c)
  }
  for i := 0; i < numCPUs; i++ {
    <-c
  }
  for i := 0; i < numCPUs; i++ {
    go world.CalcGradient(i * width / numCPUs,
                          (i + 1) * width / numCPUs,
                          c)
  }
  for i := 0; i < numCPUs; i++ {
    <-c
  }
  for i := 0; i < numCPUs; i++ {
    go world.CalcBiome(i * width / numCPUs,
                          (i + 1) * width / numCPUs,
                          mNoise, c)
  }
  for i := 0; i < numCPUs; i++ {
    <-c
  }
  fmt.Println("Duration: ", time.Now().Sub(start))

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

  // split the map into parts, select the highest point

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
  const width, height = 4000, 4000
  const hFreq, mFreq = 5, 2
  fmt.Println("CPUs: ", runtime.NumCPU())
  GenerateMap(hFreq, mFreq, width, height, runtime.NumCPU())
}
