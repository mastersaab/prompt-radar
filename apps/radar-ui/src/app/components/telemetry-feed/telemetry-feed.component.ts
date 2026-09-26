import { Component, Input } from '@angular/core';
import { CommonModule } from '@angular/common';
import { TelemetryLogItem } from '../../models/radar.models';

@Component({
  selector: 'app-telemetry-feed',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './telemetry-feed.component.html',
  styleUrls: ['./telemetry-feed.component.css']
})
export class TelemetryFeedComponent {
  @Input() logs: TelemetryLogItem[] = [];
}
