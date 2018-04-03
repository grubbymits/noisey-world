package main

import (
  "encoding/json"
  "log"
  "os"
)

type ExportLoc struct {
  X, Y int
  Blocked bool
}

type ExportWorld struct {
  Width, Height int
  Locations []ExportLoc
}

func ExportJSON(w *World) {
  file, err := os.Create("world.json")
  if err != nil {
    log.Fatal(err)
  }
  defer file.Close()

  exportWorld := new(ExportWorld)
  exportWorld.Width = w.width
  exportWorld.Height = w.height
  exportWorld.Locations = make([]ExportLoc, w.width * w.height)

  for y := 0; y < w.height; y++ {
    for x := 0; x < w.width; x++ {
      loc := w.Location(x, y)
      blocked := (loc.isRiver || loc.isWall || loc.biome == OCEAN ||
                  loc.hasFeature(ROCK_FEATURE) ||
                  loc.hasFeature(TREE_FEATURE))
      exportWorld.Locations[y * w.width + x] = ExportLoc{ loc.x, loc.y, blocked }
    }
  }
  enc := json.NewEncoder(file)
  if err := enc.Encode(exportWorld); err != nil {
    log.Println(err)
  }
}
