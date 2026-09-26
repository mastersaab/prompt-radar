import { Component, Input } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ThresholdPoint } from '../../models/radar.models';

@Component({
  selector: 'app-threshold-curve',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './threshold-curve.component.html',
  styleUrls: ['./threshold-curve.component.css']
})
export class ThresholdCurveComponent {
  @Input() curveData: ThresholdPoint[] = [];
  @Input() currentThreshold: number = 0.88;

  get defaultCurve(): ThresholdPoint[] {
    return [
      { threshold: 0.70, hit_rate_pct: 92.0, projected_tokens_saved: 14000, projected_cost_saved_usd: 0.0063 },
      { threshold: 0.75, hit_rate_pct: 88.0, projected_tokens_saved: 13200, projected_cost_saved_usd: 0.0059 },
      { threshold: 0.80, hit_rate_pct: 80.0, projected_tokens_saved: 12000, projected_cost_saved_usd: 0.0054 },
      { threshold: 0.85, hit_rate_pct: 64.0, projected_tokens_saved: 9600, projected_cost_saved_usd: 0.0043 },
      { threshold: 0.88, hit_rate_pct: 52.0, projected_tokens_saved: 7800, projected_cost_saved_usd: 0.0035 },
      { threshold: 0.90, hit_rate_pct: 44.0, projected_tokens_saved: 6600, projected_cost_saved_usd: 0.0030 },
      { threshold: 0.95, hit_rate_pct: 24.0, projected_tokens_saved: 3600, projected_cost_saved_usd: 0.0016 },
      { threshold: 0.98, hit_rate_pct: 15.0, projected_tokens_saved: 2250, projected_cost_saved_usd: 0.0010 }
    ];
  }

  get points(): ThresholdPoint[] {
    return (this.curveData && this.curveData.length > 0) ? this.curveData : this.defaultCurve;
  }

  get chartPoints(): Array<{ x: number; y: number; threshold: number; hitRate: number }> {
    const pts = this.points;
    const minX = 55;
    const maxX = 510;
    const minY = 20;
    const maxY = 140;

    return pts.map((p, i) => {
      const x = minX + (i / (pts.length - 1)) * (maxX - minX);
      const y = maxY - (p.hit_rate_pct / 100) * (maxY - minY);
      return { x, y, threshold: p.threshold, hitRate: p.hit_rate_pct };
    });
  }

  get linePath(): string {
    const pts = this.chartPoints;
    if (pts.length === 0) return '';
    return pts.reduce((acc, p, i) => i === 0 ? `M ${p.x} ${p.y}` : `${acc} L ${p.x} ${p.y}`, '');
  }

  get areaPath(): string {
    const pts = this.chartPoints;
    if (pts.length === 0) return '';
    const line = this.linePath;
    const last = pts[pts.length - 1];
    const first = pts[0];
    return `${line} L ${last.x} 140 L ${first.x} 140 Z`;
  }

  get activeMarkerX(): number {
    const minX = 55;
    const maxX = 510;
    const tMin = 0.70;
    const tMax = 0.98;
    const ratio = Math.max(0, Math.min(1, (this.currentThreshold - tMin) / (tMax - tMin)));
    return minX + ratio * (maxX - minX);
  }

  get activeMarkerY(): number {
    const pts = this.points;
    let rate = 52.0;
    for (let i = 0; i < pts.length - 1; i++) {
      if (this.currentThreshold >= pts[i].threshold && this.currentThreshold <= pts[i + 1].threshold) {
        const segRatio = (this.currentThreshold - pts[i].threshold) / (pts[i + 1].threshold - pts[i].threshold);
        rate = pts[i].hit_rate_pct + segRatio * (pts[i + 1].hit_rate_pct - pts[i].hit_rate_pct);
        break;
      }
    }
    const minY = 20;
    const maxY = 140;
    return maxY - (rate / 100) * (maxY - minY);
  }
}
