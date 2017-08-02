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

type RiverNode struct {
  preds, succs []RiverNode
}

type WorldMap struct {
  width, height int
  heightMap []float64
  biomeMap []uint8
  gradientMap []RiverNode
  hFreq, mFreq float64
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

func (world WorldMap) CalcGradient(xBegin, xEnd int, c chan int) {
  width := world.width
  height := world.height

  for y := 0; y < height; y++ {
    for x := xBegin; x < xEnd; x++ {
      startHeight := world.heightMap[(y * width) + x]
      maxGradient := 0.0
      nextIndex := 0
      
      for y2 := y - 1; y2 < y + 1; y2++ {
        if y2 < 0 || y2 >= height {
          continue
        }
        for x2 := x - 1; x2 < x + 1; x2++ {
          if x2 < 0 || x >= width {
            continue
          }
          nextHeight := world.heightMap[(y2 * width + x2)]
          if nextHeight > startHeight {
            continue
          }
          newGrad := math.Abs(startHeight - nextHeight)
          if newGrad > maxGradient {
            maxGradient = newGrad
            nextIndex = y2 * width + x2
          }
        }
      }
      world.gradientMap[y * width + x] = nextIndex
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
      world.heightMap[(y * width) + x] = h
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
      world.biomeMap[(y * width) + x] = biome(world.heightMap[(y * width) + x], m)
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
                     make([]float64, width * height),
                     make([]uint8, width * height),
                     make([]int, width * height),
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
      img.Set(x, y, colours[world.biomeMap[(y * width) + x]])
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
