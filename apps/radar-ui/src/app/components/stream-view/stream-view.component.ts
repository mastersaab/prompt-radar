import { Component, Input, OnChanges, SimpleChanges } from '@angular/core';
import { CommonModule } from '@angular/common';
import { DomSanitizer, SafeHtml } from '@angular/platform-browser';
import katex from 'katex';
import { marked } from 'marked';
import { QueryResult } from '../../models/radar.models';

@Component({
  selector: 'app-stream-view',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './stream-view.component.html',
  styleUrls: ['./stream-view.component.css']
})
export class StreamViewComponent implements OnChanges {
  @Input() result: QueryResult | null = null;
  @Input() streamText: string = '';

  renderedHtml: SafeHtml = '';

  constructor(private sanitizer: DomSanitizer) {}

  ngOnChanges(changes: SimpleChanges): void {
    const raw = this.streamText || this.result?.response || '';
    this.renderedHtml = this.formatContent(raw);
  }

  private formatContent(rawText: string): SafeHtml {
    if (!rawText) return '';

    marked.setOptions({
      gfm: true,
      breaks: true
    });

    // 1. Process Block Math: $$ ... $$
    let processed = rawText.replace(/\$\$([\s\S]*?)\$\$/g, (match, formula) => {
      try {
        const rendered = katex.renderToString(formula.trim(), {
          displayMode: true,
          throwOnError: false
        });
        return `\n\n<div class="math-block">${rendered}</div>\n\n`;
      } catch (e) {
        return match;
      }
    });

    // 2. Process Inline Math: $ ... $
    processed = processed.replace(/(?<!\\)\$([^\$\n\r]+?)\$/g, (match, formula) => {
      const trimmed = formula.trim();
      if (/^\d+(\.\d+)?$/.test(trimmed)) {
        return match;
      }
      try {
        return katex.renderToString(trimmed, {
          displayMode: false,
          throwOnError: false
        });
      } catch (e) {
        return match;
      }
    });

    // 3. Render Markdown into HTML
    try {
      const parsedHtml = marked.parse(processed) as string;
      return this.sanitizer.bypassSecurityTrustHtml(parsedHtml);
    } catch (e) {
      return this.sanitizer.bypassSecurityTrustHtml(processed);
    }
  }
}
