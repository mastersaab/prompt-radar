# Radar Canvas & Frontend Visualization Architecture

The Radar Visualizer (`apps/radar-ui/src/app/components/radar-canvas`) renders high-dimensional semantic vector state onto a 2D Cartesian radar display at **60 frames per second**.

```
                Angular Signals & State Stream
                             │
                             ▼
              RadarCanvasComponent (@Input)
                             │
            ┌────────────────┴────────────────┐
            ▼                                 ▼
   Zone.js Bypass Loop              Device Pixel Ratio Scaler
 (runOutsideAngular)                (High-DPI Retina Buffer)
            │                                 │
            └────────────────┬────────────────┘
                             │
                             ▼
                    HTML5 Canvas 2D Context
                             │
    ┌───────────┬────────────┼────────────┬───────────┐
    ▼           ▼            ▼            ▼           ▼
Polar Grid   Particle     Sweep Beam   Cached Nodes Active Query
& Crosshair  Field        (Gradient)   (Clusters)   & Hit Radius
                                                      │
                                                      ▼
                                              Shockwave Ripples
                                              (Hit: Emerald #10b981)
                                              (Miss: Purple #a855f7)
```

---

## 1. Visual Legend & State Representation

| Visual Element | Appearance | Semantic Meaning |
|---|---|---|
| **Green Dot / Shockwave** | Emerald (`#10b981`) | **Cache Hit**: Query similarity $\ge$ threshold $\tau$. Fast return from memory. |
| **Purple Dot / Shockwave** | Neon Purple (`#a855f7`) | **Cache Miss**: Query similarity $< \tau$. Invoking upstream LLM. |
| **Pulsing Circle Ring** | Dashed Cyan/Green Halo | **Similarity Threshold ($\tau$)**: Visual radius around nearest node. If query falls inside, it registers as a hit. |
| **Cyan Points** | Glowing Dots (`#00f5d4`) | **Cached Prompts**: Historical vector embeddings stored in memory. |
| **Gold Outline** | Amber Ring (`#f59e0b`) | **High-Frequency Node**: Prompts with high hit counts ($>5$ hits) scale in size. |
| **Connecting Line** | Gradient Laser Line | **Nearest Neighbor Vector**: Shows vector proximity to closest candidate prompt. |
| **Rotating Beam** | 45° Translucent Sector | **Radar Sweep**: Ambient visual scan indicating real-time proxy liveness. |

---

## 2. Canvas vs DOM/SVG Architecture

### Why HTML5 Canvas?
1. **DOM Thrashing**: Rendering 500+ vector nodes, 40 space particles, concentric polar grids, and dynamic shockwave ripples using SVG `<circle>` or DOM elements requires modifying hundreds of DOM tree nodes per frame. This triggers continuous browser style recalculations and layout reflows, dropping framerates below 30 FPS.
2. **Immediate Mode Rendering**: Canvas 2D issues direct drawing commands to the GPU backing store without retaining an internal scene graph, maintaining a flat memory footprint and steady 60 FPS.
3. **Sub-Pixel Anti-Aliasing**: High-DPI screens render micro-geometry and subtle glowing halos crisply without CSS filter lag.

---

## 3. High-DPI Screen Scaling (Retina / 4K)

To prevent blurred rendering on Retina and 4K mobile displays, the canvas internal resolution is scaled by the device pixel ratio (`window.devicePixelRatio`):

```typescript
const dpr = window.devicePixelRatio || 1;
canvas.width = this.width * dpr;
canvas.height = this.height * dpr;

this.ctx = canvas.getContext('2d')!;
this.ctx.scale(dpr, dpr);
```

All subsequent drawing operations use logical coordinate units while the hardware renders at native physical pixel density.

---

## 4. Coordinate Transformation Pipeline

The Go data plane projects 768-D vectors into normalized Cartesian space $(x, y) \in [-1.0, 1.0]$. The canvas transforms these into screen pixel coordinates $(X_{\text{screen}}, Y_{\text{screen}})$:

```typescript
// Canvas Y-axis is inverted relative to standard Cartesian coordinates
coordToScreenX(xNorm: number): number {
    return this.centerX + xNorm * this.maxRadius;
}

coordToScreenY(yNorm: number): number {
    return this.centerY - yNorm * this.maxRadius;
}
```

```
   (-1.0, 1.0)               (1.0, 1.0)
         ┌──────────────────────┐
         │          ▲ +Y        │
         │          │           │
         │   -X ────┼────► +X   │
         │          │           │
         │          ▼ -Y        │
         └──────────────────────┘
   (-1.0, -1.0)              (1.0, -1.0)
```

---

## 5. 60 FPS Performance: Angular Zone.js Bypass

Angular's default change detection monkey-patches `requestAnimationFrame`. If the render loop runs inside the Angular zone, every single animation tick (60 times per second) triggers an application-wide change detection pass across all components, degrading UI responsiveness.

PromptRadar explicitly isolates the render loop using `NgZone.runOutsideAngular()`:

```typescript
ngOnInit() {
    this.initCanvas();
    this.initParticles();
    this.setupMouseEvents();

    // Run render loop outside Angular Zone to bypass change detection
    this.ngZone.runOutsideAngular(() => {
        this.render();
    });
}
```

Angular change detection is only triggered when state explicitly changes via `@Input()` updates or RxJS/Signal subscriptions.

---

## 6. Dynamic Animation & Shockwave Mechanics

### Shockwave Wavefront Propagation
When a query arrives, a shockwave ring is spawned at the query coordinates:

```typescript
interface Shockwave {
    x: number;
    y: number;
    radius: number;
    maxRadius: number;
    color: string;
    alpha: number;
}
```

On every animation frame:
1. `radius` expands linearly: $r_{t+1} = r_t + 3.5\text{px}$.
2. `alpha` decays smoothly: $\alpha_{t+1} = 1.0 - (r_{t+1} / r_{\text{max}})$.
3. The shockwave is purged from memory once $\alpha \le 0$.

### Threshold Halo Radius
The circular boundary indicating similarity tolerance $\tau$ around the nearest candidate node is rendered with a radius proportional to angular tolerance:

$$R_{\text{halo}} = R_{\text{max}} \times (1.0 - \tau) \times 1.25$$

If the incoming query dot falls inside this halo, the prompt lies within the semantic boundary and triggers a hit.

---

## 7. Interactive Hover & Tooltips

The canvas registers native `mousemove` listeners outside Angular. It calculates the Euclidean distance to every cached node in screen space:

$$d = \sqrt{(X_{\text{mouse}} - X_{\text{node}})^2 + (Y_{\text{mouse}} - Y_{\text{node}})^2}$$

If $d < 12\text{px}$, the hovered node is selected and Angular's zone is re-entered (`this.ngZone.run(...)`) only when the tooltip visibility actually toggles, displaying:
* Prompt preview text
* Category badge
* Total cache hit count
* Creation timestamp
