import {Component} from '@angular/core';
import {MatSidenavModule} from "@angular/material/sidenav";
import {MatToolbarModule} from "@angular/material/toolbar";
import {MatIconModule} from "@angular/material/icon";
import {MatButtonModule} from "@angular/material/button";
import {RouterLink, RouterOutlet} from "@angular/router";
import {MatListModule} from "@angular/material/list";
import {BooleanInput} from "@angular/cdk/coercion";
import {NgOptimizedImage} from "@angular/common";
import {MatMenuModule} from "@angular/material/menu";

@Component({
  selector: 'app-sidenav',
  standalone: true,
  imports: [
    MatSidenavModule,
    MatToolbarModule,
    MatIconModule,
    MatButtonModule,
    RouterOutlet,
    MatListModule,
    RouterLink,
    NgOptimizedImage,
    MatMenuModule
  ],
  templateUrl: './sidenav.component.html',
  styleUrl: './sidenav.component.css'
})
export class SidenavComponent {}
