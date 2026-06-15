# Soft Engine

Soft Engine is a simple 3D engine written in Go that builds upon SoftGO's software renderer and provides a higher-level API for creating interactive terminal-based 3D scenes.

The engine adds scene management, lighting, terrain generation, skyboxes, and positional audio while retaining compatibility with X11-based terminal rendering.

## Requirements

Soft Engine requires a GNU/Linux system with X11 support. Wayland sessions are also usable if XWayland-compatible terminals are employed.

You need Go 1.26.3 or newer to build and run the project.

The project depends on [SoftGO](https://github.com/striter-no/softgo) for rendering and uses a small set of libraries for mathematics, 3D operations, audio playback, and X11 interaction.

## Features

### Rendering

- Built on top of SoftGO
- Perspective camera system
- Vertex and fragment shader pipeline
- Texture mapping
- Animated GIF textures
- Multiple objects per scene

### Scene System

- Entity-based scene management
- Runtime object transformations
- Camera controls
- Skybox support
- Procedural terrain generation
- Level of Detail (LOD) generation

### Lighting

- Ambient lighting
- Directional lighting
- Shader-based lighting calculations

### Assets

Supported asset types:

- Wavefront .obj meshes
- .jpg textures
- .png textures
- Animated .gif textures
- .mp3 audio files

### Audio
- CGO-based audio system (Miniaudio C backend)
- 3D positional audio
- Listener positioning
- Runtime sound playback
- Miniaudio backend

### Input

- X11 keyboard input
- X11 mouse input
- XWayland compatibility

## Running

Run the included example scene:
Firstly install `libXfixes-devel` for HID on X11, then:

```sh
go run ./examples/main.go
```

or
```sh
go run ./dreamcore/main.go
```

Press `Esc` to exit.

## Running on Wayland

Use an X11-compatible terminal emulator such as Alacritty:

```
env WAYLAND_DISPLAY= alacritty
```

Then run the example scene 

## Project Structure
- `api/` core engine interface
- `entity/` Scene objects and terrain generation
- `lights/` Lighting system
- `sounds/` Positional audio
- `assets/` Example assets
- `examples/` Example scenes
- `dreamcore/` Additional demo application
- `cgo/` Audio backend bindings

## License

Licensed under the GNU General Public License v3.0.
See the LICENSE file for details.