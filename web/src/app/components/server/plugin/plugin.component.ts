import {Component, OnInit} from '@angular/core';
import {ClrTabsModule, ClrTreeViewModule} from "@clr/angular";
import {RestService} from "../../../services/rest/rest.service";
import {Title} from '@angular/platform-browser';
import {PluginServerList} from "../../../model/plugins";
import {OutlineIconsModule} from "@dimaslz/ng-heroicons";
import {FiletreeComponent} from "../../util/filetree/filetree.component";
import {getChildren} from "@cds/core/internal";

@Component({
  selector: 'app-plugin',
  standalone: true,
  imports: [
    ClrTabsModule,
    ClrTreeViewModule,
    OutlineIconsModule,
    FiletreeComponent
  ],
  templateUrl: './plugin.component.html',
  styleUrl: './plugin.component.css'
})
export class PluginComponent implements OnInit {

  pluginServerLists: PluginServerList[] = []

  constructor(private restService: RestService, private titleService: Title) {
  }


  ngOnInit() {
    this.titleService.setTitle("Server Plugins")

    this.restService.getPlugins().subscribe({
      next: data => {
        this.pluginServerLists = data
      },
      error: err => {
      }
    })


  }
}
