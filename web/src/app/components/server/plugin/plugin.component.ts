import {Component, OnInit} from '@angular/core';
import {ClrTabsModule} from "@clr/angular";
import {RestService} from "../../../services/rest/rest.service";
import {MCServerDetail} from "../../../model/serverDetail";
import { Title } from '@angular/platform-browser';

@Component({
  selector: 'app-plugin',
  standalone: true,
  imports: [
    ClrTabsModule
  ],
  templateUrl: './plugin.component.html',
  styleUrl: './plugin.component.css'
})
export class PluginComponent implements OnInit {

  serverList: MCServerDetail[] = []


  constructor(private restService: RestService, private titleService: Title) {
  }

  ngOnInit() {
    this.titleService.setTitle("Server Plugins")
    this.restService.getAllServersWithDetails().subscribe({
      next: data => this.serverList = data,
      error: err => {
      }
    })

  }

}
