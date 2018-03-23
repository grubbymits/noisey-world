package main

const (
  EMPTY = 0
  TREE_FEATURE = 1
  ROCK_FEATURE = 1 << 1
  PLANT_FEATURE = 1 << 2

  RIGHT_SHADOW_FEATURE = 1 << 3
  HORIZONTAL_SHADOW_FEATURE = 1 << 4
  LEFT_SHADOW_FEATURE = 1 << 5
  BOTTOM_LEFT_SHADOW_FEATURE = 1 << 6
  BOTTOM_RIGHT_SHADOW_FEATURE = 1 << 7
  LEFT_WATER_SHADOW_FEATURE = 1 << 8
  RIGHT_WATER_SHADOW_FEATURE = 1 << 9

  GROUND_FEATURE = 1 << 10
)

const (
  TOP_LEFT_RIVER_FEATURE = iota
  TOP_RIVER_FEATURE
  TOP_RIGHT_RIVER_FEATURE
  LEFT_RIVER_FEATURE
  RIGHT_RIVER_FEATURE
  BOTTOM_LEFT_RIVER_FEATURE
  BOTTOM_RIVER_FEATURE
  BOTTOM_RIGHT_RIVER_FEATURE
)

const (
  OCEAN = iota
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
  BIOMES
)

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

  if h < WATER_LEVEL {
    return OCEAN
  } else if h < BEACH_LEVEL {
    return BEACH
  }

  // No soil
  if s < NO_SOIL {
    if m < DRY {
      return DRY_ROCK
    }
    return MOIST_ROCK
  }

  if h > HIGHLANDS {
    if s > THICK_SOIL {
      if m > WET {
        return MOORLAND
      } else if m > MOIST {
        return SHRUBLAND
      } else {
        return GRASSLAND
      }
    } else if s > SHALLOW_SOIL {
      if m > WET {
        return WOODLAND
      } else if m > MOIST {
        return SHRUBLAND
      } else {
        return GRASSLAND
      }
    } else {
      return GRASSLAND
    }
  } else if h > MIDLANDS {
    if s > THICK_SOIL {
      if m > WET {
        return FOREST
      } else if m > MOIST {
        return WOODLAND
      } else {
        return SHRUBLAND
      }
    } else if s > SHALLOW_SOIL {
      if m > WET {
        return WOODLAND
      } else if m > MOIST {
        return SHRUBLAND
      } else {
        return GRASSLAND
      }
    } else if m > WET {
      return SHRUBLAND
    } else {
      return GRASSLAND
    }
  } else if s > THICK_SOIL {  // lowlands
    if m > WET {
      return FENLAND
    } else if m > MOIST {
      return FOREST
    } else {
      return WOODLAND
    }
  } else if s > SHALLOW_SOIL {
    if m > WET {
      return WOODLAND
    } else if m > MOIST {
      return SHRUBLAND
    } else {
      return HEATHLAND
    }
  } else {
    if m > WET {
      return SHRUBLAND
    } else if m > MOIST {
      return GRASSLAND
    } else {
      return HEATHLAND
    }
  }
}

type Location struct {
  height, moisture, soilDepth, tree, rock, plant, water float64
  preds, succs [4]*Location
  numPreds, numSuccs int
  totalGradient float64
  discovered, weight int
  x, y int
  biome, nearbyBiome, terrace uint8
  features uint
  isRiverBank bool
  isRiver bool
  isWall bool
  riverBank uint
}

func (l *Location) addSuccessor(other *Location) {
  l.succs[l.numSuccs] = other
  l.numSuccs++
}

func (l *Location) addPredecessor(other *Location) {
  l.preds[l.numPreds] = other
  l.numPreds = l.numPreds + 1
}

func (l *Location) addFeature(feat uint) {
  l.features |= feat
}

func (l *Location) hasFeature(feat uint) bool {
  return feat & l.features == feat
}

func (l *Location) setRiverBank(feat uint) {
  l.isRiverBank = true
  l.riverBank = feat
}

type LocVal struct {
  index, x, y int
  val float64
}

type LocMaxHeap []*LocVal

func (lmh LocMaxHeap) Len() int { return len(lmh) }

func (lmh LocMaxHeap) Less(i, j int) bool {
  // We want Pop to give us the highest, not lowest, priority so we use
  // greater than here.
  return lmh[i].val > lmh[j].val
}

func (lmh LocMaxHeap) Swap(i, j int) {
  lmh[i], lmh[j] = lmh[j], lmh[i]
  lmh[i].index = i
  lmh[j].index = j
}

func (lmh *LocMaxHeap) Push(x interface{}) {
  n := len(*lmh)
  item := x.(*LocVal)
  item.index = n
  *lmh = append(*lmh, item)
}

func (lmh *LocMaxHeap) Pop() interface{} {
  old := *lmh
  n := len(old)
  item := old[n-1]
  item.index = -1 // for safety
  *lmh = old[0 : n-1]
  return item
}

