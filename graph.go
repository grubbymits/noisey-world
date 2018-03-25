package main

import (
  "math"
)

type GraphNode struct {
  neighbours [4]*GraphNode
  cost map[*GraphNode]int
  numNeighbours int
  loc *Location
}

type SortableNode struct {
  node *GraphNode
  cost int
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
      node.cost = make(map[*GraphNode] int)
      node.loc = loc
      g.populateNeighbours(loc)
    }
  }
  return g
}

func (g *Graph) cost(loc0, loc1 *Location) int {
  node0 := g.getNode(loc0)
  node1 := g.getNode(loc1)
  return node0.cost[node1]
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
      if loc.isRiver || loc.isRiverBank {
        continue
      }
      if loc.hasFeature(TREE_FEATURE) || loc.hasFeature(ROCK_FEATURE) {
        continue
      }
      cost := 1
      neighbour := w.Location(loc.x + x, loc.y + y)
      // favour a terrace traversal being up and down a wall tile.
      if neighbour.isWall {
        cost += 50
      } else if neighbour.terrace != loc.terrace {
        cost += 100
      }
      cost += int(10 *math.Abs(float64(loc.height - neighbour.height)))
      nodeNeighbour := g.getNode(neighbour)
      node.neighbours[idx] = nodeNeighbour
      node.cost[nodeNeighbour] = cost
      idx++
    }
  }
  node.numNeighbours = idx
}

