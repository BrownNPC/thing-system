package main

import (
	"log/slog"
	"math"
	"math/rand/v2"
	"os"
	"strconv"

	ts "github.com/BrownNPC/thing-system"
	rl "github.com/gen2brain/raylib-go/raylib"
	"github.com/lmittmann/tint"
)

type Kind int

const (
	KindNil = iota
	KindBall
	KindPaddle
)

type Things = *ts.Things[Thing]
type Thing struct {
	Kind Kind

	Position      rl.Vector2
	Width, Height float32
	Velocity      rl.Vector2

	Speed    float32
	MaxSpeed float32

	Color rl.Color

	Score             int
	ScoreTextPosition rl.Vector2
	IsLeftPaddle      bool
}

// world size.
// We don't work with screen coordinates.
const Width, Height = 160, 90

type BallConfig struct {
	Speed    float32
	Height   float32
	MaxSpeed float32
	Color    rl.Color
}

func spawnBall(things Things, cfg BallConfig, towardsLeft bool) ts.ThingRef {
	spawnPosition := rl.NewVector2(Width*.5, Height*.5)
	directionX := float32(1.0)
	if towardsLeft {
		directionX = -1
	}

	// pick a random target Y somewhere in the screen
	targetY := float32(rand.IntN(Height))

	// along X, at random Y
	targetDirection := rl.NewVector2(spawnPosition.X+directionX*100, targetY)

	// compute velocity
	velocity := targetDirection.Subtract(spawnPosition).Normalize().Scale(cfg.Speed)
	return things.New(Thing{
		Kind:     KindBall,
		Position: spawnPosition,
		Velocity: velocity,
		Color:    cfg.Color,
		Speed:    max(1, cfg.Speed),
		MaxSpeed: max(cfg.Speed, cfg.MaxSpeed),
		Height:   max(1, cfg.Height), //px
	})
}
func spawnPaddle(things Things, isLeft bool, color rl.Color) ts.ThingRef {
	const width float32 = 4
	// center
	position := rl.NewVector2(width, Height).Scale(.5)
	if !isLeft {
		position.X = Width - (width / 2)
	}

	scoreTextPosition := position
	scoreTextPosition.Y -= 13
	if isLeft {
		scoreTextPosition.X += 30
	} else {
		scoreTextPosition.X -= 30
	}

	return things.New(Thing{
		Kind:              KindPaddle,
		Width:             width,
		Height:            25,
		Color:             color,
		Speed:             250,
		MaxSpeed:          400,
		IsLeftPaddle:      isLeft,
		Position:          position,
		ScoreTextPosition: scoreTextPosition,
	})
}

var Logger = slog.New(tint.NewHandler(os.Stderr, &tint.Options{AddSource: true}))

func main() {
	var things Things = ts.NewThings(10, Thing{
		// nil Things will be of color purple.
		// Makes it easy to spot nil dereferences without reading the logs.
		Color: rl.DarkPurple,
	})
	// Initialize resizable window.
	rl.SetConfigFlags(rl.FlagWindowResizable)
	rl.InitWindow(0, 0, "Pong")
	spawnPaddle(things, true, rl.Blue)
	spawnPaddle(things, false, rl.Red)
	rl.SetTargetFPS(60)
	// surface to draw things on.
	// It gets fractionally scaled.
	surface := rl.LoadRenderTexture(Width, Height)
	const tickRate = 1.0 / 60
	var accumulator float32 = rl.GetFrameTime()
	// Event loop.
	var state = PersistantState{RespawnCooldown: 1,
		RespawnCooldownTimer: 1,
		State:                GameStateRespawning,
		BallConfig: BallConfig{
			Speed:    150,
			MaxSpeed: 300,
			Height:   8,
			Color:    rl.Red,
		}}
	for !rl.WindowShouldClose() {
		rl.BeginDrawing()
		// gray-ish color background
		rl.ClearBackground(rl.GetColor(0x211919FF))
		{ // draw on the surface in world coordinates.
			rl.BeginTextureMode(surface)
			rl.ClearBackground(rl.Blank)
			accumulator += rl.GetFrameTime()
			for accumulator > tickRate {
				UpdateThings(tickRate, things, &state)
				accumulator -= tickRate
			}
			DrawThings(things)
			rl.EndTextureMode()
			// apply fractional scaling to fit screen size.
			DrawRenderTextureScaled(surface)
		}
		rl.EndDrawing()
	}
}

type GameState int

const (
	GameStatePlaying GameState = iota

	GameStateRespawning // ball is respawning
)

// State that persists during updates.
type PersistantState struct {
	State GameState
	//CONFIG
	RespawnCooldown float32 // how long to wait before respawning ball in seconds
	BallConfig      BallConfig

	// STATE TRACKER
	RespawnCooldownTimer float32
}

// DrawThings modifeis things.
func UpdateThings(dt float32, things Things, state *PersistantState) {
	ballRef := PhysicsSystem(things, dt)
	switch state.State {
	case GameStatePlaying:
		paddles := things.Filter(func(t *Thing) bool {
			return t.Kind == KindPaddle
		})
		// Physics system modifies velocity of Things
		paddleLeft := paddles[0]
		paddleRight := paddles[1]
		ball := things.Get(ballRef)
		// collided with right side of map (goal)
		if ball.Position.X > Width+ball.Height {
			defer things.Delete(ballRef)
			paddleLeft.Score += 1
			state.State = GameStateRespawning
			ball.Velocity = ball.Velocity.Scale(1.01)
			// collided with left side of map (goal)
		} else if ball.Position.X < -ball.Height {
			defer things.Delete(ballRef)
			paddleRight.Score += 1
			state.State = GameStateRespawning
			ball.Velocity = ball.Velocity.Scale(1.01)
		}
		// collide with paddles
		for _, paddle := range paddles {
			size := rl.NewVector2(paddle.Width, paddle.Height)
			topLeft := paddle.Position.Subtract(size.Scale(0.5))
			rect := rl.NewRectangle(topLeft.X, topLeft.Y, paddle.Width, paddle.Height)

			if normal, collide := CircleRectCollisionNormal(ball.Position, ball.Height/2, rect); collide {
				penetration := ball.Height/2 - DistanceToRect(ball.Position, rect) // function that returns distance from circle center to closest point
				ball.Position = ball.Position.Add(normal.Scale(penetration))

				ball.Velocity = ball.Velocity.Reflect(normal).Scale(1.1)
			}
		}
		// end case GameStatePlaying
	case GameStateRespawning:
		state.RespawnCooldownTimer -= dt
		if state.RespawnCooldownTimer <= 0 {
			state.RespawnCooldownTimer = state.RespawnCooldown
			state.State = GameStatePlaying

			paddles := things.Filter(func(t *Thing) bool {
				return t.Kind == KindPaddle
			})

			winningPaddle := paddles[0]
			if paddles[1].Score > paddles[0].Score {
				winningPaddle = paddles[1]
			}

			spawnBall(things, state.BallConfig, winningPaddle.IsLeftPaddle)
		}
	}

	paddles := things.Filter(func(t *Thing) bool {
		return t.Kind == KindPaddle
	})
	paddleLeft := paddles[0]
	paddleRight := paddles[1]

	{ // PADDLE LEFT CONTROLS
		if rl.IsKeyPressed(rl.KeyW) { //up
			paddleLeft.Velocity.Y = -paddleLeft.Speed
		}
		if rl.IsKeyPressed(rl.KeyS) { //down
			paddleLeft.Velocity.Y = paddleLeft.Speed
		}
		// stop when released
		if rl.IsKeyUp(rl.KeyW) && rl.IsKeyUp(rl.KeyS) {
			paddleLeft.Velocity.Y = 0
		}
	}
	{ // PADDLE RIGHT CONTROLS
		if rl.IsKeyPressed(rl.KeyUp) { //up
			paddleRight.Velocity.Y = -paddleLeft.Speed
		}
		if rl.IsKeyPressed(rl.KeyDown) { //down
			paddleRight.Velocity.Y = paddleLeft.Speed
		}
		// stop when released
		if rl.IsKeyUp(rl.KeyUp) && rl.IsKeyUp(rl.KeyDown) {
			paddleRight.Velocity.Y = 0
		}
	}

}

func PhysicsSystem(things Things, dt float32) ts.ThingRef {
	var ballRef ts.ThingRef
	for ref, thing := range things.Each() {
		switch thing.Kind {
		case KindBall:
			ballRef = ref
			ball := thing
			ball.Position = ball.Position.Add(
				ball.Velocity.Scale(dt),
			)
			// Collision of ball with top and bottom of screen
			if normal, collide := CircleRectCollisionNormal(ball.Position, ball.Height/2,
				rl.NewRectangle(0, 0, Width, 0)); collide {
				// collided with top.
				ball.Velocity = ball.Velocity.Reflect(normal)
			} else if normal, collide := CircleRectCollisionNormal(ball.Position, ball.Height/2,
				rl.NewRectangle(0, Height, Width, 0)); collide {
				// collided with top.
				ball.Velocity = ball.Velocity.Reflect(normal)
			}
			ball.Velocity = ball.Velocity.ClampValue(-ball.MaxSpeed, ball.MaxSpeed)
		case KindPaddle:
			paddle := thing
			paddle.Position = paddle.Position.Add(paddle.Velocity.Scale(dt))
			paddle.Position.Y = rl.Clamp(paddle.Position.Y, paddle.Height/2, Height-paddle.Height/2)
			paddle.Velocity = paddle.Velocity.Scale(1.010)
			paddle.Velocity.ClampValue(-paddle.MaxSpeed, paddle.MaxSpeed)
		}
	}
	return ballRef
}

// DrawThings loops over all the things and draws them.
func DrawThings(things Things) {
	for _, thing := range things.Each() {
		switch thing.Kind {
		case KindBall:
			rl.DrawCircleV(thing.Position, thing.Height/2, thing.Color)
		case KindPaddle:
			const fontSize = 10
			text := strconv.Itoa(thing.Score)
			textWidth := float32(rl.MeasureText(text, fontSize))
			rl.DrawText(text, int32(thing.ScoreTextPosition.X-textWidth/2), int32(thing.ScoreTextPosition.Y), 30, rl.ColorAlpha(thing.Color, 0.5))
			size := rl.NewVector2(thing.Width, thing.Height)
			topLeft := thing.Position.Subtract(size.Scale(.5))
			rl.DrawRectangleV(topLeft, size, thing.Color)
		}
	}
}

// draw a render texture to take up the whole screen and fractionally scale it.
func DrawRenderTextureScaled(surface rl.RenderTexture2D) {
	// get fractional scaling factor
	w := float32(rl.GetRenderWidth()) / float32(Width)
	h := float32(rl.GetRenderHeight()) / float32(Height)

	scale := min(w, h)

	scaledW := float32(surface.Texture.Width) * scale
	scaledH := float32(surface.Texture.Height) * scale
	pos := rl.NewVector2(
		(float32(rl.GetRenderWidth())-scaledW)/2,
		(float32(rl.GetRenderHeight())-scaledH)/2,
	)

	destRect := rl.NewRectangle(pos.X, pos.Y, scaledW, scaledH)
	rl.DrawTexturePro(
		surface.Texture,
		rl.NewRectangle(0, 0, float32(surface.Texture.Width), -float32(surface.Texture.Height)),
		destRect,
		rl.Vector2{},
		0,
		rl.White,
	)
	// outline
	rl.DrawRectangleLinesEx(destRect, 1, rl.ColorAlpha(rl.LightGray, 0.9))

}

// Gippity generated :)
func CircleRectCollisionNormal(circlePos rl.Vector2, radius float32, rect rl.Rectangle) (normal rl.Vector2, collided bool) {
	// Step 1: Find closest point on rectangle to circle
	closestX := rl.Clamp(circlePos.X, rect.X, rect.X+rect.Width)
	closestY := rl.Clamp(circlePos.Y, rect.Y, rect.Y+rect.Height)
	closest := rl.NewVector2(closestX, closestY)

	// Step 2: Vector from rectangle to circle
	normal.X = circlePos.X - closest.X
	normal.Y = circlePos.Y - closest.Y

	distSquared := normal.X*normal.X + normal.Y*normal.Y

	// Step 3: Check collision
	if distSquared > radius*radius {
		// No collision
		return rl.NewVector2(0, 0), false
	}

	if distSquared == 0 {
		// Circle center is inside rectangle; pick nearest face
		dxLeft := circlePos.X - rect.X
		dxRight := (rect.X + rect.Width) - circlePos.X
		dyTop := circlePos.Y - rect.Y
		dyBottom := (rect.Y + rect.Height) - circlePos.Y

		minDist := dxLeft
		normal = rl.NewVector2(-1, 0) // default to left

		if dxRight < minDist {
			minDist = dxRight
			normal = rl.NewVector2(1, 0)
		}
		if dyTop < minDist {
			minDist = dyTop
			normal = rl.NewVector2(0, -1)
		}
		if dyBottom < minDist {
			normal = rl.NewVector2(0, 1)
		}
	} else {
		// Step 5: Normalize normal vector
		length := float32(math.Sqrt(float64(distSquared)))
		normal.X /= length
		normal.Y /= length
	}
	return normal, true
}

// Also gippity generated :3
func DistanceToRect(circlePos rl.Vector2, rect rl.Rectangle) float32 {
	closestX := rl.Clamp(circlePos.X, rect.X, rect.X+rect.Width)
	closestY := rl.Clamp(circlePos.Y, rect.Y, rect.Y+rect.Height)

	dx := circlePos.X - closestX
	dy := circlePos.Y - closestY

	return float32(math.Sqrt(float64(dx*dx + dy*dy)))
}
