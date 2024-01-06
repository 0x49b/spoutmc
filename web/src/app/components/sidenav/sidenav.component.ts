import {Component} from '@angular/core';
import {RouterLink, RouterLinkActive, RouterOutlet} from "@angular/router";
import {NgOptimizedImage} from "@angular/common";
import {ClrDropdownModule, ClrVerticalNavModule} from "@clr/angular";
import {NgIcon} from "@ng-icons/core";
import {NgHeroiconsModule} from "@dimaslz/ng-heroicons";

@Component({
  selector: 'app-sidenav',
  standalone: true,
  imports: [
    RouterOutlet,
    RouterLink,
    NgOptimizedImage,
    ClrVerticalNavModule,
    ClrDropdownModule,
    NgIcon,
    NgHeroiconsModule,
    RouterLinkActive
  ],
  templateUrl: './sidenav.component.html',
  styleUrl: './sidenav.component.css'
})
export class SidenavComponent {
  demoCollapsible = false
}
