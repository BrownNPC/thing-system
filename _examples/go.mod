module github.com/BrownNPC/thing-system/examples

replace github.com/BrownNPC/thing-system => ../

go 1.26

require (
	github.com/BrownNPC/thing-system v0.0.0-00010101000000-000000000000
	github.com/gen2brain/raylib-go/raylib v0.56.0-dev.0.20260217065004-2c5f1b24d85e
	github.com/lmittmann/tint v1.1.3
)

require (
	github.com/ebitengine/purego v0.10.0 // indirect
	golang.org/x/exp v0.0.0-20260218203240-3dfff04db8fa // indirect
	golang.org/x/sys v0.41.0 // indirect
)

require github.com/BrownNPC/semicolons v1.0.0 // indirect

tool github.com/BrownNPC/semicolons
