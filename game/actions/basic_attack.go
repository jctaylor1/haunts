package actions

import (
  "encoding/gob"
  "path/filepath"
  "github.com/runningwild/glop/gin"
  "github.com/runningwild/glop/sprite"
  "github.com/runningwild/glop/gui"
  "github.com/runningwild/haunts/base"
  "github.com/runningwild/haunts/sound"
  "github.com/runningwild/haunts/game"
  "github.com/runningwild/haunts/game/status"
  "github.com/runningwild/haunts/texture"
  "github.com/runningwild/opengl/gl"
)

func registerBasicAttacks() map[string]func() game.Action {
  attack_actions := make(map[string]*BasicAttackDef)
  base.RemoveRegistry("actions-attack_actions")
  base.RegisterRegistry("actions-attack_actions", attack_actions)
  base.RegisterAllObjectsInDir("actions-attack_actions", filepath.Join(base.GetDataDir(), "actions", "basic_attacks"), ".json", "json")
  makers := make(map[string]func() game.Action)
  for name := range attack_actions {
    cname := name
    makers[cname] = func() game.Action {
      a := BasicAttack{ Defname: cname }
      base.GetObject("actions-attack_actions", &a)
      if a.Ammo > 0 {
        a.Current_ammo = a.Ammo
      } else {
        a.Current_ammo = -1
      }
      return &a
    }
  }
  return makers
}

func init() {
  game.RegisterActionMakers(registerBasicAttacks)
  gob.Register(&BasicAttack{})
}

// Basic Attacks are single target and instant, they are also readyable
type BasicAttack struct {
  Defname string
  *BasicAttackDef
  basicAttackTempData

  Current_ammo int
}
type BasicAttackDef struct {
  Name       string
  Kind       status.Kind
  Ap         int
  Ammo       int  // 0 = infinity
  Strength   int
  Range      int
  Damage     int
  Animation  string
  Conditions []string
  Texture    texture.Object
  Sounds     map[string]string
}
type basicAttackTempData struct {
  ent *game.Entity

  // Potential targets
  targets []*game.Entity

  // The selected target for the attack
  target *game.Entity
}

type basicAttackExec struct {
  game.BasicActionExec
  Target game.EntityId
}
func init() {
  gob.Register(basicAttackExec{})
}

func dist(x,y,x2,y2 int) int {
  dx := x - x2
  if dx < 0 { dx = -dx }
  dy := y - y2
  if dy < 0 { dy = -dy }
  if dx > dy {
    return dx
  }
  return dy
}
func (a *BasicAttack) AP() int {
  return a.Ap
}
func (a *BasicAttack) Pos() (int, int) {
  return 0, 0
}
func (a *BasicAttack) Dims() (int, int) {
  return 0, 0
}
func (a *BasicAttack) String() string {
  return a.Name
}
func (a *BasicAttack) Icon() *texture.Object {
  return &a.Texture
}
func (a *BasicAttack) Readyable() bool {
  return true
}
func (a *BasicAttack) findTargets(ent *game.Entity, g *game.Game) []*game.Entity {
  var targets []*game.Entity
  x,y := ent.Pos()
  for _,e := range g.Ents {
    if e == ent { continue }
    if e.Stats == nil { continue }
    x2,y2 := e.Pos()
    if dist(x, y, x2, y2) <= a.Range && ent.HasLos(x2, y2, 1, 1) && e.Stats.HpCur() > 0 {
      targets = append(targets, e)
    }
  }
  return targets
}
func (a *BasicAttack) Preppable(ent *game.Entity, g *game.Game) bool {
  return a.Current_ammo != 0 && ent.Stats.ApCur() >= a.Ap && len(a.findTargets(ent, g)) > 0
}
func (a *BasicAttack) Prep(ent *game.Entity, g *game.Game) bool {
  if !a.Preppable(ent, g) {
    return false
  }
  a.ent = ent
  a.targets = a.findTargets(ent, g)
  if a.Sounds != nil {
    sound.MapSounds(a.Sounds)
  }
  return true
}
func (a *BasicAttack) AiAttackTarget(ent *game.Entity, target *game.Entity) game.ActionExec {
  if ent.Side() == target.Side() { return nil }
  if ent.Stats.ApCur() < a.Ap { return nil }
  x,y := ent.Pos()
  x2,y2 := target.Pos()
  if dist(x,y,x2,y2) > a.Range { return nil }
  return a.makeExec(ent, target)
}
func (a *BasicAttack) makeExec(ent, target *game.Entity) basicAttackExec {
  var exec basicAttackExec
  exec.SetBasicData(ent, a)
  exec.Target = target.Id
  return exec
}
func (a *BasicAttack) HandleInput(group gui.EventGroup, g *game.Game) (bool, game.ActionExec) {
  target := g.HoveredEnt()
  if target == nil { return false, nil }
  if target.Stats == nil { return false, nil }
  if found,event := group.FindEvent(gin.MouseLButton); found && event.Type == gin.Press {
    px, py := target.Pos()
    if a.ent.Stats.ApCur() >= a.Ap && target.Stats.HpCur() > 0 && a.ent.HasLos(px, py, 1, 1) {
      return true, a.makeExec(a.ent, target)
    }
    return true, nil
  }
  return false, nil
}
func (a *BasicAttack) RenderOnFloor() {
  gl.Disable(gl.TEXTURE_2D)
  gl.Begin(gl.QUADS)
  gl.Color4d(1.0, 0.2, 0.2, 0.8)
  for _,ent := range a.targets {
    ix,iy := ent.Pos()
    x := float64(ix)
    y := float64(iy)
    gl.Vertex2d(x + 0, y + 0)
    gl.Vertex2d(x + 0, y + 1)
    gl.Vertex2d(x + 1, y + 1)
    gl.Vertex2d(x + 1, y + 0)
  }
  gl.End()
}
func (a *BasicAttack) Cancel() {
  a.basicAttackTempData = basicAttackTempData{}
}
func (a *BasicAttack) Maintain(dt int64, g *game.Game, ae game.ActionExec) game.MaintenanceStatus {
  base.Log().Printf("Maintain: %v", ae)
  if ae != nil {
    exec := ae.(basicAttackExec)
    a.ent = g.EntityById(ae.EntityId())
    a.target = a.ent.Game().EntityById(exec.Target)

    // Track this information for the ais
    a.ent.Info.LastEntThatIAttacked = a.target.Id
    a.target.Info.LastEntThatAttackedMe = a.ent.Id

    if a.Ap > a.ent.Stats.ApCur() {
      base.Error().Printf("Got a basic attack that required more ap than available: %v", exec)
      return game.Complete
    }

    if a.target.Stats.HpCur() <= 0 {
      base.Error().Printf("Got a basic attack that attacked a dead person: %v", exec)
      return game.Complete
    }

    if distBetweenEnts(a.ent, a.target) > a.Range {
      base.Error().Printf("Got a basic attack that is out of range: %v", exec)
      return game.Complete
    }
  }
  if a.ent.Sprite().State() == "ready" && a.target.Sprite().State() == "ready" {
    a.target.TurnToFace(a.ent.Pos())
    a.ent.TurnToFace(a.target.Pos())
    if a.Current_ammo > 0 {
      a.Current_ammo--
    }
    a.ent.Stats.ApplyDamage(-a.Ap, 0, status.Unspecified)
    var defender_cmds []string
    if game.DoAttack(a.ent, a.target, a.Strength, a.Kind) {
      for _,name := range a.Conditions {
        a.target.Stats.ApplyCondition(status.MakeCondition(name))
      }
      a.target.Stats.ApplyDamage(0, -a.Damage, a.Kind)
      if a.target.Stats.HpCur() <= 0 {
        defender_cmds = []string{"defend", "killed"}
      } else {
        defender_cmds = []string{"defend", "damaged"}
      }
    } else {
      defender_cmds = []string{"defend", "undamaged"}
    }
    sprites := []*sprite.Sprite{a.ent.Sprite(), a.target.Sprite()}
    sprite.CommandSync(sprites, [][]string{[]string{a.Animation}, defender_cmds}, "hit")
    return game.Complete
  }
  return game.InProgress
}
func (a *BasicAttack) Interrupt() bool {
  return true
}

