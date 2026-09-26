import { Component, Input } from '@angular/core';
import { CommonModule } from '@angular/common';
import { AnalyticsSummary } from '../../models/radar.models';

@Component({
  selector: 'app-metrics-bar',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './metrics-bar.component.html',
  styleUrls: ['./metrics-bar.component.css']
})
export class MetricsBarComponent {
  @Input() summary: AnalyticsSummary | null = null;
  @Input() cachedVectorsCount: number = 0;
}
