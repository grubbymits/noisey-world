package main

type Cloud struct {
  moisture float64
  direction uint
  loc *Location
  world *World
}

const RAIN = 0.5

func CreateCloud(moisture float64, direction uint, loc *Location,
                 world *World) *Cloud {
  c := new(Cloud)
  c.moisture = moisture
  c.direction = direction
  c.loc = loc
  c.world = world
  return c
}

// Return whether this cloud no longer needs updating.
func (c *Cloud) update() bool {

  // Cloud has dried up.
  if c.moisture <= 0 {
    return true
  }

  nextLoc := c.world.getDirectedLocation(c.loc, c.direction)

  // Reached the end of map.
  if nextLoc == nil {
    return true
  }

  // Don't start 'raining' until the cloud gets to land.
  if nextLoc.height < RAIN_LEVEL {
    c.loc = nextLoc
    return false
  }

  multiplier := nextLoc.height / c.loc.height
  total := RAIN * multiplier

  // Treat terraces as obsticles that will cause the cloud to split into
  // multiple clouds, with a maximum of two new clouds, each taking some of the
  // moisture. Each new cloud will travel in a different direction.
  if nextLoc.terrace > c.loc.terrace {
    c.world.addCloud(c, (c.direction + 1) % MAX_DIR)
    c.world.addCloud(c, (c.direction - 1) % MAX_DIR)
    c.moisture /= 3
    multiplier *= 2
  }

  if c.moisture < total {
    nextLoc.moisture += c.moisture
    c.moisture = 0
  } else {
    // Dissipate some moisture to the land.
    nextLoc.moisture += total
    c.moisture -= total
  }
  c.loc = nextLoc
  return false
}
