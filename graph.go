package main

import (
  "math"
)

type GraphNode struct {
  neighbours [4]*GraphNode
  //cost map[*GraphNode]int
  numNeighbours int
  loc *Location
}

type SortableNode struct {
  node *GraphNode
  cost float64
}

type NodeQueue []*SortableNode

func (queue NodeQueue) Len() int {
  return len(queue)
}

func (queue NodeQueue) Less(i, j int) bool {
  return queue[i].cost < queue[j].cost
}

func (queue NodeQueue) Swap(i, j int) {
  queue[i], queue[j] = queue[j], queue[i]
}

type Graph struct {
  w *World
  nodes []GraphNode
}

func CreateGraph(w *World) *Graph {
  g := new(Graph)
  g.w = w
  g.nodes = make([]GraphNode, w.width * w.height)
  for y := 0; y < w.height; y++ {
    for x := 0; x < w.width; x++ {
      loc := w.Location(x, y)
      node := g.getNode(loc)
      node.loc = loc
      g.populateNeighbours(loc)
    }
  }
  return g
}

func (g *Graph) cost(from, to *GraphNode) float64 {
  loc0 := from.loc
  loc1 := to.loc
  cost := 1.0
  factor := 1.0
  cost += factor * math.Abs(float64(loc0.height - loc1.height))
  return cost
}

func (g *Graph) getNode(loc *Location) *GraphNode {
  return &g.nodes[loc.y  * g.w.width + loc.x]
}

func (g *Graph) getNumNeighbours(loc *Location) int {
  return g.getNode(loc).numNeighbours
}

func (g *Graph) getNeighbours(loc *Location) [4]*GraphNode {
  return g.getNode(loc).neighbours
}

func (g *Graph) populateNeighbours(loc *Location) {
  idx := 0
  w := g.w
  node := g.getNode(loc)

  if loc.isWall {
    neighbour := w.Location(loc.x, loc.y - 1)
    node.numNeighbours = 0

    if neighbour.isRiver || neighbour.isRiverBank ||
       neighbour.hasFeature(TREE_FEATURE) || neighbour.hasFeature(ROCK_FEATURE) {
      return
    }
    nodeNeighbour := g.getNode(neighbour)
    node.neighbours[0] = nodeNeighbour
    node.numNeighbours = 1
    return
  }

  for x := -1; x < 2; x++ {
    for y := -1; y < 2; y++ {
      // is loc
      if x == 0 && y == 0 {
        continue
      }
      // don't consider diagonal neighbours
      if x != 0 && y != 0 {
        continue
      }
      // out of range
      if loc.x + x < 0 || loc.x + x >= w.width ||
         loc.y + y < 0 || loc.y + y >= w.height {
        continue
      }
      neighbour := w.Location(loc.x + x, loc.y + y)

      if neighbour.isRiver || neighbour.isRiverBank {
        continue
      }
      if neighbour.hasFeature(TREE_FEATURE) || neighbour.hasFeature(ROCK_FEATURE) {
        continue
      }

      // Only cross terraces when moving north
      if neighbour.terrace != loc.terrace && y != -1 {
        continue
      }

      // only allow paths up terraces where there's a section of wall that is
      // 3 tiles wide.
      if neighbour.isWall &&
         !(w.Location(neighbour.x - 1, neighbour.y).isWall &&
           w.Location(neighbour.x + 1, neighbour.y).isWall) {
        continue
      }

      nodeNeighbour := g.getNode(neighbour)
      node.neighbours[idx] = nodeNeighbour
      idx++
    }
  }
  node.numNeighbours = idx
}

