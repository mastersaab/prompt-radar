import {
  Component,
  ElementRef,
  Input,
  OnChanges,
  OnDestroy,
  OnInit,
  SimpleChanges,
  ViewChild,
  NgZone
} from '@angular/core';
import { CommonModule } from '@angular/common';
import { Coordinate, RadarNode } from '../../models/radar.models';

interface Shockwave {
  x: number;
  y: number;
  radius: number;
  maxRadius: number;
  color: string;
  alpha: number;
}

interface Particle {
  x: number;
  y: number;
  vx: number;
  vy: number;
  alpha: number;
  size: number;
}

@Component({
  selector: 'app-radar-canvas',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './radar-canvas.component.html',
  styleUrls: ['./radar-canvas.component.css']
})
export class RadarCanvasComponent implements OnInit, OnChanges, OnDestroy {
  @ViewChild('canvas', { static: true }) canvasRef!: ElementRef<HTMLCanvasElement>;
  @ViewChild('container', { static: true }) containerRef!: ElementRef<HTMLDivElement>;

  @Input() nodes: RadarNode[] = [];
  @Input() queryCoord: Coordinate | null = null;
  @Input() nearestCoord: Coordinate | null = null;
  @Input() threshold: number = 0.88;
  @Input() lastEvent: { type: 'hit' | 'miss_start' | 'miss_complete'; similarity?: number } | null = null;

  hoveredNode: RadarNode | null = null;
  tooltipX = 0;
  tooltipY = 0;

  private ctx!: CanvasRenderingContext2D;
  private animFrameId: number | null = null;
  private sweepAngle = 0;
  private shockwaves: Shockwave[] = [];
  private particles: Particle[] = [];
  private width = 600;
  private height = 540;
  private centerX = 300;
  private centerY = 270;
  private maxRadius = 240;

  // Active query smooth interpolation
  private currentQueryX: number | null = null;
  private currentQueryY: number | null = null;
  private targetQueryX: number | null = null;
  private targetQueryY: number | null = null;

  constructor(private ngZone: NgZone) {}

  ngOnInit() {
    this.initCanvas();
    this.initParticles();
    this.setupMouseEvents();

    // Run render loop outside Angular's zone for guaranteed 60 FPS
    this.ngZone.runOutsideAngular(() => {
      this.render();
    });
  }

  ngOnChanges(changes: SimpleChanges) {
    if (changes['queryCoord'] && this.queryCoord) {
      this.targetQueryX = this.queryCoord.x;
      this.targetQueryY = this.queryCoord.y;
      if (this.currentQueryX === null) {
        this.currentQueryX = this.targetQueryX;
        this.currentQueryY = this.targetQueryY;
      }
    }

    if (changes['lastEvent'] && this.lastEvent) {
      if (this.lastEvent.type === 'hit' && this.queryCoord) {
        this.addShockwave(this.queryCoord.x, this.queryCoord.y, '#10b981');
      } else if (this.lastEvent.type === 'miss_start' && this.queryCoord) {
        this.addShockwave(this.queryCoord.x, this.queryCoord.y, '#a855f7');
      }
    }
  }

  ngOnDestroy() {
    if (this.animFrameId) {
      cancelAnimationFrame(this.animFrameId);
    }
  }

  private initCanvas() {
    const canvas = this.canvasRef.nativeElement;
    const container = this.containerRef.nativeElement;
    this.width = container.clientWidth || 600;
    this.height = container.clientHeight || 540;

    const dpr = window.devicePixelRatio || 1;
    canvas.width = this.width * dpr;
    canvas.height = this.height * dpr;

    this.ctx = canvas.getContext('2d')!;
    this.ctx.scale(dpr, dpr);

    this.centerX = this.width / 2;
    this.centerY = this.height / 2;
    this.maxRadius = Math.min(this.centerX, this.centerY) * 0.86;

    window.addEventListener('resize', () => {
      if (!this.containerRef) return;
      this.width = this.containerRef.nativeElement.clientWidth;
      this.height = this.containerRef.nativeElement.clientHeight;
      canvas.width = this.width * (window.devicePixelRatio || 1);
      canvas.height = this.height * (window.devicePixelRatio || 1);
      this.ctx = canvas.getContext('2d')!;
      this.ctx.scale(window.devicePixelRatio || 1, window.devicePixelRatio || 1);
      this.centerX = this.width / 2;
      this.centerY = this.height / 2;
      this.maxRadius = Math.min(this.centerX, this.centerY) * 0.86;
    });
  }

  private initParticles() {
    this.particles = [];
    for (let i = 0; i < 40; i++) {
      this.particles.push({
        x: Math.random() * this.width,
        y: Math.random() * this.height,
        vx: (Math.random() - 0.5) * 0.25,
        vy: (Math.random() - 0.5) * 0.25,
        alpha: Math.random() * 0.35 + 0.1,
        size: Math.random() * 1.5 + 0.5
      });
    }
  }

  private addShockwave(xNorm: number, yNorm: number, color: string) {
    const screenX = this.coordToScreenX(xNorm);
    const screenY = this.coordToScreenY(yNorm);
    this.shockwaves.push({
      x: screenX,
      y: screenY,
      radius: 6,
      maxRadius: 200,
      color,
      alpha: 1.0
    });
  }

  private render = () => {
    this.ctx.clearRect(0, 0, this.width, this.height);

    // 1. Polar Grid & Range Rings
    this.drawRadarGrid();

    // 2. Space Particles
    this.drawParticles();

    // 3. Radar Sweep Beam
    this.drawSweepBeam();

    // 4. Cached Vector Nodes
    this.drawCachedNodes();

    // 5. Active Query Marker & Hit-Radius Boundary
    this.drawActiveQuery();

    // 6. Shockwave Ripples
    this.drawShockwaves();

    this.sweepAngle = (this.sweepAngle + 0.018) % (Math.PI * 2);
    this.animFrameId = requestAnimationFrame(this.render);
  };

  private drawRadarGrid() {
    const ctx = this.ctx;

    // Crosshair axes
    ctx.strokeStyle = 'rgba(0, 245, 212, 0.14)';
    ctx.lineWidth = 1;
    ctx.beginPath();
    ctx.moveTo(this.centerX, 20);
    ctx.lineTo(this.centerX, this.height - 20);
    ctx.moveTo(20, this.centerY);
    ctx.lineTo(this.width - 20, this.centerY);
    ctx.stroke();

    // Concentric polar rings (0.25, 0.50, 0.75, 1.00)
    const rings = [0.25, 0.5, 0.75, 1.0];
    rings.forEach((r, idx) => {
      const radius = this.maxRadius * r;
      ctx.beginPath();
      ctx.arc(this.centerX, this.centerY, radius, 0, Math.PI * 2);
      ctx.strokeStyle = idx === rings.length - 1 ? 'rgba(0, 245, 212, 0.35)' : 'rgba(0, 245, 212, 0.12)';
      ctx.setLineDash(idx === rings.length - 1 ? [] : [4, 4]);
      ctx.stroke();
      ctx.setLineDash([]);

      ctx.fillStyle = 'rgba(0, 245, 212, 0.4)';
      ctx.font = '10px "JetBrains Mono"';
      ctx.fillText(`r=${r.toFixed(2)}`, this.centerX + 6, this.centerY - radius + 12);
    });

    // Outer glow ring
    ctx.beginPath();
    ctx.arc(this.centerX, this.centerY, this.maxRadius, 0, Math.PI * 2);
    ctx.strokeStyle = 'rgba(0, 245, 212, 0.3)';
    ctx.lineWidth = 2;
    ctx.stroke();
  }

  private drawParticles() {
    const ctx = this.ctx;
    ctx.fillStyle = '#00f5d4';

    for (const p of this.particles) {
      p.x += p.vx;
      p.y += p.vy;

      if (p.x < 0) p.x = this.width;
      if (p.x > this.width) p.x = 0;
      if (p.y < 0) p.y = this.height;
      if (p.y > this.height) p.y = 0;

      ctx.globalAlpha = p.alpha;
      ctx.beginPath();
      ctx.arc(p.x, p.y, p.size, 0, Math.PI * 2);
      ctx.fill();
    }
    ctx.globalAlpha = 1.0;
  }

  private drawSweepBeam() {
    const ctx = this.ctx;
    const beamAngle = this.sweepAngle;
    const trailSpan = Math.PI / 3;

    ctx.save();
    ctx.beginPath();
    ctx.moveTo(this.centerX, this.centerY);
    ctx.arc(this.centerX, this.centerY, this.maxRadius, beamAngle - trailSpan, beamAngle, false);
    ctx.closePath();

    const gradient = ctx.createRadialGradient(
      this.centerX, this.centerY, 0,
      this.centerX, this.centerY, this.maxRadius
    );
    gradient.addColorStop(0, 'rgba(0, 245, 212, 0.18)');
    gradient.addColorStop(1, 'rgba(0, 245, 212, 0.0)');
    ctx.fillStyle = gradient;
    ctx.fill();

    ctx.beginPath();
    ctx.moveTo(this.centerX, this.centerY);
    ctx.lineTo(
      this.centerX + Math.cos(beamAngle) * this.maxRadius,
      this.centerY + Math.sin(beamAngle) * this.maxRadius
    );
    ctx.strokeStyle = 'rgba(0, 245, 212, 0.65)';
    ctx.lineWidth = 1.5;
    ctx.stroke();
    ctx.restore();
  }

  private drawCachedNodes() {
    const ctx = this.ctx;

    for (const node of this.nodes) {
      const screenX = this.coordToScreenX(node.x);
      const screenY = this.coordToScreenY(node.y);
      const color = this.getCategoryColor(node.category);

      // Outer glow
      ctx.beginPath();
      ctx.arc(screenX, screenY, 9, 0, Math.PI * 2);
      ctx.fillStyle = color;
      ctx.globalAlpha = 0.3;
      ctx.fill();

      // Core
      ctx.beginPath();
      ctx.arc(screenX, screenY, 4.5, 0, Math.PI * 2);
      ctx.fillStyle = color;
      ctx.globalAlpha = 0.95;
      ctx.fill();

      // White outline ring
      ctx.strokeStyle = '#ffffff';
      ctx.lineWidth = 1;
      ctx.globalAlpha = 0.8;
      ctx.stroke();
    }
    ctx.globalAlpha = 1.0;
  }

  private drawActiveQuery() {
    if (this.targetQueryX === null || this.targetQueryY === null) return;

    if (this.currentQueryX !== null && this.currentQueryY !== null) {
      this.currentQueryX += (this.targetQueryX - this.currentQueryX) * 0.12;
      this.currentQueryY += (this.targetQueryY - this.currentQueryY) * 0.12;
    } else {
      this.currentQueryX = this.targetQueryX;
      this.currentQueryY = this.targetQueryY;
    }

    const qX = this.coordToScreenX(this.currentQueryX);
    const qY = this.coordToScreenY(this.currentQueryY);
    const ctx = this.ctx;

    // Dynamic Hit Radius Sensitivity Circle
    const radiusScale = (1.0 - (this.threshold - 0.70) / (0.98 - 0.70)) * 65 + 18;

    ctx.beginPath();
    ctx.arc(qX, qY, radiusScale, 0, Math.PI * 2);
    ctx.strokeStyle = 'rgba(0, 245, 212, 0.7)';
    ctx.lineWidth = 1.5;
    ctx.setLineDash([4, 4]);
    ctx.stroke();
    ctx.setLineDash([]);

    ctx.fillStyle = 'rgba(0, 245, 212, 0.08)';
    ctx.fill();

    // Connector line to nearest node
    if (this.nearestCoord) {
      const nX = this.coordToScreenX(this.nearestCoord.x);
      const nY = this.coordToScreenY(this.nearestCoord.y);

      ctx.beginPath();
      ctx.moveTo(qX, qY);
      ctx.lineTo(nX, nY);
      ctx.strokeStyle = this.lastEvent?.type === 'hit' ? 'rgba(16, 185, 129, 0.8)' : 'rgba(168, 85, 247, 0.6)';
      ctx.lineWidth = 2;
      ctx.setLineDash([3, 3]);
      ctx.stroke();
      ctx.setLineDash([]);
    }

    // Active Pin
    ctx.beginPath();
    ctx.arc(qX, qY, 13, 0, Math.PI * 2);
    ctx.fillStyle = 'rgba(0, 245, 212, 0.35)';
    ctx.fill();

    ctx.beginPath();
    ctx.arc(qX, qY, 6, 0, Math.PI * 2);
    ctx.fillStyle = '#00f5d4';
    ctx.fill();
    ctx.strokeStyle = '#ffffff';
    ctx.lineWidth = 2;
    ctx.stroke();
  }

  private drawShockwaves() {
    const ctx = this.ctx;

    for (let i = this.shockwaves.length - 1; i >= 0; i--) {
      const sw = this.shockwaves[i];
      sw.radius += 4;
      sw.alpha = 1.0 - sw.radius / sw.maxRadius;

      if (sw.alpha <= 0) {
        this.shockwaves.splice(i, 1);
        continue;
      }

      ctx.beginPath();
      ctx.arc(sw.x, sw.y, sw.radius, 0, Math.PI * 2);
      ctx.strokeStyle = sw.color;
      ctx.globalAlpha = sw.alpha * 0.8;
      ctx.lineWidth = 2.5;
      ctx.stroke();

      ctx.beginPath();
      ctx.arc(sw.x, sw.y, sw.radius * 0.7, 0, Math.PI * 2);
      ctx.strokeStyle = sw.color;
      ctx.globalAlpha = sw.alpha * 0.4;
      ctx.lineWidth = 1;
      ctx.stroke();
    }
    ctx.globalAlpha = 1.0;
  }

  private coordToScreenX(normX: number): number {
    return this.centerX + normX * (this.maxRadius * 85);
  }

  private coordToScreenY(normY: number): number {
    return this.centerY + normY * (this.maxRadius * 85);
  }

  public getCategoryColor(category: string): string {
    switch (category?.toLowerCase()) {
      case 'golang': return '#10b981';
      case 'python': return '#3b82f6';
      case 'machine learning': return '#f59e0b';
      case 'system design': return '#06b6d4';
      case 'culinary': return '#ec4899';
      case 'physics': return '#8b5cf6';
      default: return '#00f5d4';
    }
  }

  private setupMouseEvents() {
    const canvas = this.canvasRef.nativeElement;

    canvas.addEventListener('mousemove', (e) => {
      const rect = canvas.getBoundingClientRect();
      const mouseX = e.clientX - rect.left;
      const mouseY = e.clientY - rect.top;

      let found: RadarNode | null = null;
      let minDistance = 16;

      for (const node of this.nodes) {
        const sx = this.coordToScreenX(node.x);
        const sy = this.coordToScreenY(node.y);
        const dist = Math.hypot(mouseX - sx, mouseY - sy);

        if (dist < minDistance) {
          minDistance = dist;
          found = node;
        }
      }

      this.ngZone.run(() => {
        this.hoveredNode = found;
        if (found) {
          this.tooltipX = mouseX;
          this.tooltipY = mouseY;
        }
      });
    });

    canvas.addEventListener('mouseleave', () => {
      this.ngZone.run(() => {
        this.hoveredNode = null;
      });
    });
  }
}
