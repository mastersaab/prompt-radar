import { Component, EventEmitter, Input, Output } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';

interface PresetItem {
  tag: string;
  text: string;
  hint: string;
  isHit: boolean;
}

@Component({
  selector: 'app-prompt-input',
  standalone: true,
  imports: [CommonModule, FormsModule],
  templateUrl: './prompt-input.component.html',
  styleUrls: ['./prompt-input.component.css']
})
export class PromptInputComponent {
  @Input() threshold: number = 0.88;
  @Input() isStreaming: boolean = false;

  @Output() querySubmit = new EventEmitter<{ prompt: string; threshold: number }>();
  @Output() thresholdChange = new EventEmitter<number>();

  promptText: string = 'Golang sort slice example';

  presets: PresetItem[] = [
    { tag: 'HIT ⚡', text: 'Golang sort slice example', hint: 'Matches cached Go slice sorting prompt (~99.6% similarity)', isHit: true },
    { tag: 'HIT ⚡', text: 'How to invert a binary tree in Python', hint: 'Matches Python binary tree algorithm (~99% similarity)', isHit: true },
    { tag: 'HIT ⚡', text: 'Best brewing ratio for French press coffee', hint: 'Matches French press coffee ratio (~99% similarity)', isHit: true },
    { tag: 'MISS 🌐', text: 'How does WebAssembly work in browser engines?', hint: 'Novel technical prompt triggers LLM streaming & auto-indexing', isHit: false },
    { tag: 'MISS 🌐', text: 'Explain the Drake Equation in astrobiology', hint: 'Novel physics/astronomy prompt triggers LLM streaming', isHit: false }
  ];

  onSubmit() {
    if (!this.promptText.trim() || this.isStreaming) return;
    this.querySubmit.emit({
      prompt: this.promptText.trim(),
      threshold: this.threshold
    });
  }

  onThresholdChange(val: number) {
    this.threshold = parseFloat(val as any);
    this.thresholdChange.emit(this.threshold);
  }

  selectPreset(text: string) {
    this.promptText = text;
    this.onSubmit();
  }
}
