import {Component} from '@angular/core';
import {CommonModule, NgOptimizedImage} from '@angular/common';
import {SidenavComponent} from "./components/sidenav/sidenav.component";
import {ClarityModule} from "@clr/angular";

@Component({
  selector: 'app-root',
  standalone: true,
  imports: [CommonModule, SidenavComponent, ClarityModule],
  templateUrl: './app.component.html',
  styleUrl: './app.component.css'
})
export class AppComponent {
  title = 'web';
}
