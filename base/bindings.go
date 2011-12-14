package base

import (
  "glop/gin"
  "strings"
  "fmt"
)

type KeyBinds map[string]string
type KeyMap map[string]gin.Key

func getKeysFromString(str string) []gin.KeyId {
  parts := strings.Split(str, "+")
  var kids []gin.KeyId
  for _,part := range parts {
    var kid gin.KeyId
    switch {
    case len(part) == 1:  // Single character - should be ascii
      kid = gin.KeyId(part[0])

    case part == "ctrl":
      kid = gin.EitherControl

    case part == "shift":
      kid = gin.EitherShift

    case part == "alt":
      kid = gin.EitherAlt

    case part == "gui":
      kid = gin.EitherGui
    }
    kids = append(kids, kid)
  }
  return kids
}

func (kb KeyBinds) MakeKeyMap() KeyMap {
  key_map := make(KeyMap)
  for key,val := range kb {
    kids := getKeysFromString(val)

    fmt.Printf("key: %v\n", key)
    fmt.Printf("val: %v\n", val)
    fmt.Printf("kids: %v\n", kids)
    if len(kids) == 1 {
      key_map[key] = gin.In().GetKey(kids[0])
    } else {
      // The last kid is the main kid and the rest are modifiers
      main := kids[len(kids) - 1]
      kids = kids[0 : len(kids) - 1]
      var down []bool
      for _ = range kids {
        down = append(down, true)
      }
      key_map[key] = gin.In().BindDerivedKey(key, gin.In().MakeBinding(main, kids, down))
    }
  }
  return key_map
}
