# mipix

[![Go Reference](https://pkg.go.dev/badge/github.com/tinne26/mipix.svg)](https://pkg.go.dev/github.com/tinne26/mipix)

A package to assist the development of Ebitengine pixel art games.

This package allows you to implement your game working with logical canvases and ignoring `Game.Layout()` completely. Scaling is managed internally with pixel art aware scaling algorithms, and support for camera movement, zoom and screen shakes are also available through the API.

## Features

- Draw pixel art logically without having to deal with device scale factors, DPI, scaling and projections yourself.
- Basic camera with smooth tracking, zoom and screen shakes. Most behaviors can be customized through interfaces.
- Interleave high resolution and logical draws when necessary.

## How to use mipix

The best way to learn mipix is to read the [introductory tutorial with code samples](https://github.com/tinne26/mipix/tree/main/docs/introduction.md) and the [package documentation](https://pkg.go.dev/github.com/tinne26/mipix). More code samples are available on [tinne26/mipix-examples](https://github.com/tinne26/mipix-examples), but you should really spend a couple minutes to understand how the draw model works on mipix first.

If you are too lazy for that, at least read the summary:
- The canvas you receive on `Draw()` represents pixels 1-to-1. Your pixel art must be drawn directly to the canvas, in its original size. Just draw your pixel art.
- The canvas you receive on `Draw()` does *not* always have the same size, and it does *not* necessarily match the resolution you set for your game with `mipix.SetResolution(W, H)`. This can happen due to zoom effects, smooth camera movement and others. Just focus on rendering the logical area requested by `mipix.Camera().Area()`.

## Context

This package implements the second model described on [lopix](https://github.com/tinne26/lopix). If `lopix` implements the simplest model for pixel art games, `mipix` is slightly more advanced and provides a much more practical foundation to build pixel art games:
- You are expected to place most of your game world elements at integer coordinates.
- Your draw method receives a logically sized offscreen corresponding to a specific area of your game world. This area can vary based on the current camera position, zoom and shake, but you are simply given the task of *rendering a logical area of your game world*, directly and in pure pixels, to a logically sized canvas.
- For UI and other camera-independent parts of your game, you can create `mipix.Offscreen`s manually, again with pure pixel based sizes, render to them and then use the built-in `Offscreen.Project()` method to deal with scaling and stretching and filtering and all that nonsense.

While side scrollers can be implemented with this model, that's probably not ideal. In most side-scrollers, developers use floating point positions for characters, which are smoothly "sliding through the floor" as animations change[^1]. Doing this requires drawing the characters to a high resolution canvas. The API offers [basic support for this](https://pkg.go.dev/github.com/tinne26/mipix#AccessorHiRes.Draw), but it's not the main focus of the package. If you have *many* fractionally positioned characters and elements, the `mipix` model might not be the best match for your game.

[^1]: As opposed to this "sliding animation" model, pixel artists can design their animations to look good as they advance through the concrete pixel grid, with concrete pixel advances between frames. This isn't supported by most animation libraries, takes more work for the artist, takes more work for the developer, and makes animation usage more rigid (e.g., need separate "walking on stairs" animation)... but with this model, you can draw moving characters in the logical canvas. This is more common on RPGs and top down games than side scrollers, but it's fairly uncommon everywhere in general. Smoothness perception throughout movement is also of a different class.

## Random thoughts and comments

- Need to add camera area limits, blockers and so on.
- Motion blur should be considered for the high resolution draws and maybe even the main canvas. While most people say they hate motion blur, that mostly refers to motion blur as an aesthetic effect. Light amounts of motion blur could greatly reduce image stability problems during movement.
- The pixel art sampling shaders do cause a visible amount of "fattening" for the pixels.
