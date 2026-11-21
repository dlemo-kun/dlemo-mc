# dlemo-mc

## What is `dlemo-mc`?

**`dlemo-mc`** is an independent implementation of the **Minecraft (Java Edition)** client built from scratch using the **Go (Golang)** programming language.

The project aims to explore the game's internal architecture, performance, and graphical capabilities using modern technologies.

### Key Features

* **Written in Go:** Leverages Go's simplicity, concurrency, and speed to handle client logic and networking.
* **Rendered with WGPU:** Uses **WGPU** (the Rust implementation of the WebGPU API, adapted via Go bindings) to achieve hardware-accelerated, modern, cross-platform graphics performance (Vulkan, DirectX 12, or OpenGL depending on the OS).
* **Current Development Status:** Currently in its earliest initialization phase. Running the project displays a title menu with several buttons. Most of the buttons do nothing, but the **Singleplayer** and **Multiplayer** buttons attempt to connect to a local server on port **25565** using protocol **777**, while the **Exit** button closes the application.

### Third-Party Assets & Licenses

* **UI Icons (`assets/icons/ui/`):** Based on **Material Design Icons (v4.0.0)** by Google ([`google/material-design-icons`](https://github.com/google/material-design-icons)), licensed under the **Apache License, Version 2.0** (`http://www.apache.org/licenses/LICENSE-2.0`).
  * **Modifications:** The original icons were modified (colors were changed to match the client UI theme).
  * A copy of the Apache License 2.0 can be found in [`assets/icons/ui/LICENSE`](assets/icons/ui/LICENSE).