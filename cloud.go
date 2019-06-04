package main

type Cloud struct {
  moisture float64
  direction uint32
  loc *Location
  world *World
}

func CreateCloud(moisture float64, direction uint32, loc *Location,
                 world *World) *Cloud {
  c := new(Cloud)
  c.moisture = moisture
  c.direction = direction
  c.loc = loc
  c.world = world
  return c
}

func (c Cloud) update() bool {

  // Cloud has dried up.
  if c.moisture < 0 {
    return false
  }

  nextLoc := c.world.getDirectedLocation(c.loc, c.direction)

  // Reached the end of map.
  if nextLoc == nil {
    return false
  }

  // Don't start 'raining' until the cloud gets to land.
  if nextLoc.biome == OCEAN {
    c.loc = nextLoc
    return true
  }

  // Treat terraces as obsticles that will cause the cloud to split into
  // multiple clouds, with a maximum of two new clouds, each taking some of the
  // moisture. Each new cloud will travel in a different direction.
  if nextLoc.terrace != c.loc.terrace {
    c.world.addCloud(&c, (c.direction + 1) % MAX_DIR)
    c.world.addCloud(&c, (c.direction - 1) % MAX_DIR)
  }

  // Dissipate some moisture to the land.
  nextLoc.moisture += 0.5
  c.moisture -= 0.5
  c.loc = nextLoc
  return true
}
